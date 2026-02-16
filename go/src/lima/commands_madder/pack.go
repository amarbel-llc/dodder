package commands_madder

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
)

func init() {
	utility.AddCmd("pack", &Pack{})
}

type Pack struct {
	command_components_madder.EnvBlobStore

	StoreId     blob_store_id.Id
	DeleteLoose bool
}

var _ interfaces.CommandComponentWriter = (*Pack)(nil)

func (cmd *Pack) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.Var(&cmd.StoreId, "store", "inventory archive store id")
	flagSet.BoolVar(&cmd.DeleteLoose, "delete-loose", false,
		"validate archive then delete packed loose blobs")
}

func (cmd Pack) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStoreMap := envBlobStore.GetBlobStores()

	for _, blobStore := range blobStoreMap {
		if !cmd.StoreId.IsEmpty() {
			if blobStore.Path.GetId().String() != cmd.StoreId.String() {
				continue
			}
		}

		packable, ok := blobStore.BlobStore.(blob_stores.PackableArchive)
		if !ok {
			continue
		}

		if err := packable.Pack(blob_stores.PackOptions{
			DeleteLoose:          cmd.DeleteLoose,
			DeletionPrecondition: blob_stores.NopDeletionPrecondition(),
		}); err != nil {
			req.Cancel(err)
			return
		}
	}
}
