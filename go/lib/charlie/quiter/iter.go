package quiter

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeSeqErrorFromSeq[ELEMENT any](
	iter interfaces.Seq[ELEMENT],
) interfaces.SeqError[ELEMENT] {
	return func(yield func(ELEMENT, error) bool) {
		for element := range iter {
			if !yield(element, nil) {
				break
			}
		}
	}
}

func MakeSeqErrorWithError[ELEMENT any](
	err error,
) interfaces.SeqError[ELEMENT] {
	return func(yield func(ELEMENT, error) bool) {
		var t ELEMENT
		yield(t, err)
	}
}

func MakeSeqErrorEmpty[ELEMENT any]() interfaces.SeqError[ELEMENT] {
	return func(yield func(ELEMENT, error) bool) {}
}

func CollectError[ELEMENT any](
	seq interfaces.SeqError[ELEMENT],
) ([]ELEMENT, error) {
	slice := make([]ELEMENT, 0)
	for element, errIter := range seq {
		if errIter != nil {
			return slice, errors.Wrap(errIter)
		}

		slice = append(slice, element)
	}

	return slice, nil
}
