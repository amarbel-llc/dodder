package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/internal/uniform/mcp_dodder"
)

func init() {
	utility.AddCmd("mcp", &Mcp{})
}

func (cmd Mcp) GetDescription() command.Description {
	return command.Description{
		Short: "start the MCP server",
	}
}

type Mcp struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*Mcp)(nil)

// GetArgs returns nil: no positional arguments.
func (cmd *Mcp) GetArgs() []command.ArgGroup { return nil }

func (cmd Mcp) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)
	envWorkspace := repo.GetEnvWorkspace()
	envWorkspace.AssertNotTemporary(repo)

	if err := mcp_dodder.RunServer(req.Utility, repo); err != nil {
		req.Cancel(err)
	}
}
