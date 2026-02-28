package commands_madder

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("cat-ids", &CatIds{})
}

type CatIds struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore
}

func (cmd CatIds) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	// args := commandLine.FlagsOrArgs[1:]

	// if commandLine.InProgress != "" {
	// 	args = args[:len(args)-1]
	// }

	for id, blobStore := range blobStores {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

func (cmd CatIds) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)

	blobStores := cmd.MakeBlobStoresFromIdsOrAll(req, envBlobStore)

	var blobErrors collections_slice.Slice[command_components_madder.BlobError]

	for _, blobStore := range blobStores {
		cmd.runOne(envBlobStore, blobStore, &blobErrors)
	}

	command_components_madder.PrintBlobErrors(envBlobStore, blobErrors)
}

func (cmd CatIds) runOne(
	envBlobStore env_repo.BlobStoreEnv,
	blobStore blob_stores.BlobStoreInitialized,
	blobErrors *collections_slice.Slice[command_components_madder.BlobError],
) {
	for id, err := range blobStore.AllBlobs() {
		errors.ContextContinueOrPanic(envBlobStore)

		if err != nil {
			blobErrors.Append(
				command_components_madder.BlobError{BlobId: id, Err: err},
			)
		} else {
			envBlobStore.GetUI().Print(id)
		}
	}
}
