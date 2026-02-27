package collections_slice

import (
	"sort"

	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
)

func (slice *Slice[ELEMENT]) SortByStringFunc(getKey func(ELEMENT) string) {
	sort.Slice(
		*slice,
		func(left, right int) bool {
			return getKey(slice.At(left)) < getKey(slice.At(right))
		},
	)
}

func (slice *Slice[ELEMENT]) SortWithComparer(cmp cmp.Func[ELEMENT]) {
	sort.Slice(
		*slice,
		func(left, right int) bool {
			return cmp(slice.At(left), slice.At(right)).IsLess()
		},
	)
}
