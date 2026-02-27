package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/box_format"
	"code.linenisgreat.com/dodder/go/internal/kilo/dormant_index"
	index_ids "code.linenisgreat.com/dodder/go/internal/kilo/store_abbr"
	"code.linenisgreat.com/dodder/go/internal/lima/env_lua"
	"code.linenisgreat.com/dodder/go/internal/mike/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/quebec/env_workspace"
	"code.linenisgreat.com/dodder/go/internal/romeo/store_config"
	"code.linenisgreat.com/dodder/go/internal/sierra/env_box"
	"code.linenisgreat.com/dodder/go/internal/tango/store"
)

type (
	envLocal     = env_local.Env
	envBox       = env_box.Env
	envWorkspace = env_workspace.Env
)

type Repo struct {
	envLocal
	envBox
	envWorkspace envWorkspace

	sunrise ids.Tai

	envRepo env_repo.Env
	config  store_config.StoreMutable

	indexIds     sku.IdIndex
	dormantIndex dormant_index.Index

	storesInitialized bool
	typedBlobStore    typed_blob_store.Stores
	store             store.Store

	// TODO switch key to be workspace type
	workspaceStores map[ids.RepoId]*env_workspace.Store

	DormantCounter queries.DormantCounter

	envLua env_lua.Env
}

func Make(
	env env_local.Env,
	options Options,
) *Repo {
	var basePath string
	if repoConfig, ok := env.GetCLIConfig().(domain_interfaces.RepoCLIConfigProvider); ok {
		basePath = repoConfig.GetBasePath()
	}

	layoutOptions := env_repo.Options{
		BasePath: basePath,
	}

	var envRepo env_repo.Env

	{
		var err error

		if envRepo, err = env_repo.Make(
			env,
			layoutOptions,
		); err != nil {
			env.Cancel(err)
		}
	}

	return MakeWithEnvRepo(options, envRepo)
}

func MakeWithEnvRepo(
	options Options,
	envRepo env_repo.Env,
) (repo *Repo) {
	repo = &Repo{
		config:         store_config.Make(),
		envLocal:       envRepo,
		envRepo:        envRepo,
		DormantCounter: queries.MakeDormantCounter(),
	}

	repo.config.Reset()

	if err := repo.initialize(options); err != nil {
		repo.Cancel(err)
	}

	repo.After(errors.MakeFuncContextFromFuncErr(repo.Flush))

	return repo
}

// TODO investigate removing unnecessary resets like from organize
func (local *Repo) Reset() (err error) {
	return local.initialize(OptionsEmpty)
}

func (local *Repo) initialize(
	options Options,
) (err error) {
	if err = local.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// ui.Debug().Print(repo.layout.GetConfig().GetBlobStoreImmutableConfig().GetCompressionType())
	local.sunrise = ids.NowTai()

	if err = local.dormantIndex.Load(
		local.envRepo,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	boxFormatArchive := box_format.MakeBoxTransactedArchive(
		local.GetEnv(),
		local.GetConfig().GetPrintOptions().WithPrintTai(true),
	)

	// Type assertion is safe here because local_working_copy is dodder-specific
	// and always receives repo_config_cli.Config
	cliConfig, _ := local.GetCLIConfig().(repo_config_cli.Config)
	if err = local.config.Initialize(
		local.envRepo,
		cliConfig,
	); err != nil {
		if options.GetAllowConfigReadError() {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	if local.envWorkspace, err = env_workspace.Make(
		local.envRepo,
		local.config.GetConfig(),
		local.PrinterFDDeleted(),
		local.GetEnvRepo(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if local.indexIds, err = index_ids.NewIndex(
		local.config.GetConfig().GetPrintOptions(),
		local.envRepo,
	); err != nil {
		err = errors.Wrapf(err, "failed to init abbr index")
		return err
	}

	local.envBox = env_box.Make(
		local.envRepo,
		local.config.GetConfig(),
		local.envWorkspace.GetStoreFS(),
		local.indexIds,
	)

	local.envLua = env_lua.Make(
		local.envRepo,
		local.GetStore(),
		local.SkuFormatBoxTransactedNoColor(),
	)

	// for _, rb := range u.GetConfig().Recipients {
	// 	if err = u.age.AddBech32PivYubikeyEC256(rb); err != nil {
	// 		errors.Wrap(err)
	// 		return
	// 	}
	// }

	local.typedBlobStore = typed_blob_store.MakeStores(
		local.envRepo,
		local.envLua,
		boxFormatArchive,
	)

	if err = local.store.Initialize(
		local.config,
		local.envRepo,
		local.envWorkspace,
		local.sunrise,
		local.envLua,
		local.makeQueryBuilder().WithOptions(
			queries.BuilderOptionDefaultGenres(genres.All()...),
		),
		boxFormatArchive,
		local.typedBlobStore,
		&local.dormantIndex,
		local.indexIds,
	); err != nil {
		err = errors.Wrapf(err, "failed to initialize store util")
		return err
	}

	ui.Log().Printf(
		"store version: %s",
		local.GetConfig().GetGenesisConfigPublic().GetStoreVersion(),
	)

	if err = local.envWorkspace.SetWorkspaceTypes(
		local.initializeBrowserWorkspace(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = local.envWorkspace.SetSupplies(
		local.store.MakeSupplies(ids.RepoId{}),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ui.Log().Print("done initing checkout store")

	local.store.SetUIDelegate(local.GetUIStorePrinters())

	local.storesInitialized = true

	return err
}

func (local *Repo) Flush() (err error) {
	waitGroup := errors.MakeWaitGroupParallel()

	if local.envWorkspace != nil {
		waitGroup.Do(local.envWorkspace.Flush)
	}

	if err = waitGroup.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (local *Repo) PrintMatchedDormantIfNecessary() {
	if !local.GetConfig().GetPrintOptions().PrintMatchedDormant {
		return
	}

	c := local.GetMatcherDormant().Count()
	ca := local.GetMatcherDormant().CountArchiviert()

	if c != 0 || ca == 0 {
		return
	}

	ui.Err().Printf("%d archived objects matched", c)
}

func (local *Repo) GetMatcherDormant() queries.DormantCounter {
	return local.DormantCounter
}

func (local *Repo) GetWorkspaceStoreForQuery(
	repoId ids.RepoId,
) (store_workspace.Store, bool) {
	return local.envWorkspace.GetStore(), true
}
