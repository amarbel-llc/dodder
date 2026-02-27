package heap

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

func Make[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]](
	equaler interfaces.Equaler[ELEMENT_PTR],
	lessor interfaces.Lessor[ELEMENT_PTR],
	resetter interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR],
) *Heap[ELEMENT, ELEMENT_PTR] {
	return &Heap[ELEMENT, ELEMENT_PTR]{
		private: heapPrivate[ELEMENT, ELEMENT_PTR]{
			equaler:  equaler,
			Lessor:   lessor,
			Resetter: resetter,
			Elements: make([]ELEMENT_PTR, 0),
		},
	}
}

func MakeHeapFromSliceUnsorted[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]](
	equaler interfaces.Equaler[ELEMENT_PTR],
	lessor interfaces.Lessor[ELEMENT_PTR],
	resetter interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR],
	slice []ELEMENT_PTR,
) *Heap[ELEMENT, ELEMENT_PTR] {
	h := heapPrivate[ELEMENT, ELEMENT_PTR]{
		Lessor:   lessor,
		Resetter: resetter,
		Elements: slice,
		equaler:  equaler,
	}

	sort.Sort(h)

	return &Heap[ELEMENT, ELEMENT_PTR]{
		private: h,
	}
}

func MakeHeapFromSlice[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]](
	equaler interfaces.Equaler[ELEMENT_PTR],
	lessor interfaces.Lessor[ELEMENT_PTR],
	resetter interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR],
	slice []ELEMENT_PTR,
) *Heap[ELEMENT, ELEMENT_PTR] {
	h := heapPrivate[ELEMENT, ELEMENT_PTR]{
		Lessor:   lessor,
		Resetter: resetter,
		Elements: slice,
		equaler:  equaler,
	}

	return &Heap[ELEMENT, ELEMENT_PTR]{
		private: h,
	}
}
