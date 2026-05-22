package sku

import (
	"bufio"
	"fmt"
	"sync"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/descriptions"
	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/alfa/collections_map"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/heap"
)

type ListCoder = interfaces.CoderBufferedReadWriter[*Transacted]

// TODO add lock
// TODO add iterate method
type WorkingList struct {
	lock        sync.RWMutex
	description descriptions.Description

	coder                    ListCoder
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
	blobWriter mad_domain_interfaces.BlobWriter,
	funcPreWrite interfaces.FuncIter[*Transacted],
) *WorkingList {
	return &WorkingList{
		coder:          coder,
		blobWriter:     blobWriter,
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

func (list *WorkingList) getBufferedBlobWriter() *bufio.Writer {
	if list.bufferedBlobWriter == nil {
		list.bufferedBlobWriter, list.bufferedBlobWriterRepool = pool.GetBufferedWriter(
			list.blobWriter,
		)
	}

	return list.bufferedBlobWriter
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
	if n, err = list.coder.EncodeTo(
		object,
		list.getBufferedBlobWriter(),
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = list.getBufferedBlobWriter().Flush(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	list.count += 1

	return n, err
}

func (list *WorkingList) Close() (err error) {
	if !list.lock.TryLock() {
		err = errors.Errorf("trying to close open list while lock is acquired")
		return err
	}

	defer list.lock.Unlock()

	if err = list.getBufferedBlobWriter().Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = list.blobWriter.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	list.cursor.Reset()
	list.indexOrder.Reset()
	list.indexObjectIds.Reset()
	list.bufferedBlobWriter = nil
	list.bufferedBlobWriterRepool()

	return err
}

func (list *WorkingList) GetMarklId() mad_domain_interfaces.MarklId {
	if !list.lock.TryLock() {
		panic(fmt.Sprintf("trying to get markl id from open list while lock is acquired"))
	}

	defer list.lock.Unlock()

	return list.blobWriter.GetMarklId()
}
