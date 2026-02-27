package store

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/thyme"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/india/zettel_id_index"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/stream_index"
	"code.linenisgreat.com/dodder/go/internal/mike/inventory_list_store"
	"code.linenisgreat.com/dodder/go/internal/mike/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/romeo/store_config"
)

func (store *Store) GetTypedBlobStore() typed_blob_store.Stores {
	return store.typedBlobStore
}

func (store *Store) GetProtoZettel() sku.Proto {
	return store.protoZettel
}

func (store *Store) GetTime() thyme.Time {
	return thyme.Now()
}

func (store *Store) GetTai() ids.Tai {
	return ids.NowTai()
}

func (store *Store) GetInventoryListStore() *inventory_list_store.Store {
	return &store.inventoryListStore
}

func (store *Store) GetAbbrStore() sku.IdIndex {
	return store.Abbr
}

func (store *Store) GetZettelIdIndex() zettel_id_index.Index {
	return store.zettelIdIndex
}

func (store *Store) GetEnvRepo() env_repo.Env {
	return store.envRepo
}

func (store *Store) GetConfigStore() store_config.Store {
	return store.storeConfig
}

func (store *Store) GetConfigStoreMutable() store_config.StoreMutable {
	return store.storeConfig
}

func (store *Store) GetStreamIndex() *stream_index.Index {
	return store.streamIndex
}

func (store *Store) GetConfigBlobCoder() interfaces.CoderReadWriter[*repo_configs.TypedBlob] {
	return store.configBlobCoder
}
