package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func init() {
	bigBang := env_repo.BigBang{}
	bigBang.SetDefaults()

	utility.AddCmd("init", &Init{
		Genesis: command_components_dodder.Genesis{
			BigBang: bigBang,
		},
	})
}

type Init struct {
	command_components_dodder.Genesis
}

var (
	_ interfaces.CommandComponentWriter = (*Init)(nil)
	_ command.CommandWithArgs           = (*Init)(nil)
)

func (cmd *Init) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "repo-id",
			Description: "identifier for the new repository",
			Required:    true,
		}},
	}}
}

func (cmd Init) GetDescription() command.Description {
	return command.Description{
		Short: "initialize a new repository",
		Long: "Create a new dodder repository in the current directory. " +
			"Generates cryptographic keys for signing, initializes the " +
			"default blob store, and sets up the object index. The " +
			"repo-id argument names the repository (used for remote " +
			"synchronization).",
	}
}

func (cmd *Init) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.Genesis.SetFlagDefinitions(flagSet)
}

func (cmd *Init) Run(req command.Request) {
	repoId := req.PopArg("repo-id")
	req.AssertNoMoreArgs()

	cmd.OnTheFirstDay(req, repoId)
}
