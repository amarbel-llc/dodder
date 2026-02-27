package sku

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/lib/_/reset"
	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/delta/heap"
)

type (
	TransactedCursor struct {
		tai            ids.Tai
		objectIdString string
		cursor         ohio.Cursor
	}

	HeapTransactedCursor = heap.Heap[TransactedCursor, *TransactedCursor]
)

func MakeHeapTransactedCursor() *HeapTransactedCursor {
	heap := heap.MakeNew(
		TransactedCursorCompare,
		reset.MakeResetter(
			(*TransactedCursor).Reset,
			(*TransactedCursor).ResetWith,
		),
	)

	return heap
}

func (cursor *TransactedCursor) Reset() {
	cursor.tai.Reset()
	cursor.objectIdString = ""
	cursor.cursor.Reset()
}

func (cursor *TransactedCursor) ResetWith(src *TransactedCursor) {
	cursor.tai.ResetWith(src.tai)
	cursor.objectIdString = src.objectIdString
	cursor.cursor = src.cursor
}

func TransactedCursorCompare(left, right *TransactedCursor) cmp.Result {
	if TransactedCursorLess(left, right) {
		return cmp.Less
	} else if TransactedCursorEqual(left, right) {
		return cmp.Equal
	} else {
		return cmp.Greater
	}
}

func TransactedCursorLess(a, b *TransactedCursor) bool {
	if result := a.tai.SortCompare(b.tai); !result.IsEqual() {
		return result.IsLess()
	}

	return a.objectIdString < b.objectIdString
}

func TransactedCursorEqual(a, b *TransactedCursor) bool {
	if result := a.tai.SortCompare(b.tai); !result.IsEqual() {
		return result.IsLess()
	}

	return a.objectIdString < b.objectIdString
}
