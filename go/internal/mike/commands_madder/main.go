package commands_madder

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/lib/foxtrot/config_cli"
)

var utility = command.MakeUtility("madder", config_cli.Default())

func GetUtility() command.Utility {
	return utility
}
