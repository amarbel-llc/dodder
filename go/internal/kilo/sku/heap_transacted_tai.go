package sku

import (
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/heap"
)

type HeapTransacted = heap.Heap[Transacted, *Transacted]

func MakeListTransacted() *HeapTransacted {
	heap := heap.MakeNew(
		TransactedCompare,
		transactedResetter{},
	)

	heap.SetPool(GetTransactedPool())

	return heap
}

var ResetterList resetterList

type resetterList struct{}

func (resetterList) Reset(list *HeapTransacted) {
	list.Reset()
}

func (resetterList) ResetWith(left, right *HeapTransacted) {
	left.ResetWith(right)
}

func CollectList(
	seq Seq,
) (list *HeapTransacted, err error) {
	list = MakeListTransacted()

	for object, iterErr := range seq {
		if iterErr != nil {
			err = errors.Wrap(iterErr)
			return list, err
		}

		list.Add(object)
	}

	return list, err
}
