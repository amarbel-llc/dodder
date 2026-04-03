package commands_madder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	gomcp_command "github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func init() {
	utility.AddCmd("install-mcp", &InstallMcp{})
}

type InstallMcp struct{}

func (cmd InstallMcp) GetDescription() command.Description {
	return command.Description{
		Short: "install MCP server configuration",
	}
}

func (cmd InstallMcp) Run(req command.Request) {
	app := gomcp_command.NewApp("madder", "Blob store MCP server")
	app.MCPArgs = []string{"mcp"}

	if err := app.InstallMCP(); err != nil {
		req.Cancel(err)
	}
}
