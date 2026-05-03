package stream_index_fixed

import (
	"io"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/alfa/page_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type probePageReader struct {
	pageId           page_id.PageId
	readerAt         io.ReaderAt
	overflowReaderAt *overflowReaderAt
	decoder          binaryDecoder
}

func (index *Index) makeProbePageReader(
	pageIndex PageIndex,
) (probePageReader, errors.FuncErr) {
	page := &index.pages[pageIndex]
	pageReader := probePageReader{
		pageId:  page.pageId,
		decoder: makeBinaryDecoderWithQueryGroup(nil, ids.SigilHistory),
	}

	var err error
	var blobReader mad_domain_interfaces.BlobReader

	if blobReader, err = index.envRepo.MakeNamedBlobReader(
		pageReader.pageId.Path(),
	); err != nil {
		if errors.IsNotExist(err) {
			return pageReader, func() error { return nil }
		} else {
			panic(err)
		}
	}

	pageReader.readerAt = blobReader

	// Open overflow sidecar for probe reads.
	ovId := overflowPageId(page.pageId)
	overflowPath := ovId.Path()
	var overflowBlobReader mad_domain_interfaces.BlobReader

	overflowBlobReader, err = env_repo.MakeNamedBlobReaderOrNullReader(
		index.envRepo,
		overflowPath,
	)
	if err != nil {
		panic(err)
	}

	pageReader.overflowReaderAt = &overflowReaderAt{readerAt: overflowBlobReader}

	return pageReader, func() (err error) {
		overflowBlobReader.Close()

		if err = blobReader.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}

func (pageReader *probePageReader) readOneCursor(
	cursor ohio.Cursor,
	object *sku.Transacted,
) (ok bool, err error) {
	if pageReader.readerAt == nil {
		return
	}

	if err = pageReader.decoder.readFixedEntry(
		pageReader.readerAt,
		pageReader.overflowReaderAt,
		cursor.Offset,
		object,
	); err != nil {
		err = errors.Wrapf(
			err,
			"Range: %q, Page: %q",
			cursor,
			pageReader.pageId.Path(),
		)

		return
	}

	ok = true

	return
}
