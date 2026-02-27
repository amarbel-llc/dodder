package equality

import (
	"golang.org/x/exp/constraints"
)

func MapsOrdered[K constraints.Ordered, V constraints.Ordered](
	a, b map[K]V,
) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		v1, ok := b[k]

		if !ok {
			return false
		}

		if v != v1 {
			return false
		}
	}

	return true
}

func SliceOrdered[V constraints.Ordered](
	a, b []V,
) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		v1 := b[i]

		if v != v1 {
			return false
		}
	}

	return true
}
