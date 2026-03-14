package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/mcp_dodder"
)

func init() {
	utility.AddCmd("mcp", &Mcp{})
}

type Mcp struct{}

func (cmd Mcp) Run(req command.Request) {
	if err := mcp_dodder.RunServer(req.Utility); err != nil {
		req.Cancel(err)
	}
}
