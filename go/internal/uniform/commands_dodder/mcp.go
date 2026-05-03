package commands_dodder

import (
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
	repo := cmd.MakeLocalWorkingCopy(req)

	// The MCP server starts cleanly whether or not the CWD is a dodder
	// workspace. mcp_dodder.RunServer inspects the workspace env and
	// advertises only the tools that make sense in the current mode (see
	// github.com/amarbel-llc/dodder/issues/116).
	if err := mcp_dodder.RunServer(req.Utility, repo); err != nil {
		req.Cancel(err)
	}
}
