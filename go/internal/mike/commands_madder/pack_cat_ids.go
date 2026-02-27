package commands_madder

import (
	"code.linenisgreat.com/dodder/go/internal/juliett/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/internal/lima/command_components_madder"
)

func init() {
	utility.AddCmd("pack-cat-ids", &PackCatIds{})
}

type PackCatIds struct {
	command_components_madder.EnvBlobStore
}

func (cmd PackCatIds) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStoreMap := envBlobStore.GetBlobStores()

	archiveFilter := make(map[string]struct{})
	for _, arg := range req.PopArgs() {
		archiveFilter[arg] = struct{}{}
	}

	for _, blobStore := range blobStoreMap {
		cmd.runOne(envBlobStore, blobStore, archiveFilter)
	}
}

func (cmd PackCatIds) runOne(
	envBlobStore env_repo.BlobStoreEnv,
	blobStore blob_stores.BlobStoreInitialized,
	archiveFilter map[string]struct{},
) {
	archiveIndex, ok := blobStore.BlobStore.(blob_stores.ArchiveIndex)
	if !ok {
		return
	}

	entries := archiveIndex.AllArchiveEntryChecksums()

	for checksum, blobIds := range entries {
		if len(archiveFilter) > 0 {
			if _, ok := archiveFilter[checksum]; !ok {
				continue
			}
		}

		for _, blobId := range blobIds {
			envBlobStore.GetUI().Print(blobId)
		}
	}
}
