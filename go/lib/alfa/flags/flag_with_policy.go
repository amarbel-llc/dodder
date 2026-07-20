package flags

import (
	"strings"

	"code.linenisgreat.com/dodder/go/lib/0/flag_policy"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
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

// setElementsFromTokens shares the tokenize -> Set -> yield loop between
// SplitCommasAndTrimAndMake and SplitSpacesAndTrimAndMake, which otherwise
// differ only in how `value` is tokenized.
func setElementsFromTokens[
	ELEMENT interfaces.Value,
	ELEMENT_PTR interfaces.ValuePtr[ELEMENT],
](tokens interfaces.Seq[string]) interfaces.SeqError[ELEMENT] {
	return func(yield func(ELEMENT, error) bool) {
		for elementString := range tokens {
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

func SplitCommasAndTrimAndMake[
	ELEMENT interfaces.Value,
	ELEMENT_PTR interfaces.ValuePtr[ELEMENT],
](value string) interfaces.SeqError[ELEMENT] {
	return setElementsFromTokens[ELEMENT, ELEMENT_PTR](SplitCommasAndTrim(value))
}

// SplitSpacesAndTrimAndMake splits on whitespace runs (trellis/RFC 0015
// conjunction terms), not commas — comma is disjunctive in trellis, so a
// comma-separated split would silently misparse a heading's AND semantics.
func SplitSpacesAndTrimAndMake[
	ELEMENT interfaces.Value,
	ELEMENT_PTR interfaces.ValuePtr[ELEMENT],
](value string) interfaces.SeqError[ELEMENT] {
	tokens := func(yield func(string) bool) {
		for _, elementString := range strings.Fields(value) {
			if !yield(elementString) {
				return
			}
		}
	}

	return setElementsFromTokens[ELEMENT, ELEMENT_PTR](tokens)
}
