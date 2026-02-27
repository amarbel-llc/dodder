package quiter

import (
	"iter"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

// merges the two sorted sequences into a single output sequence. when `funcCmp`
// yields cmp.Equal for two elements, the left element is yielded to the output
// sequence.
func MergeSeqErrorLeft[
	ELEMENT any,
	ELEMENT_PTR interfaces.Ptr[ELEMENT],
](
	seqLeft, seqRight interfaces.SeqError[ELEMENT_PTR],
	funcCmp func(ELEMENT_PTR, ELEMENT_PTR) cmp.Result,
) interfaces.SeqError[ELEMENT_PTR] {
	return func(yield func(ELEMENT_PTR, error) bool) {
		pullLeft, stopLeft := iter.Pull2(seqLeft)
		defer stopLeft()

		pullRight, stopRight := iter.Pull2(seqRight)
		defer stopRight()

		seq := MergeStreamPreferringLeft(
			pullLeft,
			pullRight,
			funcCmp,
		)

		for element, errIter := range seq {
			if errIter != nil {
				yield(nil, errors.Wrap(errIter))
				return
			}

			if !yield(element, nil) {
				return
			}
		}
	}
}

func MergeStreamPreferringLeft[
	ELEMENT any,
	ELEMENT_PTR interfaces.Ptr[ELEMENT],
](
	pullLeft, pullRight interfaces.Pull2[ELEMENT_PTR, error],
	funcCmp func(ELEMENT_PTR, ELEMENT_PTR) cmp.Result,
) interfaces.SeqError[ELEMENT_PTR] {
	return func(yield func(ELEMENT_PTR, error) bool) {
		type readWithElement struct {
			read  func() (ELEMENT_PTR, error, bool)
			carry ELEMENT_PTR
		}

		left := readWithElement{
			read: pullLeft,
		}

		right := readWithElement{
			read: pullRight,
		}

		var remaining readWithElement

		for {
			if right.carry == nil {
				var err error
				var ok bool
				if right.carry, err, ok = right.read(); !ok {
					remaining = left
					goto EMIT_REMAINING
				} else if err != nil {
					yield(nil, err)
					return
				}

				if right.carry == nil {
					remaining = left
					break
				}
			}

			if left.carry == nil {
				var err error
				var ok bool
				if left.carry, err, ok = left.read(); !ok {
					remaining = right
					goto EMIT_REMAINING
				} else if err != nil {
					yield(nil, err)
					return
				}

				if left.carry == nil {
					remaining = right
					break
				}
			}

			cmpResult := funcCmp(left.carry, right.carry)

			switch cmpResult {
			case cmp.Equal:
				right.carry = nil
				fallthrough

			case cmp.Less:
				if !yield(left.carry, nil) {
					return
				}

				left.carry = nil

			case cmp.Greater:
				if !yield(right.carry, nil) {
					return
				}

				right.carry = nil
			}
		}

	EMIT_REMAINING:

		if remaining.carry != nil {
			if !yield(remaining.carry, nil) {
				return
			}
		}

		for {
			var element ELEMENT_PTR
			var err error
			var ok bool

			if element, err, ok = remaining.read(); !ok {
				return
			} else if err != nil {
				err = errors.Wrap(err)
				yield(nil, err)
				return
			}

			if !yield(element, nil) {
				return
			}
		}
	}
}
