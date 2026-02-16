package heap

import (
	pkg_heap "container/heap"
	"sort"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
)

type Element any

type ElementPtr[ELEMENT Element] interface {
	interfaces.Ptr[ELEMENT]
}

type heapPrivate[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]] struct {
	Lessor  interfaces.Lessor[ELEMENT_PTR]
	equaler interfaces.Equaler[ELEMENT_PTR]

	Resetter         interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR]
	Elements         []ELEMENT_PTR
	lastPopped       ELEMENT_PTR
	lastPoppedRepool interfaces.FuncRepool
	pool             interfaces.PoolPtr[ELEMENT, ELEMENT_PTR]
}

func (heap *heapPrivate[ELEMENT, ELEMENT_PTR]) GetPool() interfaces.PoolPtr[ELEMENT, ELEMENT_PTR] {
	if heap.pool == nil {
		heap.pool = pool.MakeFakePool[ELEMENT, ELEMENT_PTR]()
	}

	return heap.pool
}

func (heap heapPrivate[ELEMENT, ELEMENT_PTR]) Len() int {
	return len(heap.Elements)
}

func (heap heapPrivate[ELEMENT, ELEMENT_PTR]) Less(i, j int) (ok bool) {
	ok = heap.Lessor.Less(heap.Elements[i], heap.Elements[j])
	return ok
}

func (heap heapPrivate[ELEMENT, ELEMENT_PTR]) Swap(i, j int) {
	heap.Elements[i], heap.Elements[j] = heap.Elements[j], heap.Elements[i]
}

func (heap *heapPrivate[ELEMENT, ELEMENT_PTR]) Push(x any) {
	heap.Elements = append(heap.Elements, x.(ELEMENT_PTR))
}

func (heap *heapPrivate[ELEMENT, ELEMENT_PTR]) discardDupes() {
	panic("don't use this yet")

	for heap.lastPopped != nil &&
		heap.Len() > 0 &&
		heap.equaler.Equals(heap.lastPopped, heap.Elements[0]) {
		pkg_heap.Pop(heap)
		// d := heap.Pop(h)
		// log.Debug().Printf("discarded: %s", d)
	}
}

func (heap *heapPrivate[ELEMENT, ELEMENT_PTR]) Pop() any {
	old := heap.Elements
	n := len(old)
	x := old[n-1]
	heap.Elements = old[0 : n-1]

	return x
}

func (heap *heapPrivate[ELEMENT, ELEMENT_PTR]) saveLastPopped(e ELEMENT_PTR) {
	if heap.lastPopped == nil {
		heap.lastPopped, heap.lastPoppedRepool = heap.GetPool().GetWithRepool()
	}

	heap.Resetter.ResetWith(heap.lastPopped, e)
}

func (heap heapPrivate[ELEMENT, ELEMENT_PTR]) Copy() (b heapPrivate[ELEMENT, ELEMENT_PTR]) {
	b = heapPrivate[ELEMENT, ELEMENT_PTR]{
		Lessor:   heap.Lessor,
		equaler:  heap.equaler,
		pool:     heap.pool,
		Resetter: heap.Resetter,
		Elements: make([]ELEMENT_PTR, heap.Len()),
	}

	copy(b.Elements, heap.Elements)

	return b
}

func (heap heapPrivate[ELEMENT, ELEMENT_PTR]) Sorted() (b heapPrivate[ELEMENT, ELEMENT_PTR]) {
	b = heap.Copy()
	sort.Sort(b)
	return b
}
