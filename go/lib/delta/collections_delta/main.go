package collections_delta

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/charlie/collections_value"
)

type delta[ELEMENT interfaces.Stringer] struct {
	Added, Removed interfaces.SetMutable[ELEMENT]
}

func (d delta[ELEMENT]) GetAdded() interfaces.Set[ELEMENT] {
	return d.Added
}

func (d delta[ELEMENT]) GetRemoved() interfaces.Set[ELEMENT] {
	return d.Removed
}

func MakeSetDelta[ELEMENT interfaces.Stringer](
	from, to interfaces.Set[ELEMENT],
) interfaces.Delta[ELEMENT] {
	delta := delta[ELEMENT]{
		Added:   collections_value.MakeMutableValueSet[ELEMENT](nil),
		Removed: collections_value.MakeMutableValueSet[ELEMENT](nil),
	}

	for element := range from.All() {
		delta.Removed.Add(element)
	}

	for element := range to.All() {
		if from.ContainsKey(element.String()) {
			// had previously
		} else {
			// did not have previously
			delta.Added.Add(element)
		}

		quiter_set.Del(delta.Removed, element)
	}

	return delta
}
