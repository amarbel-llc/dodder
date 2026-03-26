package env_repo

import (
	"maps"
	"path/filepath"
	"slices"
	"sort"

	"code.linenisgreat.com/dodder/go/internal/alfa/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type BlobStoreEnv struct {
	directory_layout.BlobStore
	env_local.Env

	defaultBlobStoreIdString string
	orderedBlobStoreIds      []string // nil = alphabetical (discovery mode)

	// TODO switch to implementing LocalBlobStore directly and writing to all of
	// the defined blob stores instead of having a default
	// TODO switch to primary blob store and others, and add support for v10
	// directory layout
	blobStores map[string]blob_stores.BlobStoreInitialized
}

func makeBlobStoreEnvBase(
	envLocal env_local.Env,
) (BlobStoreEnv, bool) {
	env := BlobStoreEnv{
		Env: envLocal,
	}

	var err error

	if env.BlobStore, err = directory_layout.MakeBlobStore(
		store_version.VCurrent,
		envLocal.GetXDGForBlobStores(),
	); err != nil {
		envLocal.Cancel(err)
		return env, false
	}

	return env, true
}

func MakeBlobStoreEnv(
	envLocal env_local.Env,
) BlobStoreEnv {
	env, ok := makeBlobStoreEnvBase(envLocal)
	if !ok {
		return env
	}

	env.setupStores()

	return env
}

func MakeBlobStoreEnvWithOrder(
	envLocal env_local.Env,
	blobStoreIds []blob_store_id.Id,
) BlobStoreEnv {
	env, ok := makeBlobStoreEnvBase(envLocal)
	if !ok {
		return env
	}

	env.setupStores()
	env.SetBlobStoreOrder(blobStoreIds)

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

func (env *BlobStoreEnv) SetBlobStoreOrder(blobStoreIds []blob_store_id.Id) {
	if len(blobStoreIds) == 0 {
		return
	}

	ids := make([]string, len(blobStoreIds))

	for i, id := range blobStoreIds {
		ids[i] = id.String()
	}

	env.orderedBlobStoreIds = ids
	env.defaultBlobStoreIdString = ids[0]
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
	if env.orderedBlobStoreIds != nil {
		blobStores := make([]blob_stores.BlobStoreInitialized, 0, len(env.orderedBlobStoreIds))

		for _, id := range env.orderedBlobStoreIds {
			if bs, ok := env.blobStores[id]; ok {
				blobStores = append(blobStores, bs)
			}
		}

		return blobStores
	}

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

	if env.orderedBlobStoreIds != nil {
		remaining := make(blob_stores.BlobStoreMap, len(env.orderedBlobStoreIds)-1)

		for _, id := range env.orderedBlobStoreIds[1:] {
			if bs, ok := env.blobStores[id]; ok {
				remaining[id] = bs
			}
		}

		return defaultBlobStore, remaining
	}

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

	if err := hyphence.EncodeToFile(
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
