package sku

import (
	"bufio"
	"fmt"
	"sync"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/descriptions"
	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/alfa/collections_map"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/heap"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
)

type ListCoder = interfaces.CoderBufferedReadWriter[*Transacted]

// TODO add lock
// TODO add iterate method
type WorkingList struct {
	lock        sync.RWMutex
	description descriptions.Description

	coder ListCoder
	// makeBlobWriter is invoked lazily on the first Add: a blob writer
	// eagerly creates its temp file at construction, so an empty working
	// list that opened one would leak that file when discarded without
	// Close (issue #366).
	makeBlobWriter           domain_interfaces.FuncObjectWriter
	blobWriter               mad_domain_interfaces.BlobWriter
	bufferedBlobWriter       *bufio.Writer
	bufferedBlobWriterRepool interfaces.FuncRepool
	cursor                   ohio.Cursor
	count                    int

	indexOrder     *heap.Heap[TransactedCursor, *TransactedCursor]
	indexObjectIds collections_map.Map[string, collections_slice.Slice[ohio.Cursor]]

	funcPreWrite func(*Transacted) error
}

func MakeWorkingList(
	coder ListCoder,
	makeBlobWriter domain_interfaces.FuncObjectWriter,
	funcPreWrite interfaces.FuncIter[*Transacted],
) *WorkingList {
	return &WorkingList{
		coder:          coder,
		makeBlobWriter: makeBlobWriter,
		indexOrder:     MakeHeapTransactedCursor(),
		indexObjectIds: make(collections_map.Map[string, collections_slice.Slice[ohio.Cursor]]),
		funcPreWrite:   funcPreWrite,
	}
}

func (list *WorkingList) GetDescription() descriptions.Description {
	return list.description
}

func (list *WorkingList) GetDescriptionMutable() *descriptions.Description {
	return &list.description
}

func (list *WorkingList) getBufferedBlobWriter() (*bufio.Writer, error) {
	if list.blobWriter == nil {
		var err error

		if list.blobWriter, err = list.makeBlobWriter(); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	if list.bufferedBlobWriter == nil {
		list.bufferedBlobWriter, list.bufferedBlobWriterRepool = pool.GetBufferedWriter(
			list.blobWriter,
		)
	}

	return list.bufferedBlobWriter, nil
}

func (list *WorkingList) Len() int {
	list.lock.RLock()
	defer list.lock.RUnlock()

	return list.count
}

func (list *WorkingList) Add(object *Transacted) (err error) {
	list.lock.Lock()
	defer list.lock.Unlock()

	if list.funcPreWrite != nil {
		if err = list.funcPreWrite(object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	list.cursor.Offset += list.cursor.ContentLength

	if list.cursor.ContentLength, err = list.writeObject(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	objectIdString := object.GetObjectId().String()

	list.indexOrder.Push(&TransactedCursor{
		tai:            object.GetTai(),
		objectIdString: objectIdString,
		cursor:         list.cursor,
	})

	{
		objects, _ := list.indexObjectIds.Get(objectIdString)
		objects.Append(list.cursor)
		list.indexObjectIds.Set(objectIdString, objects)
	}

	return err
}

func (list *WorkingList) writeObject(
	object *Transacted,
) (n int64, err error) {
	var bufferedBlobWriter *bufio.Writer

	if bufferedBlobWriter, err = list.getBufferedBlobWriter(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if n, err = list.coder.EncodeTo(
		object,
		bufferedBlobWriter,
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = bufferedBlobWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	list.count += 1

	return n, err
}

// CloseEmpty closes a working list that MUST hold no pending records:
// the discard path for lists being replaced or abandoned
// (store.Initialize's re-initialization guard,
// inventory_list_store.Create's empty-list early return) — as opposed
// to Close, which is also the finalize path for sealing a non-empty
// list's blob. A pending Add reaching a discard site would be silently
// lost: the object stays durable in the stream index but is never
// recorded in any inventory list, invisible to history and sync
// (dodder#369). Error loudly instead of trusting call sites to uphold
// the emptiness invariant by convention.
func (list *WorkingList) CloseEmpty() (err error) {
	if count := list.Len(); count > 0 {
		err = errors.Errorf(
			"refusing to discard working list with %d pending record(s): closing here would silently drop them from inventory-list history (dodder#369)",
			count,
		)
		return err
	}

	return list.Close()
}

func (list *WorkingList) Close() (err error) {
	if !list.lock.TryLock() {
		err = errors.Errorf("trying to close open list while lock is acquired")
		return err
	}

	defer list.lock.Unlock()

	// bufferedBlobWriter is non-nil exactly when the lazy blob writer is
	// open: skip when nothing was ever added (no temp file exists) or a
	// previous Close already released everything
	if list.bufferedBlobWriter != nil {
		// repool even when flush or close errors: the borrowed buffer is
		// abandoned either way
		defer func() {
			list.bufferedBlobWriter = nil
			list.bufferedBlobWriterRepool()
			list.bufferedBlobWriterRepool = nil
		}()

		if err = list.bufferedBlobWriter.Flush(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = list.blobWriter.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	list.cursor.Reset()
	list.indexOrder.Reset()
	list.indexObjectIds.Reset()

	return err
}

func (list *WorkingList) GetMarklId() mad_domain_interfaces.MarklId {
	if !list.lock.TryLock() {
		panic(fmt.Sprintf("trying to get markl id from open list while lock is acquired"))
	}

	defer list.lock.Unlock()

	if list.blobWriter == nil {
		panic("trying to get markl id from working list that never opened its blob writer")
	}

	return list.blobWriter.GetMarklId()
}
