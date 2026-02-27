package quiter_set

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_seq"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func Any[ELEMENT any](set interfaces.Set[ELEMENT]) (element ELEMENT) {
	return quiter_seq.Any(set.All())
}

func AllWithIndex[ELEMENT any](set interfaces.Set[ELEMENT]) interfaces.Seq2[int, ELEMENT] {
	return quiter_seq.SeqWithIndex(set.All())
}

func Contains[ELEMENT any](
	set interfaces.Set[ELEMENT],
	element ELEMENT,
) bool {
	return set.ContainsKey(set.Key(element))
}

func Del[ELEMENT any](set interfaces.SetMutable[ELEMENT], element ELEMENT) {
	key := set.Key(element)
	errors.PanicIfError(set.DelKey(key))
}

func Equals[ELEMENT interfaces.Stringer](
	left, right interfaces.Set[ELEMENT],
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

	for element := range left.All() {
		if ok := right.ContainsKey(element.String()); !ok {
			return false
		}
	}

	return true
}

func Strings[ELEMENT interfaces.Stringer](
	collections ...interfaces.Set[ELEMENT],
) interfaces.Seq[string] {
	return func(yield func(string) bool) {
		for _, collection := range collections {
			if collection == nil {
				continue
			}

			for element := range collection.All() {
				if !yield(element.String()) {
					return
				}
			}
		}
	}
}
