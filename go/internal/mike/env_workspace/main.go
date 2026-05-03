package env_workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/caldav"
	"code.linenisgreat.com/dodder/go/internal/0/filesystem_ops"
	"code.linenisgreat.com/dodder/go/internal/0/webdav"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/env_local"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/lima/haustoria_caldav"
	"code.linenisgreat.com/dodder/go/internal/lima/haustoria_orgmode"
	"code.linenisgreat.com/dodder/go/internal/lima/store_fs"
	"github.com/amarbel-llc/madder/go/pkgs/fd"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	madder_ids "github.com/amarbel-llc/madder/go/pkgs/ids"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/echo/xdg"
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
	GetParentPath() string
	GetSyncBaseline() (tai string, digest string)
	UpdateSyncBaseline(inventoryListStore sku.InventoryListStore) error
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
		Type: madder_ids.TypeStruct{},
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
		if err = hyphence.DecodeFromFileInto(
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

	fsOps := filesystem_ops.MakeOsFilesystemOps(envRepo.GetCwd())

	if outputEnv.storeFS, err = store_fs.Make(
		config,
		deletedPrinter,
		config.GetFileExtensions(),
		envRepo,
		fsOps,
	); err != nil {
		err = errors.Wrap(err)
		return outputEnv, err
	}

	if cfg, ok := outputEnv.blob.(workspace_config_blobs.ConfigWithHaustoria); ok {
		hCfg := cfg.GetHaustoriaConfig()
		if hCfg.Type == "caldav" && hCfg.CalDAV != nil {
			resolved, resolveErr := hCfg.CalDAV.Resolve()
			if resolveErr == nil {
				caldavCfg := &caldav.Config{
					URL:      resolved.URL,
					Username: resolved.Username,
					Password: resolved.Password,
				}

				var calendars []haustoria_caldav.CalendarMapping

				if len(hCfg.Calendars) > 0 {
					for _, cal := range hCfg.Calendars {
						if cal.URL == "" {
							continue
						}

						calendars = append(calendars, haustoria_caldav.CalendarMapping{
							URL:    cal.URL,
							TypeId: cal.Type,
							Tags:   cal.Tags,
						})
					}
				}

				if len(calendars) == 0 {
					// Backwards compat: single URL from caldav config
					calendars = []haustoria_caldav.CalendarMapping{{
						URL:    resolved.URL,
						TypeId: "!task",
					}}
				}

				outputEnv.store.StoreLike = haustoria_caldav.MakeStore(
					caldavCfg,
					calendars,
				)
			}

		} else if hCfg.Type == "orgmode" && hCfg.Orgmode != nil {
			resolved, resolveErr := hCfg.Orgmode.ResolveOrgmode()
			if resolveErr == nil {
				var transport haustoria_orgmode.Transport

				switch resolved.Transport {
				case "webdav":
					transport = haustoria_orgmode.MakeWebDAVTransport(
						&webdav.Config{
							URL:      resolved.WebDAVURL,
							Username: resolved.WebDAVUsername,
							Password: resolved.WebDAVPassword,
						},
					)

				case "sftp":
					var sftpErr error
					transport, sftpErr = haustoria_orgmode.MakeSFTPTransport(
						haustoria_orgmode.SFTPConfig{
							Host:           resolved.SFTPHost,
							Port:           resolved.SFTPPort,
							User:           resolved.SFTPUser,
							Password:       resolved.SFTPPassword,
							PrivateKeyPath: resolved.SFTPPrivateKeyPath,
							KnownHostsFile: resolved.SFTPKnownHostsFile,
						},
					)
					if sftpErr != nil {
						err = errors.Wrapf(sftpErr, "orgmode sftp transport")
						return
					}
				}

				if transport != nil {
					var folders []haustoria_orgmode.FolderMapping

					for _, folder := range hCfg.Folders {
						if folder.Path == "" {
							continue
						}

						folders = append(folders, haustoria_orgmode.FolderMapping{
							Path:   folder.Path,
							TypeId: folder.Type,
							Tags:   folder.Tags,
						})
					}

					if len(folders) == 0 {
						// Default: use the WebDAV URL or SFTP path as a single folder.
						defaultPath := resolved.WebDAVURL
						if resolved.Transport == "sftp" {
							defaultPath = "/org"
						}

						folders = []haustoria_orgmode.FolderMapping{{
							Path:   defaultPath,
							TypeId: "!md",
						}}
					}

					outputEnv.store.StoreLike = haustoria_orgmode.MakeStore(
						transport,
						folders,
					)
				}
			}
		}
	}

	if outputEnv.store.StoreLike == nil {
		outputEnv.store.StoreLike = outputEnv.storeFS
	}

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
	ceilings := xdg.ParseCeilingDirectories(
		os.Getenv(xdg.CeilingEnvVarName(env.GetXDG().UtilityName)),
	)

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

		parent := filepath.Dir(dir)

		if xdg.IsAtOrAboveCeiling(parent, ceilings) {
			return found
		}

		dir = parent

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
	typeString := ids.TypeTomlWorkspaceConfigV0
	if _, ok := env.blob.(workspace_config_blobs.ConfigWithHaustoria); ok {
		typeString = ids.TypeTomlWorkspaceConfigV2
	} else if _, ok := env.blob.(workspace_config_blobs.ConfigWithParentPath); ok {
		typeString = ids.TypeTomlWorkspaceConfigV1
	}

	typeWorkspaceConfig := ids.GetOrPanic(typeString).TypeStruct

	return workspace_config_blobs.TypedConfig{
		Type: typeWorkspaceConfig.ToMadder(),
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

func (env *env) GetParentPath() string {
	if cp, ok := env.blob.(workspace_config_blobs.ConfigWithParentPath); ok {
		return cp.GetParentPath()
	}

	return ""
}

func (env *env) GetSyncBaseline() (tai string, digest string) {
	if sb, ok := env.blob.(workspace_config_blobs.ConfigWithSyncBaseline); ok {
		return sb.GetSyncTai(), sb.GetSyncDigest()
	}

	return "", ""
}

func (env *env) UpdateSyncBaseline(
	inventoryListStore sku.InventoryListStore,
) (err error) {
	v1, ok := env.blob.(*workspace_config_blobs.V1)
	if !ok {
		return nil // V0 workspace, no-op
	}

	last, err := inventoryListStore.ReadLast()
	if err != nil {
		return errors.Wrap(err)
	}

	v1.SyncTai = last.GetTai().String()
	v1.SyncDigest = last.GetMetadata().GetObjectDigest().String()

	return env.rewriteConfig()
}

func (env *env) rewriteConfig() (err error) {
	object := env.GetWorkspaceConfigTyped()

	file, err := os.Create(env.GetWorkspaceConfigFilePath())
	if err != nil {
		return errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, file)

	if _, err = workspace_config_blobs.Coder.EncodeTo(&object, file); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (env *env) CreateWorkspace(
	blob workspace_config_blobs.Config,
) (err error) {
	env.blob = blob

	typeString := ids.TypeTomlWorkspaceConfigV0
	if _, ok := blob.(workspace_config_blobs.ConfigWithHaustoria); ok {
		typeString = ids.TypeTomlWorkspaceConfigV2
	} else if _, ok := blob.(workspace_config_blobs.ConfigWithParentPath); ok {
		typeString = ids.TypeTomlWorkspaceConfigV1
	}

	typeWorkspaceConfig := ids.GetOrPanic(typeString).TypeStruct

	object := workspace_config_blobs.TypedConfig{
		Type: typeWorkspaceConfig.ToMadder(),
		Blob: env.blob,
	}

	env.dir = env.GetCwd()

	if err = hyphence.EncodeToFile(
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
