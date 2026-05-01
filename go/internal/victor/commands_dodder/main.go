package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
)

var utility = command.MakeUtility(
	"dodder",
	repo_config_cli.Default(),
)

func GetUtility(name string) command.Utility {
	return utility
}
