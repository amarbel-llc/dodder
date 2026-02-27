package repo

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/india/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/juliett/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/mike/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/romeo/env_workspace"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

// TODO explore permissions for who can read / write from the inventory list
// store
type Repo interface {
	GetEnv() env_ui.Env
	GetImmutableConfigPublic() genesis_configs.ConfigPublic
	GetImmutableConfigPublicType() ids.TypeStruct
	GetBlobStore() blob_stores.BlobStoreInitialized
	GetObjectStore() sku.RepoStore
	GetInventoryListCoderCloset() inventory_list_coders.Closet
	GetInventoryListStore() sku.InventoryListStore

	MakeImporter(
		options ImporterOptions,
		storeOptions sku.StoreOptions,
	) Importer

	ImportSeq(
		interfaces.SeqError[*sku.Transacted],
		Importer,
	) error

	MakeExternalQueryGroup(
		builderOptions queries.BuilderOption,
		externalQueryOptions sku.ExternalQueryOptions,
		args ...string,
	) (qg *queries.Query, err error)

	MakeInventoryList(
		qg *queries.Query,
	) (list *sku.HeapTransacted, err error)

	// TODO replace with WorkingCopy
	PullQueryGroupFromRemote(
		remote Repo,
		qg *queries.Query,
		options ImporterOptions,
	) (err error)

	ReadObjectHistory(
		oid *ids.ObjectId,
	) (skus []*sku.Transacted, err error)
}

type LocalRepo interface {
	Repo

	GetEnvRepo() env_repo.Env
	GetImmutableConfigPrivate() genesis_configs.TypedConfigPrivate

	Lock() error
	Unlock() error

	GetEnvWorkspace() env_workspace.Env
}
