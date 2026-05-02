package command_components_madder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
)

type BlobStore struct{}

func (cmd *BlobStore) MakeBlobStoreFromIdOrConfigPath(
	envBlobStore env_repo.BlobStoreEnv,
	basePath string,
	blobStoreIndexOrConfigPath string,
) (blobStore blob_stores.BlobStoreInitialized) {
	if blobStoreIndexOrConfigPath == "" {
		goto tryDefaultBlobStore
	}

	{
		configPath := blobStoreIndexOrConfigPath
		var typedConfig blob_store_configs.TypedConfig

		{
			var err error

			if typedConfig, err = hyphence.DecodeFromFile(
				blob_store_configs.Coder,
				configPath,
			); err != nil {
				if errors.IsNotExist(err) {
					err = nil
					goto tryBlobStoreId
				} else {
					envBlobStore.Cancel(err)
					return blobStore
				}
			}
		}

		blobStore.Config = typedConfig

		blobStore.Path = directory_layout.GetBlobStorePathForCustomPath(
			blobStoreIndexOrConfigPath,
			basePath,
			blobStoreIndexOrConfigPath,
		)

		{
			var err error

			if blobStore.BlobStore, err = blob_stores.MakeBlobStore(
				envBlobStore,
				blobStore.ConfigNamed,
				nil,
			); err != nil {
				envBlobStore.Cancel(err)
				return blobStore
			}
		}

		return blobStore
	}

tryBlobStoreId:
	return cmd.MakeBlobStoreFromIdString(envBlobStore, blobStoreIndexOrConfigPath)

tryDefaultBlobStore:
	return envBlobStore.GetDefaultBlobStore()
}

func (cmd *BlobStore) MakeBlobStoreFromIdString(
	envBlobStore env_repo.BlobStoreEnv,
	blobStoreIdString string,
) (blobStore blob_stores.BlobStoreInitialized) {
	var blobStoreId blob_store_id.Id

	if err := blobStoreId.Set(blobStoreIdString); err != nil {
		envBlobStore.Cancel(err)
		return blobStore
	}

	return envBlobStore.GetBlobStore(blobStoreId)
}
