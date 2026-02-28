package local_working_copy

import (
	"time"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/echo/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/dormant_index"
	"code.linenisgreat.com/dodder/go/internal/hotel/object_finalizer"
	"code.linenisgreat.com/dodder/go/internal/india/env_lua"
	"code.linenisgreat.com/dodder/go/internal/india/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/juliett/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/november/env_workspace"
	"code.linenisgreat.com/dodder/go/internal/oscar/store_config"
	"code.linenisgreat.com/dodder/go/internal/papa/store"
)

func (local *Repo) GetEnv() env_ui.Env {
	return local
}

func (local *Repo) GetImmutableConfigPublic() genesis_configs.ConfigPublic {
	return local.GetEnvRepo().GetConfigPublic().Blob
}

func (local *Repo) GetImmutableConfigPublicType() ids.TypeStruct {
	return local.GetEnvRepo().GetConfigPublic().Type
}

func (local *Repo) GetImmutableConfigPrivate() genesis_configs.TypedConfigPrivate {
	return local.GetEnvRepo().GetConfigPrivate()
}

func (local *Repo) GetEnvLocal() env_local.Env {
	return local
}

func (local *Repo) GetEnvWorkspace() env_workspace.Env {
	return local.envWorkspace
}

func (local *Repo) GetWorkspaceConfig() workspace_config_blobs.Config {
	return local.GetEnvWorkspace().GetWorkspaceConfig()
}

func (local *Repo) GetEnvLua() env_lua.Env {
	return local.envLua
}

func (local *Repo) GetTime() time.Time {
	return time.Now()
}

func (local *Repo) GetConfigStore() store_config.Store {
	return local.config
}

func (local *Repo) GetConfigStoreMutable() store_config.StoreMutable {
	return local.config
}

func (local *Repo) GetConfig() store_config.Config {
	return local.config.GetConfig()
}

func (local *Repo) GetConfigPtr() *store_config.Config {
	return local.config.GetConfigPtr()
}

func (local *Repo) GetDormantIndex() *dormant_index.Index {
	return &local.dormantIndex
}

func (local *Repo) GetEnvRepo() env_repo.Env {
	return local.envRepo
}

func (local *Repo) GetTypedBlobStore() typed_blob_store.Stores {
	return local.typedBlobStore
}

func (local *Repo) GetInventoryListCoderCloset() inventory_list_coders.Closet {
	return local.typedBlobStore.InventoryList
}

func (local *Repo) GetBlobStore() blob_stores.BlobStoreInitialized {
	return local.GetEnvRepo().GetDefaultBlobStore()
}

func (local *Repo) GetObjectStore() sku.RepoStore {
	return &local.store
}

func (local *Repo) GetInventoryListStore() sku.InventoryListStore {
	return local.GetStore().GetInventoryListStore()
}

func (local *Repo) GetStore() *store.Store {
	return &local.store
}

func (local *Repo) GetAbbr() sku.IdIndex {
	return local.indexIds
}

func (local *Repo) GetObjectFinalizer() object_finalizer.Finalizer {
	return object_finalizer.Make()
}
