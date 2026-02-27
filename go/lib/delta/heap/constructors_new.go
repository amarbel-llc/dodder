package heap

import (
	"sort"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
)

func MakeNew[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]](
	funcCmp cmp.Func[ELEMENT_PTR],
	resetter interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR],
) *Heap[ELEMENT, ELEMENT_PTR] {
	return &Heap[ELEMENT, ELEMENT_PTR]{
		private: heapPrivate[ELEMENT, ELEMENT_PTR]{
			equaler:  cmp.Equaler[ELEMENT_PTR](funcCmp),
			Lessor:   cmp.Lesser[ELEMENT_PTR](funcCmp),
			Resetter: resetter,
			Elements: make([]ELEMENT_PTR, 0),
		},
	}
}

func MakeNewHeapFromSliceUnsorted[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]](
	funcCmp cmp.Func[ELEMENT_PTR],
	resetter interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR],
	slice []ELEMENT_PTR,
) *Heap[ELEMENT, ELEMENT_PTR] {
	heap := heapPrivate[ELEMENT, ELEMENT_PTR]{
		equaler:  cmp.Equaler[ELEMENT_PTR](funcCmp),
		Lessor:   cmp.Lesser[ELEMENT_PTR](funcCmp),
		Resetter: resetter,
		Elements: slice,
	}

	sort.Sort(heap)

	return &Heap[ELEMENT, ELEMENT_PTR]{
		private: heap,
	}
}

func MakeNewHeapFromSlice[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]](
	funcCmp cmp.Func[ELEMENT_PTR],
	resetter interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR],
	slice []ELEMENT_PTR,
) *Heap[ELEMENT, ELEMENT_PTR] {
	heap := heapPrivate[ELEMENT, ELEMENT_PTR]{
		equaler:  cmp.Equaler[ELEMENT_PTR](funcCmp),
		Lessor:   cmp.Lesser[ELEMENT_PTR](funcCmp),
		Resetter: resetter,
		Elements: slice,
	}

	return &Heap[ELEMENT, ELEMENT_PTR]{
		private: heap,
	}
}
