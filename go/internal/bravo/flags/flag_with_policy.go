package flags

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/flag_policy"
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

func MakeWithPolicy(
	fp flag_policy.FlagPolicy,
	stringer func() string,
	set func(string) error,
	reset func(),
) FlagWithPolicy {
	return FlagWithPolicy{
		FlagPolicy: fp,
		stringer:   stringer,
		set:        set,
		reset:      reset,
	}
}

type FlagWithPolicy struct {
	flag_policy.FlagPolicy
	stringer func() string
	set      func(string) error
	reset    func()
}

func (flag FlagWithPolicy) Set(v string) (err error) {
	if flag.FlagPolicy == flag_policy.FlagPolicyReset {
		flag.reset()
	}

	if err = flag.set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (flag FlagWithPolicy) String() string {
	if flag.stringer == nil {
		return "nil"
	} else {
		return flag.stringer()
	}
}

func SplitCommasAndTrim(value string) interfaces.Seq[string] {
	return func(yield func(string) bool) {
		elements := strings.SplitSeq(value, ",")

		for element := range elements {
			element = strings.TrimSpace(element)

			if !yield(element) {
				return
			}
		}
	}
}

func SplitCommasAndTrimAndMake[
	ELEMENT interfaces.Value,
	ELEMENT_PTR interfaces.ValuePtr[ELEMENT],
](value string) interfaces.SeqError[ELEMENT] {
	return func(yield func(ELEMENT, error) bool) {
		elements := strings.SplitSeq(value, ",")

		for elementString := range elements {
			elementString = strings.TrimSpace(elementString)

			var element ELEMENT

			if err := ELEMENT_PTR(&element).Set(elementString); err != nil {
				if !yield(element, err) {
					return
				}

				continue
			}

			if !yield(element, nil) {
				return
			}
		}
	}
}
