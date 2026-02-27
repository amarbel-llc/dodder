package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/lima/commands_madder"
)

var utility = command.MakeUtility(
	"dodder",
	repo_config_cli.Default(),
).MergeUtilityWithPrefix(
	commands_madder.GetUtility(),
	"blob_store",
)

func GetUtility(name string) command.Utility {
	return utility
}
