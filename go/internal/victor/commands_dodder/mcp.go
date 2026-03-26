package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/internal/uniform/mcp_dodder"
)

func init() {
	utility.AddCmd("mcp", &Mcp{})
}

type Mcp struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd Mcp) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)
	envWorkspace := repo.GetEnvWorkspace()
	envWorkspace.AssertNotTemporary(repo)

	if err := mcp_dodder.RunServer(req.Utility, repo); err != nil {
		req.Cancel(err)
	}
}
