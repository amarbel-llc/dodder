package command

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/cli"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
)

type SupportsCompletion interface {
	SupportsCompletion()
}

type CLICompleter = cli.CLICompleter

type Completion struct {
	Value, Description string
}

type Completer interface {
	Complete(Request, env_local.Env, CommandLineInput)
}

type FuncCompleter func(Request, env_local.Env, CommandLineInput)

type FlagValueCompleter struct {
	interfaces.FlagValue
	FuncCompleter
}

func (completer FlagValueCompleter) String() string {
	// TODO still not sure why this condition can exist, but this makes the
	// output
	// nice
	if completer.FlagValue == nil {
		return ""
	} else {
		return completer.FlagValue.String()
	}
}

func (completer FlagValueCompleter) Complete(
	req Request,
	envLocal env_local.Env,
	commandLine CommandLineInput,
) {
	completer.FuncCompleter(req, envLocal, commandLine)
}
