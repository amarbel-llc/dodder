package commands_madder

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
)

func init() {
	utility.AddCmd("list", &List{})
}

type List struct {
	command_components_madder.EnvBlobStore
}

var _ interfaces.CommandComponentWriter = (*List)(nil)

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
