package quiter_seq

import (
	"slices"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func Seq[ELEMENT any](elements ...ELEMENT) interfaces.Seq[ELEMENT] {
	return slices.Values(elements)
}

func Any[ELEMENT any](seq interfaces.Seq[ELEMENT]) (element ELEMENT) {
	for element = range seq {
		break
	}

	return element
}

func SeqWithIndex[ELEMENT any](seq interfaces.Seq[ELEMENT]) interfaces.Seq2[int, ELEMENT] {
	return func(yield func(int, ELEMENT) bool) {
		var index int

		for element := range seq {
			if !yield(index, element) {
				return
			}

			index++
		}
	}
}

func Strings[ELEMENT interfaces.Stringer](
	seq interfaces.Seq[ELEMENT],
) interfaces.Seq[string] {
	return func(yield func(string) bool) {
		for element := range seq {
			if !yield(element.String()) {
				return
			}
		}
	}
}

func SeqErrorToSeqAndPanic[ELEMENT any](
	seq interfaces.SeqError[ELEMENT],
) interfaces.Seq[ELEMENT] {
	return func(yield func(ELEMENT) bool) {
		for element, err := range seq {
			if err != nil {
				panic(err)
			}

			if !yield(element) {
				return
			}
		}
	}
}
