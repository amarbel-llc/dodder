package commands_madder

import (
	"code.linenisgreat.com/dodder/go/lib/echo/config_cli"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
)

var utility = command.MakeUtility("madder", config_cli.Default())

func GetUtility() command.Utility {
	return utility
}
