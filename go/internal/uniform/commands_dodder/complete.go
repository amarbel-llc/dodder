package commands_dodder

import (
	"io"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/command_components"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/flags"
	env_local "code.linenisgreat.com/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"complete",
		&Complete{},
	)
}

type Complete struct {
	command_components.Env
	command_components_dodder.Complete

	bashStyle  bool
	inProgress string
}

var (
	_ interfaces.CommandComponentWriter = (*Complete)(nil)
	_ command.CommandWithArgs           = (*Complete)(nil)
)

func (cmd *Complete) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "name",
				Description: "subcommand name to complete",
				Required:    true,
			},
			{
				Name:        "args",
				Description: "remaining arguments for completion context",
				Variadic:    true,
			},
		},
	}}
}

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

func (cmd Complete) Run(req command.Request) {
	utility := req.Utility
	envLocal := cmd.MakeEnv(req)

	// TODO extract into constructor
	// TODO find double-hyphen
	// TODO keep track of all args
	commandLine := command.CommandLineInput{
		FlagsOrArgs: req.PeekArgs(),
		InProgress:  cmd.inProgress,
	}

	// TODO determine state:
	// bare: `dodder`
	// subcommand or arg or flag:
	//  - `dodder subcommand`
	//  - `dodder subcommand -flag=true`
	//  - `dodder subcommand -flag value`
	// flag: `dodder subcommand -flag`
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
	(&repo_config_cli.Config{}).SetFlagDefinitions(flagSet)

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

	// -repo_id's value is madder's scoped_id.Id (no methods we can add) and
	// the flag is registered in charlie-tier repo_config_cli (can't import
	// delta/command to wrap it as a FlagValueCompleter), so complete it by
	// flag name here: offer the repos present in the active scope (FDR-0019).
	if after == "repo_id" {
		cmd.completeRepoId(req, envLocal)
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

// completeRepoId offers the repos addressable from here as -repo_id
// candidates across both scopes, spelled `.name` for a cwd-scope repo and
// `name` for an XDG-user repo so the candidate is directly usable. The
// deferred grammar (//system, ..multi-dot) is intentionally not offered —
// repo_id.CheckSupported rejects it (FDR-0019).
func (cmd Complete) completeRepoId(
	req command.Request,
	envLocal env_local.Env,
) {
	repos, err := listScopedRepos(req)
	if err != nil {
		envLocal.Cancel(err)
		return
	}

	for _, repo := range repos {
		envLocal.GetUI().Printf("%s\t%s", repo.Spelling(), repo.ScopeLabel())
	}
}
