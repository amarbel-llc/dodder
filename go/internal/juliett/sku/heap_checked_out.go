package sku

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/heap"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
)

type HeapCheckedOut = heap.Heap[CheckedOut, *CheckedOut]

func MakeListCheckedOut() *HeapCheckedOut {
	heap := heap.MakeNew(
		cmp.MakeFuncFromEqualerAndLessor3LessFirst(
			genericEqualer[*CheckedOut]{},
			genericLessorStable[*CheckedOut]{},
		),
		CheckedOutResetter,
	)

	heap.SetPool(GetCheckedOutPool())

	return heap
}

var ResetterListCheckedOut resetterListCheckedOut

type resetterListCheckedOut struct{}

func (resetterListCheckedOut) Reset(a *HeapCheckedOut) {
	a.Reset()
}

func (resetterListCheckedOut) ResetWith(a, b *HeapCheckedOut) {
	a.ResetWith(b)
}

func CollectListCheckedOut(
	seq interfaces.SeqError[*CheckedOut],
) (list *HeapCheckedOut, err error) {
	list = MakeListCheckedOut()

	for checkedOut, iterErr := range seq {
		if iterErr != nil {
			err = errors.Wrap(iterErr)
			return list, err
		}

		list.Add(checkedOut)
	}

	return list, err
}

type genericLessorTaiOnly[T ids.Clock] struct{}

func (genericLessorTaiOnly[T]) Less(a, b T) bool {
	return a.GetTai().Less(b.GetTai())
}

type clockWithObjectId interface {
	ids.Clock
	// TODO figure out common interface for this
	GetObjectId() *ids.ObjectId
}

type genericLessorStable[T clockWithObjectId] struct{}

func (genericLessorStable[T]) Less(a, b T) bool {
	if result := a.GetTai().SortCompare(b.GetTai()); !result.IsEqual() {
		return result.IsLess()
	}

	return a.GetObjectId().String() < b.GetObjectId().String()
}

type genericEqualer[T interface {
	Equals(T) bool
}] struct{}

func (genericEqualer[T]) Equals(a, b T) bool {
	return a.Equals(b)
}
