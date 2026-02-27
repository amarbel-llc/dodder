package commands_madder

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/flags"
	"code.linenisgreat.com/dodder/go/lib/delta/debug"
	"code.linenisgreat.com/dodder/go/lib/echo/config_cli"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
)

func init() {
	utility.AddCmd(
		"complete",
		&Complete{})
}

type Complete struct {
	bashStyle  bool
	inProgress string
}

var _ interfaces.CommandComponentWriter = (*Complete)(nil)

func (cmd Complete) GetDescription() command.Description {
	return command.Description{
		Short: "complete a command-line",
	}
}

func (cmd *Complete) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	flagDefinitions.BoolVar(&cmd.bashStyle, "bash-style", false, "")
	flagDefinitions.StringVar(&cmd.inProgress, "in-progress", "", "")
}

func (cmd Complete) makeEnv(req command.Request) env_local.Env {
	configAny := req.Utility.GetConfigAny()

	var debugOptions debug.Options
	var cliConfig domain_interfaces.CLIConfigProvider

	switch c := configAny.(type) {
	case *config_cli.Config:
		debugOptions = c.Debug
		cliConfig = c
	case *repo_config_cli.Config:
		debugOptions = c.Debug
		cliConfig = c
	default:
		panic(fmt.Sprintf("unsupported config type: %T", configAny))
	}

	dir := env_dir.MakeDefault(
		req,
		req.Utility.GetName(),
		debugOptions,
	)

	return env_local.Make(
		env_ui.Make(
			req,
			cliConfig,
			debugOptions,
			env_ui.Options{},
		),
		dir,
	)
}

func (cmd Complete) Run(req command.Request) {
	utility := req.Utility
	envLocal := cmd.makeEnv(req)

	// TODO extract into constructor
	// TODO find double-hyphen
	// TODO keep track of all args
	commandLine := command.CommandLineInput{
		FlagsOrArgs: req.PeekArgs(),
		InProgress:  cmd.inProgress,
	}

	// TODO determine state:
	// bare: `madder`
	// subcommand or arg or flag:
	//  - `madder subcommand`
	//  - `madder subcommand -flag=true`
	//  - `madder subcommand -flag value`
	// flag: `madder subcommand -flag`
	lastArg, hasLastArg := commandLine.LastArg()

	if !hasLastArg {
		cmd.completeSubcommands(envLocal, commandLine, utility)
		return
	}

	name := req.PopArg("name")
	subcmd, foundSubcmd := utility.GetCmd(name)

	if !foundSubcmd {
		cmd.completeSubcommands(envLocal, commandLine, utility)
		return
	}

	flagSet := flags.NewFlagSet(name, flags.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	(&config_cli.Config{}).SetFlagDefinitions(flagSet)

	if subcmd, ok := subcmd.(interfaces.CommandComponentWriter); ok {
		subcmd.SetFlagDefinitions(flagSet)
	}

	var containsDoubleHyphen bool

	if slices.Contains(commandLine.FlagsOrArgs, "--") {
		containsDoubleHyphen = true
	}

	if !containsDoubleHyphen &&
		cmd.completeSubcommandFlags(
			req,
			envLocal,
			subcmd,
			flagSet,
			commandLine,
			lastArg,
		) {
		return
	}

	cmd.completeSubcommandArgs(req, envLocal, subcmd, commandLine)
}

func (cmd Complete) completeSubcommands(
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
	utility command.Utility,
) {
	for name, subcmd := range utility.AllCmds() {
		cmd.completeSubcommand(envLocal, name, subcmd)
	}
}

func (cmd Complete) completeSubcommand(
	envLocal env_local.Env,
	name string,
	subcmd command.Cmd,
) {
	var shortDescription string

	if hasDescription, ok := subcmd.(command.CommandWithDescription); ok {
		description := hasDescription.GetDescription()
		shortDescription = description.Short
	}

	if shortDescription != "" {
		envLocal.GetUI().Printf("%s\t%s", name, shortDescription)
	} else {
		envLocal.GetUI().Printf("%s", name)
	}
}

func (cmd Complete) completeSubcommandArgs(
	req command.Request,
	envLocal env_local.Env,
	subcmd command.Cmd,
	commandLine command.CommandLineInput,
) {
	if subcmd == nil {
		return
	}

	completer, isCompleter := subcmd.(command.Completer)

	if !isCompleter {
		return
	}

	completer.Complete(req, envLocal, commandLine)
}

func (cmd Complete) completeSubcommandFlags(
	req command.Request,
	envLocal env_local.Env,
	subcmd command.Cmd,
	flagSet *flags.FlagSet, commandLine command.CommandLineInput,
	lastArg string,
) (shouldNotCompleteArgs bool) {
	if subcmd == nil {
		return shouldNotCompleteArgs
	}

	if strings.HasPrefix(lastArg, "-") && commandLine.InProgress != "" {
		shouldNotCompleteArgs = true
	} else if commandLine.InProgress != "" && len(commandLine.FlagsOrArgs) > 1 {
		lastArg = commandLine.FlagsOrArgs[len(commandLine.FlagsOrArgs)-2]
		commandLine.InProgress = ""
		shouldNotCompleteArgs = strings.HasPrefix(lastArg, "-")
	}

	if commandLine.InProgress != "" {
		flagSet.VisitAll(func(flag *flags.Flag) {
			envLocal.GetUI().Printf("-%s\t%s", flag.Name, flag.Usage)
		})
	} else if err := flagSet.Parse([]string{lastArg}); err != nil {
		cmd.completeSubcommandFlagOnParseError(
			req,
			envLocal,
			subcmd,
			flagSet,
			commandLine,
			err,
		)
	} else {
		flagSet.VisitAll(func(flag *flags.Flag) {
			envLocal.GetUI().Printf("-%s\t%s", flag.Name, flag.Usage)
		})
	}

	return shouldNotCompleteArgs
}

func (cmd Complete) completeSubcommandFlagOnParseError(
	req command.Request,
	envLocal env_local.Env,
	subcmd command.Cmd,
	flagSet *flags.FlagSet,
	commandLine command.CommandLineInput,
	err error,
) {
	if subcmd == nil {
		return
	}

	after, found := strings.CutPrefix(
		err.Error(),
		"flag needs an argument: -",
	)

	if !found {
		errors.ContextCancelWithBadRequestError(envLocal, err)
		return
	}

	var flag *flags.Flag

	if flag = flagSet.Lookup(after); flag == nil {
		// exception
		errors.ContextCancelWithErrorf(
			envLocal,
			"expected to find flag %q, but none found. All flags: %#v",
			after,
			flagSet,
		)

		return
	}

	flagValue := flag.Value

	switch flagValue := flagValue.(type) {
	case interface{ GetCLICompletion() map[string]string }:
		completions := flagValue.GetCLICompletion()

		for name, description := range completions {
			if name != "" && description != "" {
				envLocal.GetUI().Printf("%s\t%s", name, description)
			} else if description == "" {
				envLocal.GetUI().Printf("%s", name)
			} else {
				envLocal.GetErr().Printf("empty flag value for %s (description: %q)", flag.Name, description)
			}
		}

	case command.Completer:
		flagValue.Complete(req, envLocal, commandLine)

	default:
		errors.ContextCancelWithBadRequestf(
			req,
			"no completion available for flag: %q. Flag Value: %T, *flag.Flag %#v",
			after,
			flagValue,
			flag,
		)
	}
}
