package stream_index

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/page_id"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
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
) (ok bool, err error) {
	// pages get deleted before reindexing, so this is actually valid to have a
	// non-nil cursor request
	if pageReader.readerAt == nil {
		return
	}

	var bytesRead int64

	objectPlus := objectWithCursorAndSigil{
		objectWithSigil: objectWithSigil{
			Transacted: object,
		},
		Cursor: cursor,
	}

	if bytesRead, err = pageReader.decoder.readFormatExactly(
		pageReader.readerAt,
		&objectPlus,
	); err != nil {
		ui.Debug().Print(err)
		if err == io.EOF {
			if bytesRead == cursor.ContentLength {
				err = nil
				ok = true
				return
			}

			err = errors.Wrap(io.ErrUnexpectedEOF)
			return
		}

		err = errors.Wrapf(
			err,
			"Range: %q, Page: %q, BytesRead: %d",
			cursor,
			pageReader.pageId.Path(),
			bytesRead,
		)

		return
	}

	ok = true

	return
}
