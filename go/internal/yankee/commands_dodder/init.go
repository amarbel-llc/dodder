package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
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

var _ interfaces.CommandComponentWriter = (*Init)(nil)

func (cmd *Init) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.Genesis.SetFlagDefinitions(flagSet)
}

func (cmd *Init) Run(req command.Request) {
	repoId := req.PopArg("repo-id")
	req.AssertNoMoreArgs()

	cmd.OnTheFirstDay(req, repoId)
}
