package commands_madder

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
)

func init() {
	utility.AddCmd("pack-list", &PackList{})
}

type PackList struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore
}

var _ command.CommandWithArgs = (*PackList)(nil)

func (cmd *PackList) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "store-ids",
			Description: "blob store IDs to list packs from (defaults to all)",
			Variadic:    true,
		}},
	}}
}

func (cmd PackList) GetDescription() command.Description {
	return command.Description{
		Short: "list archive files in inventory archive stores",
		Long: "List the archive pack files in one or more inventory archive " +
			"stores, showing each archive's checksum and the number of " +
			"blobs it contains. With no arguments, lists archives from " +
			"all packable stores.",
	}
}

func (cmd PackList) Complete(
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

func (cmd PackList) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStoreMap := cmd.MakeBlobStoresFromIdsOrAll(req, envBlobStore)

	for _, blobStore := range blobStoreMap {
		archiveIndex, ok := blobStore.BlobStore.(blob_stores.ArchiveIndex)
		if !ok {
			continue
		}

		entries := archiveIndex.AllArchiveEntryChecksums()

		checksums := make([]string, 0, len(entries))
		for checksum := range entries {
			checksums = append(checksums, checksum)
		}
		sort.Strings(checksums)

		for _, checksum := range checksums {
			envBlobStore.GetUI().Printf(
				"%s: %d entries",
				checksum,
				len(entries[checksum]),
			)
		}
	}
}
