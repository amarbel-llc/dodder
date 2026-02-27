package quiter

import (
	"iter"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func Pull[ELEMENT any](
	seq interfaces.Seq[ELEMENT],
) (func() (ELEMENT, bool), func()) {
	return iter.Pull(seq)
}

func Pull2[ELEMENT_ONE any, ELEMENT_TWO any](
	seq interfaces.Seq2[ELEMENT_ONE, ELEMENT_TWO],
) (func() (ELEMENT_ONE, ELEMENT_TWO, bool), func()) {
	return iter.Pull2(seq)
}

func PullError[ELEMENT any](
	seq interfaces.SeqError[ELEMENT],
) (func() (ELEMENT, error, bool), func()) {
	return iter.Pull2(seq)
}

func PullErrorWithoutOk[ELEMENT any, ELEMENT_PTR interfaces.Ptr[ELEMENT]](
	seq interfaces.SeqError[ELEMENT_PTR],
) (func() (ELEMENT_PTR, error), func()) {
	iter, stop := iter.Pull2(seq)

	return func() (ELEMENT_PTR, error) {
		element, err, _ := iter()
		return element, err
	}, stop
}
