package commands_madder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/hotel/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

func init() {
	utility.AddCmd("pack", &Pack{})
}

type Pack struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore

	DeleteLoose bool
}

var _ interfaces.CommandComponentWriter = (*Pack)(nil)

func (cmd Pack) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	for id, blobStore := range blobStores {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

func (cmd *Pack) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(&cmd.DeleteLoose, "delete-loose", false,
		"validate archive then delete packed loose blobs")
}

func (cmd Pack) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStoreMap := cmd.MakeBlobStoresFromIdsOrAll(req, envBlobStore)

	tw := tap.NewWriter(os.Stdout)

	for storeId, blobStore := range blobStoreMap {
		packable, ok := blobStore.BlobStore.(blob_stores.PackableArchive)
		if !ok {
			tw.Skip(storeId, "not packable")
			continue
		}

		if err := packable.Pack(blob_stores.PackOptions{
			DeleteLoose:          cmd.DeleteLoose,
			DeletionPrecondition: blob_stores.NopDeletionPrecondition(),
		}); err != nil {
			tw.NotOk(
				fmt.Sprintf("pack %s", storeId),
				tap_diagnostics.FromError(err),
			)
			req.Cancel(err)
			return
		}

		tw.Ok(fmt.Sprintf("pack %s", storeId))
	}

	tw.Plan()
}
