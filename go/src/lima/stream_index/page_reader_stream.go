package stream_index

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/comments"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type streamPageReader struct {
	*page
	blobReader      domain_interfaces.BlobReader
	bufferedReader  *bufio.Reader
	namedBlobAccess domain_interfaces.NamedBlobAccess
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

	return pageReader, func() error {
		repool()
		return pageReader.blobReader.Close()
	}
}

func makeSeqObjectWithCursorAndSigilFromReader(
	reader io.Reader,
	queryGroup sku.PrimitiveQueryGroup,
) interfaces.SeqError[objectWithCursorAndSigil] {
	return func(yield func(objectWithCursorAndSigil, error) bool) {
		decoder := makeBinaryWithQueryGroup(queryGroup, ids.SigilHistory)

		var object objectWithCursorAndSigil
		object.Transacted, _ = sku.GetTransactedPool().GetWithRepool()

		for {
			sku.TransactedResetter.Reset(object.Transacted)
			object.Offset += object.ContentLength

			var err error

			if object.ContentLength, err = decoder.readFormatAndMatchSigil(
				reader,
				&object,
			); err != nil {
				yield(object, errors.WrapExceptSentinelAsNil(err, io.EOF))
				return
			}

			if !yield(object, nil) {
				return
			}
		}
	}
}

func makeSeqObjectFromReader(
	reader io.Reader,
	queryGroup sku.PrimitiveQueryGroup,
) interfaces.SeqError[*sku.Transacted] {
	return func(yield func(*sku.Transacted, error) bool) {
		seq := makeSeqObjectWithCursorAndSigilFromReader(reader, queryGroup)
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
		decoder := makeBinaryWithQueryGroup(queryGroup, ids.SigilHistory)

		var object objectWithCursorAndSigil

		object.Transacted, _ = sku.GetTransactedPool().GetWithRepool()

		for {
			sku.TransactedResetter.Reset(object.Transacted)

			object.Offset += object.ContentLength

			var err error

			if object.ContentLength, err = decoder.readFormatAndMatchSigil(
				reader,
				&object,
			); err == io.EOF {
				if err == io.EOF {
					return
				}
			} else if err != nil {
				object.Transacted = nil
				yield(object, errors.Wrap(err))
				return
			}

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
		)
	}

	return func(yield func(*sku.Transacted, error) bool) {
		seqAddedHistory := quiter.MakeSeqErrorFromSeq(
			pageReader.additionsHistory.All(),
		)

		{
			seq := quiter.MergeSeqErrorLeft(
				seqAddedHistory,
				makeSeqObjectFromReader(pageReader.bufferedReader, query),
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
