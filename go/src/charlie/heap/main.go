package heap

import (
	pkg_heap "container/heap"
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
)

// TODO rewrite with cmp.FuncCmp
type Heap[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]] struct {
	lock       sync.Mutex
	private    heapPrivate[ELEMENT, ELEMENT_PTR]
	savedIndex int
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) GetCollection() interfaces.Collection[ELEMENT_PTR] {
	return heap
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Any() ELEMENT_PTR {
	element, _ := heap.Peek()
	return element
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) All() interfaces.Seq[ELEMENT_PTR] {
	return func(yield func(ELEMENT_PTR) bool) {
		heap.lock.Lock()
		defer heap.lock.Unlock()
		defer heap.restore()

		for {
			element, ok := heap.popAndSave()

			if !ok {
				return
			}

			if !yield(element) {
				break
			}
		}
	}
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) SetPool(
	v interfaces.PoolPtr[ELEMENT, ELEMENT_PTR],
) {
	heap.private.pool = v
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) GetEqualer() interfaces.Equaler[ELEMENT_PTR] {
	return heap.private.equaler
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) GetLessor() interfaces.Lessor[ELEMENT_PTR] {
	return heap.private.Lessor
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) GetResetter() interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR] {
	return heap.private.Resetter
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Peek() (element ELEMENT_PTR, ok bool) {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	if heap.private.Len() > 0 {
		element = heap.private.Elements[0]
		ok = true
	}

	return element, ok
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Add(element ELEMENT_PTR) (err error) {
	heap.Push(element)
	return err
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Push(element ELEMENT_PTR) {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	if heap.savedIndex > 0 {
		panic(
			errors.ErrorWithStackf(
				"attempting to push to a heap that has saved elements",
			),
		)
	}

	pkg_heap.Push(&heap.private, element)
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) PopAndSave() (element ELEMENT_PTR, ok bool) {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	return heap.popAndSave()
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) popAndSave() (element ELEMENT_PTR, ok bool) {
	// h.h.discardDupes()

	if heap.private.Len() == 0 {
		return element, ok
	}

	element, _ = heap.private.GetPool().GetWithRepool()
	e := pkg_heap.Pop(&heap.private).(ELEMENT_PTR)
	heap.private.Resetter.ResetWith(element, e)
	ok = true
	heap.savedIndex += 1
	faked := heap.private.Elements[:heap.private.Len()+heap.savedIndex]
	faked[heap.private.Len()] = e
	heap.private.saveLastPopped(element)

	return element, ok
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Restore() {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	heap.restore()
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) restore() {
	heap.private.Elements = heap.private.Elements[:heap.savedIndex]
	heap.savedIndex = 0

	if heap.private.lastPoppedRepool != nil {
		heap.private.lastPoppedRepool()
		heap.private.lastPoppedRepool = nil
	}

	heap.private.lastPopped = nil

	quiter.ReverseSortable(&heap.private)
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) PopOrErrStopIteration() (element ELEMENT_PTR, err error) {
	ok := false
	element, ok = heap.Pop()

	if !ok {
		err = errors.MakeErrStopIteration()
	}

	return element, err
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) PopError() (element ELEMENT_PTR, err error) {
	element, _ = heap.Pop()
	return element, err
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Pop() (element ELEMENT_PTR, ok bool) {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	// h.h.discardDupes()

	if heap.private.Len() == 0 {
		return element, ok
	}

	element, _ = heap.private.GetPool().GetWithRepool()
	heap.private.Resetter.ResetWith(
		element,
		pkg_heap.Pop(&heap.private).(ELEMENT_PTR),
	)
	ok = true
	heap.private.saveLastPopped(element)

	return element, ok
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Len() int {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	return heap.private.Len()
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Equals(
	b *Heap[ELEMENT, ELEMENT_PTR],
) bool {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	if heap.private.Len() != b.private.Len() {
		return false
	}

	for i, av := range heap.private.Elements {
		if b.private.equaler.Equals(b.private.Elements[i], av) {
			return false
		}
	}

	return true
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Copy() Heap[ELEMENT, ELEMENT_PTR] {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	return Heap[ELEMENT, ELEMENT_PTR]{
		private: heap.private.Copy(),
	}
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Fix() {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	pkg_heap.Init(&heap.private)
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Sorted() (heapCopy []ELEMENT_PTR) {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	heapCopy = heap.private.Sorted().Elements

	return heapCopy
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) Reset() {
	heap.private.Elements = make([]ELEMENT_PTR, 0)

	if heap.private.lastPoppedRepool != nil {
		heap.private.lastPoppedRepool()
		heap.private.lastPoppedRepool = nil
	}

	heap.private.pool = nil
	heap.private.lastPopped = nil
}

func (heap *Heap[ELEMENT, ELEMENT_PTR]) ResetWith(
	b *Heap[ELEMENT, ELEMENT_PTR],
) {
	heap.private.equaler = b.private.equaler
	heap.private.Lessor = b.private.Lessor
	heap.private.Resetter = b.private.Resetter
	heap.private.Elements = make([]ELEMENT_PTR, b.Len())

	copy(heap.private.Elements, b.private.Elements)

	heap.private.pool = b.private.GetPool()
}
