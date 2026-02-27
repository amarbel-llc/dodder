package command

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/collections_slice"
)

// TODO complete merging Args, consumed and FlagsOrArgs for use by Run/Complete
type CommandLineInput struct {
	FlagsOrArgs          collections_slice.String
	InProgress           string
	ContainsDoubleHyphen bool

	Args collections_slice.String
	Argi int

	consumed collections_slice.Slice[consumedArg]
}

type consumedArg struct {
	name, value string
}

func (arg consumedArg) String() string {
	if arg.name == "" {
		return fmt.Sprintf("%q", arg.value)
	} else {
		return fmt.Sprintf("%s:%q", arg.name, arg.value)
	}
}

func (commandLine CommandLineInput) LastArg() (arg string, ok bool) {
	argc := commandLine.FlagsOrArgs.Len()

	if argc > 0 {
		ok = true
		arg = commandLine.FlagsOrArgs.Last()
	}

	return arg, ok
}

func (commandLine CommandLineInput) LastCompleteArg() (arg string, ok bool) {
	argc := commandLine.FlagsOrArgs.Len()

	if commandLine.InProgress != "" {
		argc -= 1
	}

	if argc > 0 {
		ok = true
		arg = commandLine.FlagsOrArgs.Last()
	}

	return arg, ok
}
