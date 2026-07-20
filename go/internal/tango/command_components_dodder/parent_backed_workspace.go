package command_components_dodder

import (
	"os"
	"path/filepath"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/echo/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_configs"
	mad_env_dir "code.linenisgreat.com/madder/go/pkgs/env_dir"
	env_local "code.linenisgreat.com/madder/go/pkgs/env_local"
	"code.linenisgreat.com/madder/go/pkgs/scoped_id"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/files"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/xdg"
)

// ParentBackedWorkspace bundles the create-a-repo-backed-workspace-pointing-at-a-parent
// sequence shared by `init-workspace -experimental-repo` and `edit -ephemeral`:
// resolve the parent repo (explicit -parent path or the home repo), wire a
// TomlPointerV1 blob store at the parent (FDR-0005, no blob copy), link the
// parent's zettel-id word lists, genesis the workspace repo (CWD-rooted via
// Genesis.OnTheFirstDay), and pull the queried objects from the parent.
//
// The pieces are exposed individually so callers that interleave extra steps
// (init-workspace's -organize / -emit-inventory_list) can drive them directly;
// CreateRepoAndPullFromParent is the straight-line convenience used by the
// ephemeral path.
type ParentBackedWorkspace struct {
	Genesis
	Query

	// ParentPath is the explicit -parent path; empty means the home repo.
	ParentPath string

	// ParentRepoId, when non-empty, selects the parent repo by id via the
	// FDR-0019 scope resolver — the same mechanism `show`/`cat-alfred`/the MCP
	// use — instead of by -parent path or the home default. Takes precedence
	// over ParentPath. It is the id's FULL spelling (scoped_id.String()): a
	// bare name like "work" (XDG-user scope) or a cwd-scoped spelling like
	// ".notes" / "..notes" (nearest / Nth-ancestor .dodder). parentConfig
	// re-parses it via RepoId.Set, so the leading dots route through
	// MakeOperateEnvDir's cwd branch. Empty leaves the -parent/home behavior.
	ParentRepoId string
}

// ResolveParentPath mirrors init-workspace's parent resolution: an explicit
// -parent path (made absolute), an XDG-user -repo_id (resolved under the home
// XDG data dir, same root as the home repo — only the repos/<name> nesting
// differs, applied by the resolver via parentConfig), or, when both are unset,
// the home default repo.
func (cmd ParentBackedWorkspace) ResolveParentPath(
	req command.Request,
) (absPath string, isHomeRepo bool) {
	// A -parent path takes precedence only when no -repo_id was given; an
	// XDG-user repo-id resolves under the home XDG data dir (isHomeRepo shape).
	if cmd.ParentPath != "" && cmd.ParentRepoId == "" {
		absPath = cmd.ParentPath
		if !filepath.IsAbs(absPath) {
			var err error
			if absPath, err = filepath.Abs(absPath); err != nil {
				req.Cancel(err)
			}
		}
		return absPath, false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		req.Cancel(err)
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}

	absPath = filepath.Join(dataHome, dodder_env.XDGUtilityName)
	return absPath, true
}

// parentDodderMetadataDir returns the parent repo's dodder-metadata directory
// (<dataRoot>/repos/<repoName>) by building the parent's dodder env_dir via the
// FDR-0019 resolver and reading its data dir — the resolver applies the
// repos/<name> nesting, so no hardcoded layout math is needed here. Replaces
// the former ParentRepoMetadataDir + parentRepoName pair.
//
// noInit selects the side-effect-free resolver (MakeOperateEnvDirNoInit) for
// callers that must compute the path for a possibly-nonexistent root without
// mkdir'ing it — namely ValidateParentRepo, which runs before the parent is
// known to exist. Post-validation callers pass false (the repo exists, so the
// initializing resolver's mkdir is a no-op).
func (cmd ParentBackedWorkspace) parentDodderMetadataDir(
	req command.Request,
	absParentPath string,
	isHomeRepo bool,
	noInit bool,
) string {
	config := cmd.parentConfig(req, absParentPath, isHomeRepo)

	var parentDodderEnv mad_env_dir.Env
	if noInit {
		parentDodderEnv = MakeOperateEnvDirNoInit(req, config, dodder_env.XDGUtilityName)
	} else {
		parentDodderEnv = MakeOperateEnvDir(req, config, dodder_env.XDGUtilityName)
	}

	return parentDodderEnv.GetXDG().Data.ActualValue
}

// ValidateParentRepo cancels the request when no dodder repo exists at the
// resolved parent path.
func (cmd ParentBackedWorkspace) ValidateParentRepo(
	req command.Request,
	absPath string,
	isHomeRepo bool,
) {
	// noInit: this runs before the parent is known to exist, so use the
	// side-effect-free resolver — the initializing form would mkdir under a
	// nonexistent -parent path (surfacing a raw "mkdir ...: read-only file
	// system" instead of the clean error below, and creating dirs under a bad
	// path). madder#260 added the no-init XDG-root-override constructor this
	// relies on.
	inventoryListLog := filepath.Join(
		cmd.parentDodderMetadataDir(req, absPath, isHomeRepo, true),
		"inventory_lists_log",
	)

	if !files.Exists(inventoryListLog) {
		if isHomeRepo {
			req.Cancel(
				errors.BadRequestf(
					"no dodder repo found at %s; run `dodder init` first",
					absPath,
				),
			)
		} else {
			req.Cancel(
				errors.BadRequestf(
					"no dodder repo found at -parent path %s",
					absPath,
				),
			)
		}
	}
}

// MakeParentRemote opens the resolved parent repo as a remote. For the home /
// -repo_id parent it builds the dodder + madder env_dirs through the FDR-0019
// resolver (parentConfig -> MakeOperateEnvDir), the same mechanism the metadata
// and blob-store paths now use; for a -parent path it uses the
// direct-remote-from-path machinery.
func (cmd ParentBackedWorkspace) MakeParentRemote(
	req command.Request,
	local *local_working_copy.Repo,
	absParentPath string,
	isHomeRepo bool,
) repo.Repo {
	if isHomeRepo {
		config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

		parentConfig := cmd.parentConfig(req, absParentPath, isHomeRepo)

		// The resolver applies the repos/<name> nesting for the dodder slot and
		// keeps the madder blob pool flat (configFor blanks the madder repo
		// name) — replacing the hand-built MakeWithHomeAndInitialize +
		// parentRepoName/DefaultName split.
		ownDir := MakeOperateEnvDir(req, parentConfig, dodder_env.XDGUtilityName)
		madderDir := MakeOperateEnvDir(req, parentConfig, XDGUtilityNameMadder)

		envUI := env_ui.Make(
			req,
			config,
			config.Debug,
			env_ui.Options{},
		)

		return local_working_copy.Make(
			env_local.Make(envUI, ownDir),
			env_local.Make(envUI, madderDir),
			local_working_copy.OptionsEmpty,
		)
	}

	var remote Remote
	remote.DirectPath = absParentPath
	return remote.MakeDirectRemoteFromPath(req, local)
}

// LinkParentZettelIdProviders sets BigBang.Yin and BigBang.Yang to the parent
// repo's word list files when neither flag was explicitly provided. This allows
// workspace repos to create new zettels using the parent's ID space without
// requiring the user to maintain separate word lists.
func (cmd *ParentBackedWorkspace) LinkParentZettelIdProviders(
	req command.Request,
	absParentPath string,
	isHomeRepo bool,
) {
	if cmd.Genesis.BigBang.Yin != "" || cmd.Genesis.BigBang.Yang != "" {
		return
	}

	parentObjectIdDir := filepath.Join(
		cmd.parentDodderMetadataDir(req, absParentPath, isHomeRepo, false),
		"object_ids",
	)

	parentYin := filepath.Join(
		parentObjectIdDir,
		zettel_id_provider.FilePathZettelIdYin,
	)

	parentYang := filepath.Join(
		parentObjectIdDir,
		zettel_id_provider.FilePathZettelIdYang,
	)

	if files.Exists(parentYin) {
		cmd.Genesis.BigBang.Yin = parentYin
	}

	if files.Exists(parentYang) {
		cmd.Genesis.BigBang.Yang = parentYang
	}
}

// parentConfig builds a repo_config_cli.Config that addresses the resolved
// parent repo through MakeOperateEnvDir (the FDR-0019 resolver): a -parent path
// becomes a BasePath override (-dir-dodder shape, #343 step 1), an XDG-user
// -repo_id becomes the RepoId scope, and the home parent is left at the default
// scope. Feeding this to MakeOperateEnvDir yields the parent's real on-disk
// dodder / madder env_dirs without the hardcoded repos/<name> / madder-flat
// layout math the path helpers used to carry.
func (cmd ParentBackedWorkspace) parentConfig(
	req command.Request,
	absParentPath string,
	isHomeRepo bool,
) repo_config_cli.Config {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	// The ephemeral CWD-scope RepoId the caller set for genesis must not leak
	// into the parent resolution; reset it to the parent's scope.
	config.RepoId = scoped_id.Id{}
	config.BasePath = ""

	switch {
	case cmd.ParentRepoId != "":
		if err := config.RepoId.Set(cmd.ParentRepoId); err != nil {
			req.Cancel(err)
		}

		// A cwd-scoped parent id (.name / ..name) can't be resolved in the
		// ephemeral flow: RunEphemeral chdirs into the temp dir and re-pins the
		// ceiling to it before the post-chdir resolver calls (pointer blob store,
		// MakeParentRemote), so the cwd walk-up that DEFINES a cwd-scoped id no
		// longer finds the ancestor .dodder/. Reject it up front with a clear
		// error rather than silently falling back to the home repo. Resolving the
		// ancestor to an absolute path BEFORE the chdir is tracked as #351.
		if config.RepoId.GetLocationType() == scoped_id.LocationTypeCwd {
			req.Cancel(errors.BadRequestf(
				"cwd-scoped -repo_id %q is not supported for -ephemeral; "+
					"use an XDG-user repo id (e.g. `work`) or -parent <path>",
				cmd.ParentRepoId,
			))
		}

	case !isHomeRepo:
		// A -parent path: root the resolver at it (BasePath override), default
		// scope selects repos/<default> within.
		config.BasePath = absParentPath

	default:
		// Home parent: pin to the XDG-USER default repo explicitly. A zero/auto
		// RepoId would take MakeOperateEnvDir's default branch, whose cwd walk-up
		// roots at the current workspace's own .dodder/ (the ephemeral flow runs
		// from inside the workspace dir) instead of the home repo under
		// $XDG_DATA_HOME. An explicit XDG-user scope pins to the user home with
		// NO walk-up — matching the former MakeWithHomeAndInitialize behavior.
		config.RepoId = scoped_id.MakeWithLocation(
			repo_id.DefaultName,
			scoped_id.LocationTypeXDGUser,
		)
	}

	return config
}

// SetupParentPointerBlobStore configures Genesis.BigBang so that genesis writes
// a TomlPointerV1 instead of a freshly-initialized local-hash-bucketed store
// (#200). The pointer resolves to the parent repo's default blob store, so blob
// reads (e.g. parent's konfig) flow through to where the parent actually stores
// them.
//
// The pointer id is "." + workspaceRepoIdString (CWD-scoped, prefixed with "."
// per dodder's blob_store_id convention). The base path is the parent's
// default-blob-store dir, obtained by building the parent's madder env_dir via
// the FDR-0019 resolver (parentConfig → MakeOperateEnvDir) and joining
// blob_stores/default under its data dir — replacing the previously hardcoded
// <madder>/blob_stores/default math, which could diverge from what the resolver
// produces under a non-default XDG_DATA_HOME (the bats sandbox). Closes #219.
func (cmd *ParentBackedWorkspace) SetupParentPointerBlobStore(
	req command.Request,
	workspaceRepoIdString string,
	absParentPath string,
	isHomeRepo bool,
) {
	parentMadderEnv := MakeOperateEnvDir(
		req,
		cmd.parentConfig(req, absParentPath, isHomeRepo),
		XDGUtilityNameMadder,
	)

	parentBlobStoreBasePath := filepath.Join(
		parentMadderEnv.GetXDG().Data.ActualValue,
		"blob_stores", "default",
	)

	if !files.Exists(parentBlobStoreBasePath) {
		req.Cancel(
			errors.BadRequestf(
				"parent repo has no default blob store at %s",
				parentBlobStoreBasePath,
			),
		)
		return
	}

	pointerId := "." + workspaceRepoIdString

	if err := cmd.Genesis.BigBang.BlobStoreId.Set(pointerId); err != nil {
		req.Cancel(err)
		return
	}

	pointerConfig := &blob_store_configs.TomlPointerV1{
		BasePath: parentBlobStoreBasePath,
	}

	cmd.Genesis.BigBang.BlobStoreConfigInit = &blob_store_configs.TypedMutableConfig{
		Type: blob_store_configs.TypeStructForConfig(pointerConfig),
		Blob: pointerConfig,
	}
}

// CreateRepoAndPullFromParent is the straight-line create-and-pull sequence used
// by the ephemeral path. It genesis-es the workspace repo (CWD-rooted; the
// caller is responsible for chdir-ing into the temp dir first), opens the parent
// as a remote, builds the query against the parent (which holds the tag/type
// definitions), and pulls the matching objects into the new repo. Both the new
// local repo and the parent remote are returned so the caller can drive a
// subsequent push back to the parent.
func (cmd ParentBackedWorkspace) CreateRepoAndPullFromParent(
	req command.Request,
	absParentPath string,
	isHomeRepo bool,
	queryArgs []string,
	importerOptions repo.ImporterOptions,
) (local *local_working_copy.Repo, remote repo.Repo) {
	local = cmd.Genesis.OnTheFirstDay(req)

	remote = cmd.MakeParentRemote(req, local, absParentPath, isHomeRepo)

	// Build the query against `remote` (the parent repo with all the tag
	// definitions), not `local` (the brand-new empty workspace). Tag-name
	// queries depend on the tag's typed-blob being readable through the
	// objectProbeIndex — without it the tag expression collapses to a
	// permissive bare ObjectId match. The new workspace has no tag objects
	// yet so it cannot resolve `project-X` to its tag definition.
	queryGroup := cmd.Query.MakeQuery(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
		),
		remote,
		queryArgs,
	)

	if err := local.PullQueryGroupFromRemote(
		remote,
		queryGroup,
		importerOptions,
	); err != nil {
		req.Cancel(err)
		return local, remote
	}

	return local, remote
}

// RunEphemeral drives the FDR-0023 ephemeral-workspace lifecycle shared by
// `edit -ephemeral` and `new -ephemeral`: materialize a temp repo-backed
// workspace whose blob store points at the resolved parent (zero-copy), pull
// the objects named by pullQueryArgs, run the caller's work against a fresh
// writable working copy of that workspace, push the result back to the parent,
// then tear the temp workspace down. On any failure after the workspace is
// created, the temp workspace is PRESERVED and its path surfaced so no work is
// lost.
//
// The CALLER must, before calling, have (a) set ParentPath / ParentRepoId to
// select the parent and (b) overwritten config.RepoId with repo_id.CwdDefault()
// via the shared *repo_config_cli.Config pointer, so genesis roots the
// ephemeral repo (and its pointer blob store) inside the temp dir — see the
// per-command runEphemeral for why the config must be the shared pointer, not a
// FromAny copy.
//
// work receives a fresh in-process working copy of the temp workspace (opened
// after CreateWorkspace, so it resolves as writable rather than the genesis
// env's read-only temporary). edit uses it to run the checkout/edit; new opens
// its own via runInWorkspace and ignores the argument — either is fine, both
// operate on the same temp workspace.
func (cmd ParentBackedWorkspace) RunEphemeral(
	req command.Request,
	pullQueryArgs []string,
	work func(edited *local_working_copy.Repo) error,
) {
	cmd.Genesis.BigBang.SetDefaults()

	// Workspace repos have no default type (matches init-workspace
	// -experimental-repo); creating new objects requires an explicit -type,
	// editing existing ones does not.
	cmd.Genesis.BigBang.ExcludeDefaultType = true

	absParentPath, parentIsHomeRepo := cmd.ResolveParentPath(req)
	cmd.ValidateParentRepo(req, absParentPath, parentIsHomeRepo)
	cmd.LinkParentZettelIdProviders(req, absParentPath, parentIsHomeRepo)

	originalCwd, err := os.Getwd()
	if err != nil {
		req.Cancel(err)
		return
	}

	tempDir, err := os.MkdirTemp("", "dodder-ephemeral-")
	if err != nil {
		req.Cancel(err)
		return
	}

	// Resolve symlinks so tempDir matches os.Getwd()'s canonical form after
	// chdir (macOS $TMPDIR is a symlink); the ceiling comparison is
	// path-equality based, so the two must agree.
	if resolved, resolveErr := filepath.EvalSymlinks(tempDir); resolveErr == nil {
		tempDir = resolved
	}

	if err = os.Chdir(tempDir); err != nil {
		req.Cancel(err)
		return
	}

	// Pin both ceilings to the temp dir for the rest of this process. The temp
	// dir sits under $TMPDIR, typically OUTSIDE the caller's ceiling; the fresh
	// MakeLocalWorkingCopy below discovers the workspace by walking UP from cwd
	// for .dodder-workspace, and that walk-up honors the ceiling. Without
	// re-pinning, the caller's ceiling (which may sit below tempDir under a
	// sandbox) would cut the walk short and the workspace would not be found.
	if err = os.Setenv(
		xdg.CeilingEnvVarName(dodder_env.XDGUtilityName),
		tempDir,
	); err != nil {
		req.Cancel(err)
		return
	}

	if err = os.Setenv(
		xdg.CeilingEnvVarName(dodder_env.XDGUtilityNameMadder),
		tempDir,
	); err != nil {
		req.Cancel(err)
		return
	}

	// FDR-0005 / FDR-0023: wire a TomlPointerV1 blob store pointing at the
	// parent's default store so the workspace repo holds NO blob copy of its
	// own. Because config.RepoId is CWD-scoped (set by the caller via the
	// shared *Config pointer), genesis roots .dodder/.madder AND this pointer
	// store inside tempDir, so the fresh in-process working copy discovers it.
	cmd.SetupParentPointerBlobStore(
		req,
		"ephemeral",
		absParentPath,
		parentIsHomeRepo,
	)

	local, remote := cmd.CreateRepoAndPullFromParent(
		req,
		absParentPath,
		parentIsHomeRepo,
		pullQueryArgs,
		repo.ImporterOptions{}.WithPrintCopies(true),
	)

	// Seed the ephemeral workspace config with the PARENT's resolved defaults
	// (its mutable⊕workspace overlay), so `new` in the ephemeral workspace
	// gets the parent's default type without an explicit -type — the ephemeral
	// repo itself has no default type (ExcludeDefaultType above). #15.
	//
	// GetEnvWorkspace lives on repo.LocalRepo, not the base repo.Repo. Every
	// parent-remote branch (home, -repo_id, -parent path) currently builds a
	// *local_working_copy.Repo, so the assertion holds; a future non-local
	// remote parent would just skip the seed (empty defaults, -type required).
	ephemeralConfig := &workspace_config_blobs.V0{}

	if localRemote, ok := remote.(repo.LocalRepo); ok {
		parentDefaults := localRemote.GetEnvWorkspace().GetDefaults()
		ephemeralConfig.Defaults = repo_configs.DefaultsV1OmitEmpty{
			Type: parentDefaults.GetDefaultType(),
			Tags: slices.Collect(parentDefaults.GetDefaultTags().All()),
		}
	}

	if err = local.GetEnvWorkspace().CreateWorkspace(ephemeralConfig); err != nil {
		req.Cancel(err)
		return
	}

	// Flush the genesis repo and open a fresh working copy of the now-written
	// temp workspace: the genesis env was built before .dodder-workspace
	// existed, so it still resolves as a read-only temporary workspace. The
	// fresh open resolves it as writable and re-discovers the pointer blob
	// store from tempDir.
	if err = local.Flush(); err != nil {
		req.Cancel(err)
		return
	}

	edited := cmd.MakeLocalWorkingCopy(req)

	if err = work(edited); err != nil {
		// Preserve the temp workspace so the work is not lost (FDR-0023).
		req.Cancel(
			errors.Wrapf(err, "ephemeral work failed; workspace kept at %s", tempDir),
		)
		return
	}

	// Push back to the parent: the remote pulls the whole workspace from the
	// ephemeral local (push is "remote pulls from local"; mirrors push.go).
	pushQueryGroup := cmd.Query.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		edited,
		nil,
	)

	if err = remote.PullQueryGroupFromRemote(
		edited,
		pushQueryGroup,
		repo.ImporterOptions{}.WithPrintCopies(true),
	); err != nil {
		// Preserve the temp workspace so the work is not lost (FDR-0023).
		req.Cancel(
			errors.Wrapf(err, "ephemeral push failed; workspace kept at %s", tempDir),
		)
		return
	}

	// Teardown only on success.
	if err = os.Chdir(originalCwd); err != nil {
		req.Cancel(err)
		return
	}

	if err = os.RemoveAll(tempDir); err != nil {
		req.Cancel(err)
		return
	}
}
