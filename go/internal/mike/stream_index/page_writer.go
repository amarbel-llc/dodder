package stream_index

import (
	"bufio"
	"os"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/page_id"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/object_probe_index"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

type ObjectIdToObject map[string]objectMetaWithCursorAndSigil

type pageWriter struct {
	writtenPage *page
	pageReader  streamPageReader

	tempFS   env_dir.TemporaryFS
	pageId   page_id.PageId
	preWrite interfaces.FuncIter[*sku.Transacted]
	path     string

	binaryEncoder binaryEncoder

	file *os.File

	changesAreHistorical bool

	probeIndex *probeIndex

	cursor ohio.Cursor

	latestObjects ObjectIdToObject
}

func (index *Index) makePageFlush(
	pageIndex PageIndex,
	changesAreHistorical bool,
) errors.FuncErr {
	page := &index.pages[pageIndex]

	return func() (err error) {
		if !page.writeLock.TryLock() {
			err = errors.Errorf(
				"failed to acquire write lock for page: %q",
				page.pageId,
			)

			return err
		}

		defer page.writeLock.Unlock()

		pageReader, pageReaderClose := index.makeStreamPageReader(pageIndex)
		defer errors.Deferred(&err, pageReaderClose)

		pageWriter := &pageWriter{
			tempFS:      index.envRepo.GetTempLocal(),
			pageId:      page.pageId,
			writtenPage: page,
			pageReader:  pageReader,
			preWrite:    index.preWrite,
			probeIndex:  &index.probeIndex,
			path:        page.pageId.Path(),
		}

		if changesAreHistorical {
			pageWriter.changesAreHistorical = true
			pageWriter.writtenPage.forceFullWrite = true
		}

		if err = pageWriter.Flush(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		page.forceFullWrite = false

		return err
	}
}

func (pageWriter *pageWriter) Flush() (err error) {
	if !pageWriter.writtenPage.hasChanges() {
		ui.Log().Print("not flushing, no changes")
		return err
	}

	defer pageWriter.writtenPage.additionsHistory.Reset()
	defer pageWriter.writtenPage.additionsLatest.Reset()

	pageWriter.latestObjects = make(ObjectIdToObject)

	// If the cache file does not exist and we have nothing to add, short
	// circuit the flush. This condition occurs on the initial init when the
	// konfig is changed but there are no objects yet.
	if !files.Exists(pageWriter.path) &&
		pageWriter.writtenPage.lenAdded() == 0 {
		return err
	}

	ui.Log().Print("changesAreHistorical", pageWriter.changesAreHistorical)
	ui.Log().Print("added", pageWriter.writtenPage.lenAdded())
	ui.Log().Print(
		"addedtail",
		pageWriter.writtenPage.additionsLatest.Len(),
	)

	if pageWriter.writtenPage.additionsHistory.Len() == 0 &&
		!pageWriter.changesAreHistorical {
		if pageWriter.file, err = files.OpenReadWrite(pageWriter.path); err != nil {
			err = errors.Wrap(err)
			return err
		}

		bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
			pageWriter.file,
		)
		defer repoolBufferedWriter()

		defer errors.DeferredCloser(&err, pageWriter.file)

		bufferedReader, repoolBufferedReader := pool.GetBufferedReader(
			pageWriter.file,
		)
		defer repoolBufferedReader()

		if err = pageWriter.flushJustLatest(bufferedReader, bufferedWriter); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if pageWriter.file, err = pageWriter.tempFS.FileTemp(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloseAndRename(
			&err,
			pageWriter.file,
			pageWriter.file.Name(),
			pageWriter.path,
		)

		bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(pageWriter.file)
		defer repoolBufferedWriter()

		if err = pageWriter.flushBoth(bufferedWriter); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (pageWriter *pageWriter) flushBoth(
	bufferedWriter *bufio.Writer,
) (err error) {
	ui.Log().Printf("flushing both: %s", pageWriter.path)

	chain := quiter.MakeChain(
		pageWriter.preWrite,
		pageWriter.makeWriteOne(bufferedWriter),
	)

	seq := pageWriter.pageReader.makeSeq(
		sku.MakePrimitiveQueryGroup(),
		pageReadOptions{
			includeAddedHistory: true,
			includeAddedLatest:  true,
		},
	)

	for object, errIter := range seq {
		if errIter != nil {
			err = errors.Wrap(errIter)
			return err
		}

		if err = chain(object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, object := range pageWriter.latestObjects {
		if err = pageWriter.updateSigilWithLatest(object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (pageWriter *pageWriter) updateSigilWithLatest(
	objectMeta objectMetaWithCursorAndSigil,
) (err error) {
	objectMeta.Add(ids.SigilLatest)

	if err = pageWriter.binaryEncoder.updateSigil(
		pageWriter.file,
		objectMeta.Sigil,
		objectMeta.Offset,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (pageWriter *pageWriter) flushJustLatest(
	bufferedReader *bufio.Reader,
	bufferedWriter *bufio.Writer,
) (err error) {
	ui.Log().Printf("flushing just tail: %s", pageWriter.path)

	{
		seq := pageWriter.pageReader.readFrom(
			bufferedReader,
			sku.MakePrimitiveQueryGroup(),
		)

		for object, errIter := range seq {
			if errIter != nil {
				err = errors.Wrap(errIter)
				return err
			}

			pageWriter.cursor = object.Cursor
			pageWriter.saveToLatestMap(object.Transacted, object.Sigil)
		}
	}

	chain := quiter.MakeChain(
		pageWriter.preWrite,
		pageWriter.removeOldLatest,
		pageWriter.makeWriteOne(bufferedWriter),
	)

	{
		seq := pageWriter.writtenPage.additionsLatest.All()

		for popped := range seq {
			if err = chain(popped); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, object := range pageWriter.latestObjects {
		if err = pageWriter.updateSigilWithLatest(object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (pageWriter *pageWriter) makeWriteOne(
	bufferedWriter *bufio.Writer,
) interfaces.FuncIter[*sku.Transacted] {
	return func(object *sku.Transacted) (err error) {
		// defer func() {
		// 	r := recover()

		// 	if r == nil {
		// 		return
		// 	}

		// 	ui.Debug().Print(z)
		// 	panic(r)
		// }()
		pageWriter.cursor.Offset += pageWriter.cursor.ContentLength

		if pageWriter.cursor.ContentLength, err = pageWriter.binaryEncoder.writeFormat(
			bufferedWriter,
			objectWithSigil{Transacted: object},
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = pageWriter.saveToLatestMap(object, ids.SigilHistory); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = pageWriter.probeIndex.saveOneObjectLoc(
			object,
			object_probe_index.Loc{
				Page:   pageWriter.pageId.Index,
				Cursor: pageWriter.cursor,
			},
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}

func (pageWriter *pageWriter) saveToLatestMap(
	object *sku.Transacted,
	sigil ids.Sigil,
) (err error) {
	objectId := object.GetObjectId()
	objectIdString := objectId.String()

	objectOld := pageWriter.latestObjects[objectIdString]
	objectOld.Cursor = pageWriter.cursor
	objectOld.Tai = object.GetTai()
	objectOld.Sigil = sigil

	if object.GetMetadata().GetIndex().GetDormant().Bool() {
		objectOld.Add(ids.SigilHidden)
	} else {
		objectOld.Del(ids.SigilHidden)
	}

	pageWriter.latestObjects[objectIdString] = objectOld

	return err
}

func (pageWriter *pageWriter) removeOldLatest(
	objectLatest *sku.Transacted,
) (err error) {
	objectIdString := objectLatest.ObjectId.String()
	objectOld, ok := pageWriter.latestObjects[objectIdString]

	if !ok {
		return err
	}

	objectOld.Del(ids.SigilLatest)

	if err = pageWriter.binaryEncoder.updateSigil(
		pageWriter.file,
		objectOld.Sigil,
		objectOld.Offset,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
