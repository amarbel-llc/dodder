package commands_madder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/0/interfaces"
)

func init() {
	utility.AddCmd("list", &List{})
}

type List struct {
	command_components_madder.EnvBlobStore
}

var (
	_ interfaces.CommandComponentWriter = (*List)(nil)
	_ command.CommandWithArgs           = (*List)(nil)
)

func (cmd *List) GetArgs() []command.ArgGroup { return nil }

func (cmd List) GetDescription() command.Description {
	return command.Description{
		Short: "list configured blob stores",
		Long: "List all blob stores configured for the current repository, " +
			"showing each store's ID and description. Store IDs use prefixes " +
			"to indicate scope: unprefixed for XDG user stores, '.' for " +
			"CWD-relative stores, and '/' for XDG system stores.",
	}
}

func (cmd *List) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
}

func (cmd List) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	for _, blobStore := range blobStores {
		envBlobStore.GetUI().Printf(
			"%s: %s",
			blobStore.Path.GetId(),
			blobStore.GetBlobStoreDescription(),
		)
	}
}
