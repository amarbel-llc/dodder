package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/internal/tango/mcp_dodder"
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
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	repo := cmd.MakeLocalWorkingCopy(req)

	// The MCP server starts cleanly whether or not the CWD is a dodder
	// workspace. mcp_dodder.RunServer inspects the workspace env and
	// advertises only the tools that make sense in the current mode (see
	// github.com/amarbel-llc/dodder/issues/116). config.RepoId is the
	// startup repo: explicit -repo_id pins the server to it; auto/default
	// lets each tool call select a repo via the repo_id param (FDR-0019).
	if err := mcp_dodder.RunServer(req.Utility, repo, config.RepoId); err != nil {
		req.Cancel(err)
	}
}
