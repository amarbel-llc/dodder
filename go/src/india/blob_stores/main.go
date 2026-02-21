package blob_stores

import (
	"fmt"
	"maps"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"golang.org/x/crypto/ssh"
)

var defaultBuckets = []int{2}

type BlobStoreMap = map[string]BlobStoreInitialized

func MakeBlobStoreMap(blobStores ...BlobStoreInitialized) BlobStoreMap {
	output := make(BlobStoreMap, len(blobStores))

	for _, blobStore := range blobStores {
		blobStoreIdString := blobStore.Path.GetId().String()
		output[blobStoreIdString] = blobStore
	}

	return output
}

func makeBlobStoreConfigs(
	ctx interfaces.ActiveContext,
	directoryLayout directory_layout.BlobStore,
) BlobStoreMap {
	configPaths := directory_layout.GetBlobStoreConfigPaths(
		ctx,
		directoryLayout,
	)

	blobStores := make(BlobStoreMap, len(configPaths))

	for _, configPath := range configPaths {
		blobStorePath := directory_layout.GetBlobStorePath(
			directoryLayout,
			fd.DirBaseOnly(configPath),
		)

		blobStoreIdString := blobStorePath.GetId().String()
		blobStore := blobStores[blobStoreIdString]
		blobStore.Path = blobStorePath

		if typedConfig, err := triple_hyphen_io.DecodeFromFile(
			blob_store_configs.Coder,
			configPath,
		); err != nil {
			ctx.Cancel(err)
			return blobStores
		} else {
			blobStore.Config = typedConfig
		}

		blobStores[blobStoreIdString] = blobStore
	}

	return blobStores
}

// TODO pass in custom UI context for printing
// TODO consolidated envDir and ctx arguments
func MakeBlobStores(
	ctx interfaces.ActiveContext,
	envDir env_dir.Env,
	directoryLayout directory_layout.BlobStore,
) (blobStores BlobStoreMap) {
	// based on explicit xdg (that is, may include override)
	blobStores = makeBlobStoreConfigs(ctx, directoryLayout)

	// If we're in an override directory, add the User blob stores
	if envDir.GetXDG().GetLocationType() == blob_store_id.LocationTypeCwd {
		if directoryLayoutForUser, err := directory_layout.CloneBlobStoreWithXDG(
			directoryLayout,
			envDir.GetXDGForBlobStores().CloneWithoutOverride(),
		); err != nil {
			ctx.Cancel(err)
			return blobStores
		} else {
			blobStoresForXDG := makeBlobStoreConfigs(ctx, directoryLayoutForUser)
			maps.Insert(blobStores, maps.All(blobStoresForXDG))
		}
	}

	for blobStoreIdString := range blobStores {
		blobStore := blobStores[blobStoreIdString]
		var err error

		if blobStore.BlobStore, err = MakeBlobStore(
			envDir,
			blobStore.ConfigNamed,
			blobStores,
		); err != nil {
			ctx.Cancel(err)
			return blobStores
		}

		blobStores[blobStoreIdString] = blobStore
	}

	return blobStores
}

func MakeRemoteBlobStore(
	envDir env_dir.Env,
	configNamed blob_store_configs.ConfigNamed,
) (blobStore BlobStoreInitialized) {
	blobStore.ConfigNamed = configNamed

	{
		var err error

		if blobStore.BlobStore, err = MakeBlobStore(
			envDir,
			configNamed,
			nil,
		); err != nil {
			envDir.GetActiveContext().Cancel(err)
			return blobStore
		}
	}

	return blobStore
}

// NOTE: blobStores parameter added to support inventory archive's
// loose-blob-store-id resolution. This couples MakeBlobStore to the
// store map, which may not scale well if more store types need
// cross-references. If this becomes a problem, switch to two-pass
// initialization: first pass creates all stores without cross-refs,
// second pass wires them up.
//
// TODO describe base path agnostically
func MakeBlobStore(
	envDir env_dir.Env,
	configNamed blob_store_configs.ConfigNamed,
	blobStores BlobStoreMap,
) (store domain_interfaces.BlobStore, err error) {
	printer := ui.MakePrefixPrinter(
		ui.Err(),
		fmt.Sprintf("(blob_store: %s) ", configNamed.Path.GetId()),
	)

	configBlob := configNamed.Config.Blob

	switch config := configBlob.(type) {
	case blob_store_configs.ConfigSFTPUri:
		return makeSftpStore(
			envDir.GetActiveContext(),
			printer,
			config,
			func() (*ssh.Client, error) {
				return MakeSSHClientFromSSHConfig(
					envDir.GetActiveContext(),
					printer,
					config,
				)
			},
		)

	case blob_store_configs.ConfigSFTPConfigExplicit:
		return makeSftpStore(
			envDir.GetActiveContext(),
			printer,
			config,
			func() (*ssh.Client, error) {
				return MakeSSHClientForExplicitConfig(
					envDir.GetActiveContext(),
					printer,
					config,
				)
			},
		)

	case blob_store_configs.ConfigLocalHashBucketed:
		return makeLocalHashBucketed(
			envDir,
			configNamed.Path.GetBase(),
			config,
		)

	case blob_store_configs.ConfigInventoryArchiveDelta:
		var looseBlobStore domain_interfaces.BlobStore

		if blobStores != nil {
			looseBlobStoreId := config.GetLooseBlobStoreId().String()
			if initialized, ok := blobStores[looseBlobStoreId]; ok {
				looseBlobStore = initialized.BlobStore
			}
		}

		if looseBlobStore == nil {
			err = errors.BadRequestf(
				"inventory archive store requires loose-blob-store-id %q but it was not found",
				config.GetLooseBlobStoreId(),
			)
			return store, err
		}

		// TODO: implement v1 inventory archive store
		// For now, fall through to v0 behavior by treating it as v0
		return makeInventoryArchiveV0(
			envDir,
			configNamed.Path.GetBase(),
			config,
			looseBlobStore,
		)

	case blob_store_configs.ConfigInventoryArchive:
		var looseBlobStore domain_interfaces.BlobStore

		if blobStores != nil {
			looseBlobStoreId := config.GetLooseBlobStoreId().String()
			if initialized, ok := blobStores[looseBlobStoreId]; ok {
				looseBlobStore = initialized.BlobStore
			}
		}

		if looseBlobStore == nil {
			err = errors.BadRequestf(
				"inventory archive store requires loose-blob-store-id %q but it was not found",
				config.GetLooseBlobStoreId(),
			)
			return store, err
		}

		return makeInventoryArchiveV0(
			envDir,
			configNamed.Path.GetBase(),
			config,
			looseBlobStore,
		)

	case blob_store_configs.ConfigPointer:
		var typedConfig triple_hyphen_io.TypedBlob[blob_store_configs.Config]
		otherStorePath := config.GetPath()

		if typedConfig, err = triple_hyphen_io.DecodeFromFile(
			blob_store_configs.Coder,
			otherStorePath.GetConfig(),
		); err != nil {
			err = errors.Wrap(err)
			return store, err
		}

		configNamed.Config = typedConfig
		configNamed.Path = directory_layout.MakeBlobStorePath(
			configNamed.Path.GetId(),
			otherStorePath.GetBase(),
			otherStorePath.GetConfig(),
		)

		return MakeBlobStore(envDir, configNamed, blobStores)

	default:
		err = errors.BadRequestf(
			"unsupported blob store type %q:%T",
			configBlob.GetBlobStoreType(),
			configBlob,
		)

		return store, err
	}
}
