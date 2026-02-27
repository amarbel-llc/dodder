package stream_index

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/page_id"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type probePageReader struct {
	pageId   page_id.PageId
	readerAt io.ReaderAt
	decoder  binaryDecoder
}

func (index *Index) makeProbePageReader(
	pageIndex PageIndex,
) (probePageReader, errors.FuncErr) {
	page := &index.pages[pageIndex]
	pageReader := probePageReader{
		pageId:  page.pageId,
		decoder: makeBinaryWithQueryGroup(nil, ids.SigilHistory),
	}

	var err error
	var blobReader domain_interfaces.BlobReader

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

	return pageReader, func() (err error) {
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
) (ok bool) {
	// pages get deleted before reindexing, so this is actually valid to have a
	// non-nil cursor request
	if pageReader.readerAt == nil {
		return ok
	}

	var bytesRead int64

	objectPlus := objectWithCursorAndSigil{
		objectWithSigil: objectWithSigil{
			Transacted: object,
		},
		Cursor: cursor,
	}

	{
		var err error

		if bytesRead, err = pageReader.decoder.readFormatExactly(
			pageReader.readerAt,
			&objectPlus,
		); err != nil {
			ui.Debug().Print(err)
			if err == io.EOF {
				if bytesRead == cursor.ContentLength {
					goto NO_ERR
				} else {
					panic(io.ErrUnexpectedEOF)
				}
			}

			err = errors.Wrapf(
				err,
				"Range: %q, Page: %q, BytesRead: %d",
				cursor,
				pageReader.pageId.Path(),
				bytesRead,
			)

			panic(err)
		}
	}

NO_ERR:

	ok = true

	return ok
}
