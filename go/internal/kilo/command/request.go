package command

import (
	"slices"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/flags"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
)

// TODO refactor this to have a generic config field and for the commands_madder
// and commands_dodder packages to alias a concrete version for their own use
type Request struct {
	errors.Context
	Utility Utility
	FlagSet *flags.FlagSet

	input *CommandLineInput
}

// TODO switch to ActiveContext
func (req Request) PeekArgs() collections_slice.String {
	return req.input.Args.Shift(req.input.Argi)
}

func (req Request) PopArgs() []string {
	args := req.PeekArgs()

	for _, arg := range args {
		req.input.consumed.Append(consumedArg{value: arg})
	}

	req.input.Argi += len(args)
	return args
}

func (req Request) PopArgsAsMutableSet() collections_value.MutableSet[string] {
	args := req.PeekArgs()
	set := collections_value.MakeMutableSet(
		quiter.StringKeyer,
		len(args),
		slices.Values(args),
	)

	for _, arg := range args {
		req.input.consumed.Append(consumedArg{value: arg})
	}

	req.input.Argi += len(args)
	return set
}

func (req Request) RemainingArgCount() int {
	return req.input.Args.Shift(req.input.Argi).Len()
}

func PopRequestArgs[
	VALUE interfaces.Stringer,
	VALUE_PTR interfaces.StringerSetterPtr[VALUE],
](
	req Request,
	name string,
) interfaces.Seq[VALUE_PTR] {
	return func(yield func(VALUE_PTR) bool) {
		for req.RemainingArgCount() > 0 {
			value := PopRequestArg[VALUE, VALUE_PTR](req, name)

			if !yield(value) {
				return
			}
		}
	}
}

func PopRequestArg[
	VALUE interfaces.Stringer,
	VALUE_PTR interfaces.StringerSetterPtr[VALUE],
](req Request, name string) VALUE_PTR {
	var value VALUE

	PopRequestArgTo(req, name, VALUE_PTR(&value))

	return &value
}

func PopRequestArgTo(req Request, name string, value interfaces.StringerSetter) {
	PopRequestArgToFunc(req, name, value.Set)
}

func PopRequestArgToFunc(req Request, name string, funcSet func(string) error) {
	arg := req.PopArg(name)

	if err := funcSet(arg); err != nil {
		errors.ContextCancelWithBadRequestError(req, err)
	}
}

func (req Request) PopArg(name string) string {
	if req.RemainingArgCount() == 0 {
		errors.ContextCancelWithBadRequestf(
			req,
			"expected positional argument (%d) %s, but only received %q",
			req.input.Argi+1,
			name,
			req.input.consumed,
		)
	}

	value := req.input.Args.At(req.input.Argi)
	req.input.consumed.Append(consumedArg{name: name, value: value})
	req.input.Argi++
	return value
}

func (req Request) PopArgOrDefault(name, defaultArg string) string {
	if req.RemainingArgCount() == 0 {
		return defaultArg
	}

	value := req.input.Args.At(req.input.Argi)
	req.input.consumed.Append(consumedArg{name: name, value: value})
	req.input.Argi++

	return value
}

func (req *Request) AssertNoMoreArgs() {
	if req.RemainingArgCount() > 0 {
		errors.ContextCancelWithBadRequestf(
			req,
			"expected no more arguments, but have %q",
			req.PopArgs(),
		)
	}
}

func (req Request) LastArg() (arg string, ok bool) {
	if req.RemainingArgCount() > 0 {
		ok = true
		arg = req.PopArgs()[req.RemainingArgCount()-1]
	}

	return arg, ok
}
