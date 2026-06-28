package env_repo

import (
	"os"
	"strings"

	mad_blob_store_env "github.com/amarbel-llc/madder/go/pkgs/blob_store_env"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"github.com/amarbel-llc/madder/go/pkgs/blob_stores"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/file_lock"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"github.com/amarbel-llc/hyphence/go/hyphence"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/env_vars"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

const (
	// TODO move to mutable config
	FileWorkspaceTemplate = ".%s-workspace"
	FileWorkspace         = ".dodder-workspace"
)

// BlobStoreEnv is dodder's stable name for madder's
// blob_store_env.BlobStoreEnv. Aliased so existing call sites in
// golf/blob_transfers and tango/command_components_dodder keep
// compiling while #151 bucket B's mechanical sweep is staged
// separately.
type BlobStoreEnv = mad_blob_store_env.BlobStoreEnv

// MakeBlobStoreEnv forwards to madder's blob_store_env.MakeBlobStoreEnv.
// Same aliasing rationale as BlobStoreEnv.
var MakeBlobStoreEnv = mad_blob_store_env.MakeBlobStoreEnv

type Env struct {
	config genesis_configs.TypedConfigPrivate

	lockSmith interfaces.LockSmith

	directory_layout.Repo

	// Env is the own-scope env_local (utilityName="dodder"). Its XDG
	// methods address dodder's config / cache / log / state directories.
	// Blob-store XDG goes through blobStoreEnv, not this embed.
	env_local.Env

	blobStoreEnv mad_blob_store_env.BlobStoreEnv
}

// TODO https://github.com/amarbel-llc/dodder/issues/27
// Stop returning error and cancel context instead
//
// Make takes two env_locals. ownEnvLocal addresses dodder's own
// state — its XDG namespace is "dodder" (config, cache, log, state).
// madderEnvLocal addresses madder's blob-store namespace — its XDG
// namespace is "madder". The two-env composition replaces the
// previous bridge in dodder's env_dir.GetXDGForBlobStores (which
// hardcoded utility name "madder") so the dodder env_dir / env_local
// forks can be dropped (#151 bucket B Stage B).
func Make(
	ownEnvLocal env_local.Env,
	madderEnvLocal env_local.Env,
	options Options,
) (env Env, err error) {
	env.Env = ownEnvLocal

	if options.BasePath == "" {
		options.BasePath = os.Getenv(dodder_env.EnvDir)
	}

	if options.BasePath == "" {
		if options.BasePath, err = os.Getwd(); err != nil {
			err = errors.Wrap(err)
			return env, err
		}
	}

	xdg := env.GetXDG()

	if env.GetXDG().Data.ActualValue == "" {
		err = errors.Errorf("empty data dir: %#v", env.GetXDG().Data)
		return env, err
	}

	fileConfigPermanent := env.GetPathConfigSeed().String()

	var configLoaded bool

	if options.PermitNoDodderDirectory {
		if env.config, err = hyphence.DecodeFromFile(
			genesis_configs.CoderPrivate,
			fileConfigPermanent,
		); err != nil {
			if errors.IsNotExist(err) {
				err = nil
			} else {
				err = wrapConfigSeedDecodeError(err, ownEnvLocal, fileConfigPermanent)
				return env, err
			}
		} else {
			configLoaded = true
		}
	} else {
		if env.config, err = hyphence.DecodeFromFile(
			genesis_configs.CoderPrivate,
			fileConfigPermanent,
		); err != nil {
			if errors.IsNotExist(err) {
				err = errors.Wrap(ErrNotInDodderDir{Expected: fileConfigPermanent})
			} else {
				err = wrapConfigSeedDecodeError(err, ownEnvLocal, fileConfigPermanent)
			}
			return env, err
		} else {
			configLoaded = true
		}
	}

	if env.Repo, err = directory_layout.MakeRepo(
		env.GetStoreVersion(),
		xdg,
	); err != nil {
		err = errors.Wrap(err)
		return env, err
	}

	// TODO fail on pre-existing temp local
	// if files.Exists(s.TempLocal.basePath) {
	// 	err = MakeErrTempAlreadyExists(s.TempLocal.basePath)
	// 	return
	// }

	if err = env.MakeDirsPerms(0o700, env.GetXDG().GetXDGPaths()...); err != nil {
		err = errors.Wrap(err)
		return env, err
	}

	env.lockSmith = file_lock.New(ownEnvLocal, env.FileLock(), "repo")

	envVars := env_vars.Make(env)

	env.Must(errors.MakeFuncContextFromFuncErr(envVars.Set))
	env.After(errors.MakeFuncContextFromFuncErr(envVars.Unset))

	if configLoaded {
		env.blobStoreEnv = mad_blob_store_env.MakeBlobStoreEnv(madderEnvLocal)
	} else {
		env.blobStoreEnv = mad_blob_store_env.MakeBlobStoreEnvWithoutStores(
			madderEnvLocal,
		)
	}

	return env, err
}

func (env Env) GetEnv() env_ui.Env {
	return env.Env
}

func (env Env) GetEnvBlobStore() mad_blob_store_env.BlobStoreEnv {
	return env.blobStoreEnv
}

func (env Env) GetConfigPublic() genesis_configs.TypedConfigPublic {
	return genesis_configs.TypedConfigPublic{
		Type: env.config.Type,
		Blob: env.config.Blob.GetGenesisConfigPublic(),
	}
}

func (env Env) GetObjectDigestType() string {
	return markl.GetDigestTypeForSigType(
		env.GetConfigPublic().Blob.GetObjectSigMarklTypeId(),
	)
}

func (env Env) GetConfigPrivate() genesis_configs.TypedConfigPrivate {
	return env.config
}

func (env Env) GetLockSmith() interfaces.LockSmith {
	return env.lockSmith
}

func (env Env) ResetCache() (err error) {
	if err = files.SetAllowUserChangesRecursive(env.DirDataIndex()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = os.RemoveAll(env.DirDataIndex()); err != nil {
		err = errors.Wrapf(err, "failed to remove verzeichnisse dir")
		return err
	}

	if err = env.MakeDirs(env.DirDataIndex()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = env.MakeDirs(env.DirIndexObjects()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = env.MakeDirs(env.DirIndexObjectPointers()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (env Env) GetStoreVersion() store_version.Version {
	if env.config.Blob == nil {
		return store_version.VCurrent
	} else {
		return env.config.Blob.GetStoreVersion()
	}
}

func (env Env) GetInventoryListBlobStore() mad_domain_interfaces.BlobStore {
	return env.GetDefaultBlobStore()
}

func (env Env) GetPathConfigSeed() interfaces.DirectoryLayoutPath {
	return env.GetXDG().Data.MakePath("config-seed")
}

// Delegation methods for the BlobStoreEnv API surface. With the
// embedded BlobStoreEnv replaced by a named field, the existing
// envRepo.GetDefaultBlobStore() / .GetBlobStores() / etc. call sites
// would otherwise stop compiling.

func (env Env) GetDefaultBlobStore() blob_stores.BlobStoreInitialized {
	return env.blobStoreEnv.GetDefaultBlobStore()
}

func (env Env) GetBlobStores() blob_stores.BlobStoreMap {
	return env.blobStoreEnv.GetBlobStores()
}

func (env Env) GetBlobStoresSorted() []blob_stores.BlobStoreInitialized {
	return env.blobStoreEnv.GetBlobStoresSorted()
}

func (env Env) GetBlobStore(
	id blob_store_id.Id,
) blob_stores.BlobStoreInitialized {
	return env.blobStoreEnv.GetBlobStore(id)
}

func (env Env) GetDefaultBlobStoreAndRemaining() (
	blob_stores.BlobStoreInitialized,
	blob_stores.BlobStoreMap,
) {
	return env.blobStoreEnv.GetDefaultBlobStoreAndRemaining()
}

// GetReadBlobStore returns a madder Multi blob store in write-through
// mode that reads from every enumerated blob store (default first,
// then walk-up ancestors and the XDG system store) and pins writes to
// the default store. Prefer this over GetDefaultBlobStore for any
// content-addressed read; the fallback prevents split-brain in
// multi-repo flows where a process holds two env_repo.Env instances
// rooted at different basePaths (clone, pull, push). See
// docs/features/0015-multi-store-blob-lookup.md.
func (env Env) GetReadBlobStore() mad_domain_interfaces.BlobStore {
	return env.makeReadBlobStore(nil)
}

// GetLocalReadBlobStore is GetReadBlobStore restricted to local blob
// stores: the default store plus only those fallback stores whose
// transport is local (never SFTP/WebDAV/S3). Use it for content-
// addressed reads that must not pay a network dial and that run before
// the user-configured blob-store order is known — notably the bootstrap
// mutable-config blob read, whose order can't be honored because the
// order is decoded from the very blob being read (see
// november/store_config/persist.go and issue #223). Remote stores are
// classified out from their stored config without initializing them, so
// excluding a remote store never dials it.
func (env Env) GetLocalReadBlobStore() mad_domain_interfaces.BlobStore {
	return env.makeReadBlobStore(isLocalBlobStore)
}

// makeReadBlobStore builds a madder Multi in write-through mode: writes
// pin to the default store; reads fall back across the other configured
// stores. The fallback list is built in a deterministic, user-controlled
// order — GetBlobStoresSorted honors the user-configured blob-store
// order (the repo config's blob-stores list, applied via
// SetBlobStoreOrder) when set and falls back to a stable id sort
// otherwise. This replaces ranging GetDefaultBlobStoreAndRemaining's
// BlobStoreMap, whose Go-randomized iteration order meant the store
// probed first on a default-store miss varied run to run — so a remote
// store could be probed before the local store holding the blob, paying
// a needless (and intermittent) network dial.
//
// includeReadStore, when non-nil, filters which fallback stores become
// read sources; a nil predicate includes them all.
func (env Env) makeReadBlobStore(
	includeReadStore func(blob_stores.BlobStoreInitialized) bool,
) mad_domain_interfaces.BlobStore {
	defaultStore := env.blobStoreEnv.GetDefaultBlobStore()
	defaultId := defaultStore.Path.GetId().String()
	sorted := env.blobStoreEnv.GetBlobStoresSorted()

	readStores := make(
		[]blob_stores.BlobStoreInitialized,
		0,
		len(sorted),
	)

	for _, store := range sorted {
		if store.Path.GetId().String() == defaultId {
			continue
		}
		if includeReadStore != nil && !includeReadStore(store) {
			continue
		}
		readStores = append(readStores, store)
	}

	multi, err := blob_stores.
		NewMulti(env.GetActiveContext()).
		WriteTo(defaultStore).
		Read(readStores...).
		ReadFill(false).
		Build()
	if err != nil {
		env.Cancel(errors.Wrap(err))
	}

	return multi
}

// isLocalBlobStore reports whether store's transport is local
// (filesystem-backed) rather than a network backend (SFTP/WebDAV/S3).
// It reads the store's stored config blob-store-type and does not touch
// the live BlobStore, so it never triggers a remote store's lazy
// connect. madder names every filesystem transport with a "local"
// prefix ("local", "local-inventory-archive", "local-pointer"); network
// transports use bare scheme names.
//
// Caveat: a "local-pointer" config can indirect to a remote store and
// is treated as local here; the principled fix (pinned provenance) is
// tracked by #223.
func isLocalBlobStore(store blob_stores.BlobStoreInitialized) bool {
	return strings.HasPrefix(store.Config.Blob.GetBlobStoreType(), "local")
}

func (env *Env) SetBlobStoreOrder(ids []blob_store_id.Id) {
	env.blobStoreEnv.SetBlobStoreOrder(ids)
}
