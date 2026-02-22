package env_repo

import (
	"maps"
	"path/filepath"
	"slices"
	"sort"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/charlie/store_version"
	"code.linenisgreat.com/dodder/go/src/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
)

type BlobStoreEnv struct {
	directory_layout.BlobStore
	env_local.Env

	defaultBlobStoreIdString string

	// TODO switch to implementing LocalBlobStore directly and writing to all of
	// the defined blob stores instead of having a default
	// TODO switch to primary blob store and others, and add support for v10
	// directory layout
	blobStores map[string]blob_stores.BlobStoreInitialized
}

func MakeBlobStoreEnv(
	envLocal env_local.Env,
) BlobStoreEnv {
	env := BlobStoreEnv{
		Env: envLocal,
	}

	{
		var err error

		if env.BlobStore, err = directory_layout.MakeBlobStore(
			store_version.VCurrent,
			envLocal.GetXDGForBlobStores(),
		); err != nil {
			envLocal.Cancel(err)
			return env
		}
	}

	env.setupStores()

	return env
}

func (env *BlobStoreEnv) setupStores() {
	env.blobStores = blob_stores.MakeBlobStores(
		env,
		env,
		env.BlobStore,
	)

	keys := slices.Collect(maps.Keys(env.blobStores))

	if len(keys) == 0 {
		return
	}

	sort.Strings(keys)
	env.defaultBlobStoreIdString = keys[0]
}

func (env BlobStoreEnv) GetDefaultBlobStore() blob_stores.BlobStoreInitialized {
	if len(env.blobStores) == 0 {
		panic(
			errors.Errorf(
				"calling GetDefaultBlobStore without any initialized blob stores: %#v",
				env.BlobStore,
			),
		)
	}

	return env.blobStores[env.defaultBlobStoreIdString]
}

func (env BlobStoreEnv) GetBlobStores() blob_stores.BlobStoreMap {
	blobStores := maps.Clone(env.blobStores)
	return blobStores
}

func (env BlobStoreEnv) GetBlobStoresSorted() []blob_stores.BlobStoreInitialized {
	blobStores := slices.Collect(maps.Values(env.blobStores))
	sort.Slice(blobStores, func(i, j int) bool {
		return blobStores[i].Path.GetId().Less(blobStores[j].Path.GetId())
	})
	return blobStores
}

func (env BlobStoreEnv) GetBlobStore(
	blobStoreId blob_store_id.Id,
) blob_stores.BlobStoreInitialized {
	key := blobStoreId.String()

	if blobStore, ok := env.blobStores[key]; ok {
		return blobStore
	}

	available := slices.Collect(maps.Keys(env.blobStores))
	sort.Strings(available)

	errors.ContextCancelWithBadRequestf(
		env,
		"blob store not found: %q (available: %v)",
		key,
		available,
	)

	return blob_stores.BlobStoreInitialized{}
}

func (env BlobStoreEnv) GetDefaultBlobStoreAndRemaining() (blob_stores.BlobStoreInitialized, blob_stores.BlobStoreMap) {
	defaultBlobStore := env.GetDefaultBlobStore()
	remaining := env.GetBlobStores()

	maps.DeleteFunc(
		remaining,
		func(idString string, _ blob_stores.BlobStoreInitialized) bool {
			return idString == env.defaultBlobStoreIdString
		},
	)

	return defaultBlobStore, remaining
}

func (env *BlobStoreEnv) writeBlobStoreConfigIfNecessary(
	bigBang BigBang,
	directoryLayout directory_layout.BlobStore,
) {
	if !bigBang.BlobStoreId.IsEmpty() {
		return
	}

	blobStoreConfigPath := directory_layout.GetDefaultBlobStore(
		directoryLayout,
	).GetConfig()

	blobStoreConfigDir := filepath.Dir(blobStoreConfigPath)

	if err := env.MakeDirs(blobStoreConfigDir); err != nil {
		env.Cancel(err)
		return
	}

	blobStoreConfig := bigBang.TypedBlobStoreConfig

	if err := triple_hyphen_io.EncodeToFile(
		blob_store_configs.Coder,
		&blob_store_configs.TypedConfig{
			Type: blobStoreConfig.Type,
			Blob: blobStoreConfig.Blob,
		},
		blobStoreConfigPath,
	); err != nil {
		env.Cancel(err)
		return
	}
}
