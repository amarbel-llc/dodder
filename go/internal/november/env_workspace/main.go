package env_workspace

import (
	"fmt"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/fd"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/mike/store_fs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

type Env interface {
	env_dir.Env
	GetWorkspaceDir() string
	AssertNotTemporary(errors.Context)
	AssertNotTemporaryOrOfferToCreate(errors.Context)
	IsTemporary() bool
	GetWorkspaceConfigTyped() workspace_config_blobs.TypedConfig
	GetWorkspaceConfig() workspace_config_blobs.Config
	GetWorkspaceConfigFilePath() string
	GetDefaults() repo_configs.Defaults
	CreateWorkspace(workspace_config_blobs.Config) (err error)
	GetStore() *Store

	// TODO identify users of this and reduce / isolate them
	GetStoreFS() *store_fs.Store

	SetWorkspaceTypes(map[string]*Store) (err error)
	SetSupplies(store_workspace.Supplies) (err error)

	Flush() (err error)
}

type Config interface {
	repo_configs.DefaultsGetter
	sku.Config
	file_extensions.ConfigGetter
}

func Make(
	envLocal env_local.Env,
	config Config,
	deletedPrinter interfaces.FuncIter[*fd.FD],
	envRepo env_repo.Env,
) (outputEnv *env, err error) {
	outputEnv = &env{
		envRepo:       envRepo,
		Env:           envLocal,
		configMutable: config,
	}

	object := workspace_config_blobs.TypedConfig{
		Type: ids.TypeStruct{},
	}

	dir := outputEnv.GetCwd()

	workspaceFile := outputEnv.findWorkspaceFile(dir, env_repo.FileWorkspace)

	if workspaceFile == "" {
		workspaceFile = outputEnv.findWorkspaceFile(
			dir,
			fmt.Sprintf(env_repo.FileWorkspaceTemplate, "zit"),
		)
	}

	if workspaceFile == "" {
		outputEnv.isTemporary = true
		outputEnv.blob = workspace_config_blobs.Temporary{}
	} else {
		if err = triple_hyphen_io.DecodeFromFileInto(
			&object,
			workspace_config_blobs.Coder,
			workspaceFile,
		); err != nil {
			err = errors.BadRequestf("failed to decode `%s`: %w", workspaceFile, err)
			return outputEnv, err
		}

		outputEnv.blob = object.Blob
	}

	defaults := outputEnv.configMutable.GetDefaults()

	outputEnv.defaults = repo_configs.DefaultsV1{
		Type: defaults.GetDefaultType(),
		Tags: defaults.GetDefaultTags(),
	}

	if outputEnv.blob != nil {
		defaults = outputEnv.blob.GetDefaults()

		if newType := defaults.GetDefaultType(); !newType.IsEmpty() {
			outputEnv.defaults.Type = newType
		}

		if newTags := defaults.GetDefaultTags(); newTags.Len() > 0 {
			outputEnv.defaults.Tags = append(
				outputEnv.defaults.Tags,
				newTags...,
			)
		}
	}

	if outputEnv.isTemporary {
		if outputEnv.dir, err = outputEnv.GetTempLocal().DirTempWithTemplate(
			"workspace-*",
		); err != nil {
			err = errors.Wrap(err)
			return outputEnv, err
		}
	} else {
		// TODO determine this based on the blob
		outputEnv.dir = outputEnv.GetCwd()
	}

	if outputEnv.storeFS, err = store_fs.Make(
		config,
		deletedPrinter,
		config.GetFileExtensions(),
		envRepo,
	); err != nil {
		err = errors.Wrap(err)
		return outputEnv, err
	}

	outputEnv.store.StoreLike = outputEnv.storeFS

	return outputEnv, err
}

type env struct {
	envRepo env_repo.Env
	env_local.Env

	isTemporary bool

	// dir is populated on init to either the cwd, or a temporary directory,
	// depending on whether $PWD/.dodder-workspace exists.
	//
	// Later, dir may be set to $PWD/.dodder-workspace by CreateWorkspace
	dir string

	configMutable repo_configs.DefaultsGetter
	blob          workspace_config_blobs.Config
	defaults      repo_configs.DefaultsV1

	storeFS *store_fs.Store
	store   Store
}

func (env *env) findWorkspaceFile(
	dir string,
	name string,
) (found string) {
	// TODO add workspace parent tree height limit?
	for {
		expectedWorkspaceConfigFilePath := filepath.Join(
			dir,
			name,
		)

		if files.Exists(expectedWorkspaceConfigFilePath) {
			found = expectedWorkspaceConfigFilePath
			return found
		}

		// if we hit the root, reset to empty so that we trigger the isTemporary
		// path
		if dir == string(filepath.Separator) {
			dir = ""
		}

		dir = filepath.Dir(dir)

		if dir != "." {
			continue
		}

		return found
	}
}

func (env *env) GetWorkspaceDir() string {
	return env.dir
}

func (env *env) GetWorkspaceConfigFilePath() string {
	return filepath.Join(env.GetWorkspaceDir(), env_repo.FileWorkspace)
}

func (env *env) AssertNotTemporary(context errors.Context) {
	if env.IsTemporary() {
		context.Cancel(ErrNotInWorkspace{env: env})
	}
}

func (env *env) AssertNotTemporaryOrOfferToCreate(context errors.Context) {
	if env.IsTemporary() {
		context.Cancel(
			ErrNotInWorkspace{
				env:           env,
				offerToCreate: true,
			})
	}
}

func (env *env) IsTemporary() bool {
	return env.isTemporary
}

func (env *env) GetWorkspaceConfigTyped() workspace_config_blobs.TypedConfig {
	typeWorkspaceConfig := ids.GetOrPanic(ids.TypeTomlWorkspaceConfigV0).TypeStruct

	return workspace_config_blobs.TypedConfig{
		Type: typeWorkspaceConfig,
		Blob: env.blob,
	}
}

func (env *env) GetWorkspaceConfig() workspace_config_blobs.Config {
	return env.blob
}

func (env *env) GetDefaults() repo_configs.Defaults {
	return env.defaults
}

func (env *env) GetStore() *Store {
	return &env.store
}

func (env *env) GetStoreFS() *store_fs.Store {
	return env.storeFS
}

func (env *env) CreateWorkspace(
	blob workspace_config_blobs.Config,
) (err error) {
	env.blob = blob

	typeWorkspaceConfig := ids.GetOrPanic(ids.TypeTomlWorkspaceConfigV0).TypeStruct

	object := workspace_config_blobs.TypedConfig{
		Type: typeWorkspaceConfig,
		Blob: env.blob,
	}

	env.dir = env.GetCwd()

	if err = triple_hyphen_io.EncodeToFile(
		workspace_config_blobs.Coder,
		&object,
		env.GetWorkspaceConfigFilePath(),
	); errors.IsExist(err) {
		err = errors.BadRequestf("workspace already exists")
		return err
	} else if err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (env *env) SetSupplies(supplies store_workspace.Supplies) (err error) {
	env.store.Supplies = supplies

	if err = env.store.Initialize(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// TODO persist store types and bootstrap based on workspace config
func (env *env) SetWorkspaceTypes(
	stores map[string]*Store,
) (err error) {
	return err
}

func (env *env) Flush() (err error) {
	waitGroup := errors.MakeWaitGroupParallel()

	waitGroup.Do(env.store.Flush)

	if err = waitGroup.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
