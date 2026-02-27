package store

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/hotel/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/juliett/zettel_id_index"
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/mike/stream_index"
	"code.linenisgreat.com/dodder/go/internal/november/inventory_list_store"
	"code.linenisgreat.com/dodder/go/internal/november/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/sierra/store_config"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/echo/thyme"
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
