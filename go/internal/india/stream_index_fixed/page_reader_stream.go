package stream_index_fixed

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

type streamPageReader struct {
	*page
	blobReader      domain_interfaces.BlobReader
	bufferedReader  *bufio.Reader
	namedBlobAccess domain_interfaces.NamedBlobAccess

	overflowBlobReader domain_interfaces.BlobReader
	overflowReaderAt   *overflowReaderAt
}

func (index *Index) makeStreamPageReader(
	pageIndex PageIndex,
) (streamPageReader, errors.FuncErr) {
	pageReader := streamPageReader{
		page:            &index.pages[pageIndex],
		namedBlobAccess: index.envRepo,
	}

	var err error

	if pageReader.blobReader, err = env_repo.MakeNamedBlobReaderOrNullReader(
		pageReader.namedBlobAccess,
		pageReader.pageId.Path(),
	); err != nil {
		panic(err)
	}

	var repool interfaces.FuncRepool

	pageReader.bufferedReader, repool = pool.GetBufferedReader(
		pageReader.blobReader,
	)

	// Open overflow sidecar if it exists.
	ovId := overflowPageId(pageReader.pageId)
	overflowPath := ovId.Path()
	overflowReader, overflowErr := env_repo.MakeNamedBlobReaderOrNullReader(
		pageReader.namedBlobAccess,
		overflowPath,
	)

	if overflowErr != nil {
		panic(overflowErr)
	}

	pageReader.overflowBlobReader = overflowReader
	pageReader.overflowReaderAt = &overflowReaderAt{readerAt: overflowReader}

	return pageReader, func() error {
		repool()
		overflowReader.Close()
		return pageReader.blobReader.Close()
	}
}

func makeSeqObjectWithCursorAndSigilFromReader(
	reader io.Reader,
	queryGroup sku.PrimitiveQueryGroup,
	overflowRA *overflowReaderAt,
) interfaces.SeqError[objectWithCursorAndSigil] {
	return func(yield func(objectWithCursorAndSigil, error) bool) {
		decoder := makeBinaryDecoderWithQueryGroup(queryGroup, ids.SigilHistory)

		var object objectWithCursorAndSigil
		object.Transacted, _ = sku.GetTransactedPool().GetWithRepool()

		for {
			sku.TransactedResetter.Reset(object.Transacted)
			object.Offset += object.ContentLength

			var err error

			if _, err = decoder.readFixedEntryMatchSigil(
				reader,
				overflowRA,
				&object,
			); err != nil {
				yield(object, errors.WrapExceptSentinelAsNil(err, io.EOF))
				return
			}

			object.ContentLength = int64(EntryWidth)

			if !yield(object, nil) {
				return
			}
		}
	}
}

func makeSeqObjectFromReader(
	reader io.Reader,
	queryGroup sku.PrimitiveQueryGroup,
	overflowRA *overflowReaderAt,
) interfaces.SeqError[*sku.Transacted] {
	return func(yield func(*sku.Transacted, error) bool) {
		seq := makeSeqObjectWithCursorAndSigilFromReader(
			reader,
			queryGroup,
			overflowRA,
		)
		for objectPlus, err := range seq {
			if err != nil {
				yield(nil, errors.Wrap(err))
				return
			}

			if !yield(objectPlus.Transacted, nil) {
				return
			}
		}
	}
}

func (pageReader *streamPageReader) readFrom(
	reader io.Reader,
	queryGroup sku.PrimitiveQueryGroup,
) interfaces.SeqError[objectWithCursorAndSigil] {
	return func(yield func(objectWithCursorAndSigil, error) bool) {
		decoder := makeBinaryDecoderWithQueryGroup(queryGroup, ids.SigilHistory)

		var object objectWithCursorAndSigil
		object.Transacted, _ = sku.GetTransactedPool().GetWithRepool()

		for {
			sku.TransactedResetter.Reset(object.Transacted)
			object.Offset += object.ContentLength

			var err error

			if _, err = decoder.readFixedEntryMatchSigil(
				reader,
				pageReader.overflowReaderAt,
				&object,
			); err != nil {
				if errors.IsEOF(err) {
					return
				}

				object.Transacted = nil
				yield(object, errors.Wrap(err))
				return
			}

			object.ContentLength = int64(EntryWidth)

			if !yield(object, nil) {
				return
			}
		}
	}
}

type pageReadOptions struct {
	includeAddedHistory bool
	includeAddedLatest  bool
}

func (pageReader *streamPageReader) makeSeq(
	query sku.PrimitiveQueryGroup,
	pageReadOptions pageReadOptions,
) interfaces.SeqError[*sku.Transacted] {
	if !pageReadOptions.includeAddedHistory &&
		!pageReadOptions.includeAddedLatest {
		return makeSeqObjectFromReader(
			pageReader.bufferedReader,
			query,
			pageReader.overflowReaderAt,
		)
	}

	return func(yield func(*sku.Transacted, error) bool) {
		seqAddedHistory := quiter.MakeSeqErrorFromSeq(
			pageReader.additionsHistory.All(),
		)

		{
			seq := quiter.MergeSeqErrorLeft(
				seqAddedHistory,
				makeSeqObjectFromReader(
					pageReader.bufferedReader,
					query,
					pageReader.overflowReaderAt,
				),
				sku.TransactedCompare,
			)

			for object, errIter := range seq {
				if errIter != nil {
					yield(nil, errors.Wrap(errIter))
					return
				}

				if !yield(object, nil) {
					return
				}
			}
		}

		if !pageReadOptions.includeAddedLatest {
			return
		}

		comments.Optimize("determine performance of this")
		seqAddedLatest := quiter.MakeSeqErrorFromSeq(
			pageReader.additionsLatest.All(),
		)

		{
			seq := quiter.MergeSeqErrorLeft(
				seqAddedLatest,
				quiter.MakeSeqErrorEmpty[*sku.Transacted](),
				sku.TransactedCompare,
			)

			for object, errIter := range seq {
				if errIter != nil {
					yield(nil, errors.Wrap(errIter))
					return
				}

				if !yield(object, nil) {
					return
				}
			}
		}
	}
}
