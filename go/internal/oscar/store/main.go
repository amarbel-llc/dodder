package store

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/zettel_id_index"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/internal/golf/dormant_index"
	"code.linenisgreat.com/dodder/go/internal/golf/object_finalizer"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/stream_index"
	"code.linenisgreat.com/dodder/go/internal/hotel/stream_index_fixed"
	"code.linenisgreat.com/dodder/go/internal/india/inventory_list_store"
	"code.linenisgreat.com/dodder/go/internal/india/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/mike/env_workspace"
	"code.linenisgreat.com/dodder/go/internal/november/store_config"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type Store struct {
	sunrise      ids.Tai
	storeConfig  store_config.StoreMutable
	envRepo      env_repo.Env
	envWorkspace env_workspace.Env

	typedBlobStore     typed_blob_store.Stores
	inventoryListStore inventory_list_store.Store
	Abbr               sku.IdIndex

	workingList *sku.WorkingList
	envLua      env_lua.Env

	streamIndex   sku.StreamIndex
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

	if store.storeConfig.GetConfig().GetStreamIndexFixed() {
		if store.streamIndex, err = stream_index_fixed.MakeIndex(
			store.GetEnvRepo(),
			store.applyDormantAndRealizeTags,
			store.GetEnvRepo().DirIndexObjects(),
			store.sunrise,
			0,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if store.streamIndex, err = stream_index.MakeIndex(
			store.GetEnvRepo(),
			store.applyDormantAndRealizeTags,
			store.GetEnvRepo().DirIndexObjects(),
			store.sunrise,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	store.finalizer = object_finalizer.Make()

	store.protoZettel = sku.MakeProto(
		store.envWorkspace.GetDefaults(),
	)

	return err
}

func (store *Store) MakeSupplies(
	repoId ids.RepoId,
) (supplies store_workspace.Supplies) {
	supplies.WorkspaceDir = store.envWorkspace.GetWorkspaceDir()
	supplies.StoreCommitter = store
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
	blobId mad_domain_interfaces.MarklId,
) (kt *sku.Transacted, err error) {
	return store.createOrUpdateBlobDigest(
		ids.Config,
		blobId,
	)
}
