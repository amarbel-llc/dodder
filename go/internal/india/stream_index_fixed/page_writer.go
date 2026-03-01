package stream_index_fixed

import (
	"bufio"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/page_id"
	"code.linenisgreat.com/dodder/go/internal/echo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/object_probe_index"
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

	// cursor tracks the current entry position as entry index * EntryWidth
	cursor ohio.Cursor

	latestObjects ObjectIdToObject

	overflow *overflowWriter
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

		pw := &pageWriter{
			tempFS:      index.envRepo.GetTempLocal(),
			pageId:      page.pageId,
			writtenPage: page,
			pageReader:  pageReader,
			preWrite:    index.preWrite,
			probeIndex:  &index.probeIndex,
			path:        page.pageId.Path(),
		}

		if changesAreHistorical {
			pw.changesAreHistorical = true
			pw.writtenPage.forceFullWrite = true
		}

		if err = pw.Flush(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		page.forceFullWrite = false

		return err
	}
}

func (pw *pageWriter) Flush() (err error) {
	if !pw.writtenPage.hasChanges() {
		ui.Log().Print("not flushing, no changes")
		return err
	}

	defer pw.writtenPage.additionsHistory.Reset()
	defer pw.writtenPage.additionsLatest.Reset()

	pw.latestObjects = make(ObjectIdToObject)

	if !files.Exists(pw.path) &&
		pw.writtenPage.lenAdded() == 0 {
		return err
	}

	ui.Log().Print("changesAreHistorical", pw.changesAreHistorical)
	ui.Log().Print("added", pw.writtenPage.lenAdded())
	ui.Log().Print(
		"addedtail",
		pw.writtenPage.additionsLatest.Len(),
	)

	if pw.writtenPage.additionsHistory.Len() == 0 &&
		!pw.changesAreHistorical {
		if pw.file, err = files.OpenReadWrite(pw.path); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, pw.file)

		if pw.overflow, err = makeOverflowWriter(pw.pageId); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, pw.overflow)

		bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
			pw.file,
		)
		defer repoolBufferedWriter()

		bufferedReader, repoolBufferedReader := pool.GetBufferedReader(
			pw.file,
		)
		defer repoolBufferedReader()

		if err = pw.flushJustLatest(bufferedReader, bufferedWriter); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if pw.file, err = pw.tempFS.FileTemp(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloseAndRename(
			&err,
			pw.file,
			pw.file.Name(),
			pw.path,
		)

		// For full rewrite, create overflow sidecar as temp file too.
		var overflowTempFile *os.File

		if overflowTempFile, err = pw.tempFS.FileTemp(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		ovId := overflowPageId(pw.pageId)
		overflowPath := ovId.Path()

		defer errors.DeferredCloseAndRename(
			&err,
			overflowTempFile,
			overflowTempFile.Name(),
			overflowPath,
		)

		pw.overflow = makeOverflowWriterForTempFile(overflowTempFile)

		bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(pw.file)
		defer repoolBufferedWriter()

		if err = pw.flushBoth(bufferedWriter); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (pw *pageWriter) flushBoth(
	bufferedWriter *bufio.Writer,
) (err error) {
	ui.Log().Printf("flushing both: %s", pw.path)

	chain := quiter.MakeChain(
		pw.preWrite,
		pw.makeWriteOne(bufferedWriter),
	)

	seq := pw.pageReader.makeSeq(
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

	for _, object := range pw.latestObjects {
		if err = pw.updateSigilWithLatest(object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (pw *pageWriter) updateSigilWithLatest(
	objectMeta objectMetaWithCursorAndSigil,
) (err error) {
	objectMeta.Add(ids.SigilLatest)

	if err = pw.binaryEncoder.updateSigil(
		pw.file,
		objectMeta.Sigil,
		objectMeta.Offset,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (pw *pageWriter) flushJustLatest(
	bufferedReader *bufio.Reader,
	bufferedWriter *bufio.Writer,
) (err error) {
	ui.Log().Printf("flushing just tail: %s", pw.path)

	{
		seq := pw.pageReader.readFrom(
			bufferedReader,
			sku.MakePrimitiveQueryGroup(),
		)

		for object, errIter := range seq {
			if errIter != nil {
				err = errors.Wrap(errIter)
				return err
			}

			pw.cursor = object.Cursor
			pw.saveToLatestMap(object.Transacted, object.Sigil)
		}
	}

	chain := quiter.MakeChain(
		pw.preWrite,
		pw.removeOldLatest,
		pw.makeWriteOne(bufferedWriter),
	)

	{
		seq := pw.writtenPage.additionsLatest.All()

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

	for _, object := range pw.latestObjects {
		if err = pw.updateSigilWithLatest(object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (pw *pageWriter) makeWriteOne(
	bufferedWriter *bufio.Writer,
) interfaces.FuncIter[*sku.Transacted] {
	return func(object *sku.Transacted) (err error) {
		pw.cursor.Offset += pw.cursor.ContentLength

		if err = pw.binaryEncoder.writeFixedEntry(
			bufferedWriter,
			objectWithSigil{Transacted: object},
			pw.overflow,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		pw.cursor.ContentLength = int64(EntryWidth)

		if err = pw.saveToLatestMap(object, ids.SigilHistory); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = pw.probeIndex.saveOneObjectLoc(
			object,
			object_probe_index.Loc{
				Page:   pw.pageId.Index,
				Cursor: pw.cursor,
			},
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}

func (pw *pageWriter) saveToLatestMap(
	object *sku.Transacted,
	sigil ids.Sigil,
) (err error) {
	objectId := object.GetObjectId()
	objectIdString := objectId.String()

	objectOld := pw.latestObjects[objectIdString]
	objectOld.Cursor = pw.cursor
	objectOld.Tai = object.GetTai()
	objectOld.Sigil = sigil

	if object.GetMetadata().GetIndex().GetDormant().Bool() {
		objectOld.Add(ids.SigilHidden)
	} else {
		objectOld.Del(ids.SigilHidden)
	}

	pw.latestObjects[objectIdString] = objectOld

	return err
}

func (pw *pageWriter) removeOldLatest(
	objectLatest *sku.Transacted,
) (err error) {
	objectIdString := objectLatest.GetObjectId().String()
	objectOld, ok := pw.latestObjects[objectIdString]

	if !ok {
		return err
	}

	objectOld.Del(ids.SigilLatest)

	if err = pw.binaryEncoder.updateSigil(
		pw.file,
		objectOld.Sigil,
		objectOld.Offset,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
