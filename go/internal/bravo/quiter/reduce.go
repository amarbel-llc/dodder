package quiter

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

func CollectSlice[ELEMENT any](
	collection interfaces.Collection[ELEMENT],
) (slice []ELEMENT) {
	if collection == nil {
		return slice
	}

	slice = make([]ELEMENT, 0, collection.Len())

	for element := range collection.All() {
		slice = append(slice, element)
	}

	return slice
}

func ElementsSorted[T any](
	s interfaces.Collection[T],
	sf func(T, T) bool,
) (out []T) {
	if s == nil {
		return out
	}

	out = make([]T, 0, s.Len())

	for v := range s.All() {
		out = append(out, v)
	}

	sort.Slice(out, func(i, j int) bool {
		return sf(out[i], out[j])
	})

	return out
}
