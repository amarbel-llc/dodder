package store

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/zettel_id_index"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/box_format"
	"code.linenisgreat.com/dodder/go/internal/hotel/dormant_index"
	"code.linenisgreat.com/dodder/go/internal/hotel/object_finalizer"
	"code.linenisgreat.com/dodder/go/internal/india/env_lua"
	"code.linenisgreat.com/dodder/go/internal/india/stream_index"
	"code.linenisgreat.com/dodder/go/internal/juliett/inventory_list_store"
	"code.linenisgreat.com/dodder/go/internal/juliett/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/november/env_workspace"
	"code.linenisgreat.com/dodder/go/internal/oscar/store_config"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Store struct {
	sunrise      ids.Tai
	storeConfig  store_config.StoreMutable
	envRepo      env_repo.Env
	envWorkspace env_workspace.Env

	typedBlobStore     typed_blob_store.Stores
	inventoryListStore inventory_list_store.Store
	Abbr               sku.IdIndex

	workingList     *sku.WorkingList
	configBlobCoder interfaces.CoderReadWriter[*repo_configs.TypedBlob]
	envLua          env_lua.Env

	streamIndex   *stream_index.Index
	finalizer     object_finalizer.Finalizer
	zettelIdIndex zettel_id_index.Index
	dormantIndex  *dormant_index.Index

	protoZettel  sku.Proto
	queryBuilder *queries.Builder

	ui sku.UIStorePrinters
}

var _ sku.RepoStore = &Store{}

func (store *Store) Initialize(
	config store_config.StoreMutable,
	envRepo env_repo.Env,
	envWorkspace env_workspace.Env,
	sunrise ids.Tai,
	envLua env_lua.Env,
	queryBuilder *queries.Builder,
	box *box_format.BoxTransacted,
	typedBlobStore typed_blob_store.Stores,
	dormantIndex *dormant_index.Index,
	abbrStore sku.IdIndex,
) (err error) {
	store.storeConfig = config
	store.envRepo = envRepo
	store.envWorkspace = envWorkspace
	store.typedBlobStore = typedBlobStore
	store.sunrise = sunrise
	store.envLua = envLua
	store.queryBuilder = queryBuilder
	store.dormantIndex = dormantIndex

	store.Abbr = abbrStore

	if err = store.inventoryListStore.Initialize(
		store.GetEnvRepo(),
		store,
		typedBlobStore.InventoryList,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if store.workingList, err = store.inventoryListStore.MakeWorkingList(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if store.zettelIdIndex, err = zettel_id_index.MakeIndex(
		store.GetConfigStore().GetConfig().GetGenesisConfigPublic(),
		store.storeConfig.GetConfig().CLI,
		store.GetEnvRepo(),
		store.GetEnvRepo(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if store.streamIndex, err = stream_index.MakeIndex(
		store.GetEnvRepo(),
		store.applyDormantAndRealizeTags,
		store.GetEnvRepo().DirIndexObjects(),
		store.sunrise,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	store.finalizer = object_finalizer.Make()

	store.protoZettel = sku.MakeProto(
		store.envWorkspace.GetDefaults(),
	)

	store.configBlobCoder = repo_configs.Coder

	return err
}

func (store *Store) MakeSupplies(
	repoId ids.RepoId,
) (supplies store_workspace.Supplies) {
	supplies.WorkspaceDir = store.envWorkspace.GetWorkspaceDir()
	supplies.RepoStore = store

	supplies.Env = store.GetEnvRepo()
	supplies.Clock = store.sunrise
	supplies.BlobStore = store.typedBlobStore
	supplies.RepoId = repoId
	supplies.DirCache = store.GetEnvRepo().DirCacheRepo(
		repoId.GetRepoIdString(),
	)

	return supplies
}

func (store *Store) ResetIndexes() (err error) {
	if err = store.zettelIdIndex.Reset(); err != nil {
		err = errors.Wrapf(err, "failed to reset index object id index")
		return err
	}

	if err = store.streamIndex.Reset(); err != nil {
		err = errors.Wrapf(err, "failed to reset stream index")
		return err
	}

	return err
}

func (store *Store) SetUIDelegate(ud sku.UIStorePrinters) {
	store.ui = ud
	store.inventoryListStore.SetUIDelegate(ud)
}

func (store *Store) UpdateKonfig(
	blobId domain_interfaces.MarklId,
) (kt *sku.Transacted, err error) {
	return store.CreateOrUpdateBlobDigest(
		ids.Config,
		blobId,
	)
}
