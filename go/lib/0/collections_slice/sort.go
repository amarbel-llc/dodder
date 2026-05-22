package collections_slice

import (
	"slices"
	"sort"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/cmp"
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

func SortedValues[ELEMENT interfaces.Value](
	seq interfaces.Seq[ELEMENT],
) []ELEMENT {
	out := slices.Collect(seq)

	sort.Slice(
		out,
		func(i, j int) bool { return out[i].String() < out[j].String() },
	)

	return out
}

func SortedValuesBy[ELEMENT any](
	collection interfaces.Collection[ELEMENT],
	cmpFunc cmp.Func[ELEMENT],
) []ELEMENT {
	out := slices.Collect(collection.All())

	sort.Slice(
		out,
		func(left, right int) bool {
			return cmpFunc(out[left], out[right]).IsLess()
		},
	)

	return out
}
