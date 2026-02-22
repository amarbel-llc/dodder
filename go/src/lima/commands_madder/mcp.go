package commands_madder

import (
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/lima/mcp_madder"
)

func init() {
	utility.AddCmd("mcp", &Mcp{})
}

type Mcp struct{}

func (cmd Mcp) Run(req command.Request) {
	if err := mcp_madder.RunServer(req.Utility); err != nil {
		req.Cancel(err)
	}
}
