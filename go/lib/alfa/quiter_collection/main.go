package quiter_collection

import (
	"iter"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func Equals[ELEMENT interfaces.Stringer](
	left, right interfaces.Collection[ELEMENT],
) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil {
		return right.Len() == 0
	}

	if right == nil {
		return left.Len() == 0
	}

	if left.Len() != right.Len() {
		return false
	}

	pullRight, stop := iter.Pull(right.All())
	defer stop()

	for elementLeft := range left.All() {
		elementRight, ok := pullRight()

		if !ok {
			return false
		}

		if elementLeft.String() != elementRight.String() {
			return false
		}
	}

	return true
}
