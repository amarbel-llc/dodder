package commands_dodder

import (
	"os"
	"path/filepath"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/echo/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_configs"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/values"
)

func init() {
	utility.AddCmd(
		"init-workspace",
		&InitWorkspace{},
	)
}

type InitWorkspace struct {
	command_components_dodder.Genesis
	repo.ImporterOptions
	command_components_dodder.Query

	complete command_components_dodder.Complete

	ExperimentalRepo      bool
	ParentPath            string
	Haustoria             string
	EmitInventoryListPath string
	DefaultQueryGroup     values.String
	Proto                 sku.Proto
}

var _ interfaces.CommandComponentWriter = (*InitWorkspace)(nil)

func (cmd InitWorkspace) GetDescription() command.Description {
	return command.Description{
		Short: "initialize a workspace directory",
	}
}

func (cmd *InitWorkspace) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{Args: []command.Arg{{
			Name:        "dir",
			Description: "directory for the workspace (created if needed)",
		}}},
		cmd.Query.GetArgGroup(),
	}
}

func (cmd *InitWorkspace) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(
		&cmd.ExperimentalRepo,
		"experimental-repo",
		true,
		"create a repo-backed workspace with its own inventory and commit history; blob storage points back to the parent repo",
	)

	flagSet.StringVar(
		&cmd.ParentPath,
		"parent",
		"",
		"path to a CWD-scoped parent dodder repository (omit for home repo)",
	)

	flagSet.StringVar(
		&cmd.Haustoria,
		"haustoria",
		"",
		"haustoria store type for external sync (caldav)",
	)

	flagSet.StringVar(
		&cmd.EmitInventoryListPath,
		"emit-inventory_list",
		"",
		"if set, write the inventory list selected by -query to this path (inventory_list-v2 format) before pulling; useful for offline reproduction with `der fsck -inventory_list-path`",
	)

	cmd.Genesis.SetFlagDefinitions(flagSet)
	cmd.ImporterOptions.SetFlagDefinitions(flagSet)
	cmd.Query.SetFlagDefinitions(flagSet)

	flagSet.Var(
		cmd.complete.GetFlagValueMetadataTags(&cmd.Proto.Metadata),
		"tags",
		"tags added for new objects in `checkin`, `new`, `organize`",
	)

	flagSet.Var(
		cmd.complete.GetFlagValueMetadataType(&cmd.Proto.Metadata),
		"type",
		"type used for new objects in `new` and `organize`",
	)

	flagSet.Var(
		cmd.complete.GetFlagValueStringTags(&cmd.DefaultQueryGroup),
		"query",
		"default query for `show`",
	)
}

func (cmd InitWorkspace) Complete(
	_ command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	searchDir := envLocal.GetCwd()

	if commandLine.InProgress != "" && files.Exists(commandLine.InProgress) {
		var err error

		if commandLine.InProgress, err = filepath.Abs(commandLine.InProgress); err != nil {
			envLocal.Cancel(err)
			return
		}

		if searchDir, err = filepath.Rel(searchDir, commandLine.InProgress); err != nil {
			envLocal.Cancel(err)
			return
		}
	}

	for dirEntry, err := range files.WalkDir(searchDir) {
		if err != nil {
			envLocal.Cancel(err)
			return
		}

		if !dirEntry.IsDir() {
			continue
		}

		if files.WalkDirIgnoreFuncHidden(dirEntry) {
			continue
		}

		envLocal.GetUI().Printf("%s/\tdirectory", dirEntry.RelPath)
	}
}

func (cmd InitWorkspace) Run(req command.Request) {
	if !cmd.ExperimentalRepo && cmd.ParentPath != "" {
		req.Cancel(
			errors.BadRequestf(
				"-parent cannot be used with -experimental-repo=false",
			),
		)
		return
	}

	if cmd.ExperimentalRepo {
		cmd.runExperimentalRepo(req)
		return
	}

	cmd.runLightweight(req)
}

func (cmd InitWorkspace) runLightweight(req command.Request) {
	envLocal := cmd.Genesis.MakeEnv(req)

	switch req.RemainingArgCount() {
	case 0:
		break

	case 1:
		dir := req.PopArg("dir")

		if err := envLocal.MakeDirs(dir); err != nil {
			req.Cancel(err)
			return
		}

		if err := os.Chdir(dir); err != nil {
			req.Cancel(err)
			return
		}
	}

	req.AssertNoMoreArgs()

	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	blob := &workspace_config_blobs.V0{
		Query: cmd.DefaultQueryGroup.String(),
		Defaults: repo_configs.DefaultsV1OmitEmpty{
			Type: cmd.Proto.Metadata.GetType().ToType(),
			Tags: slices.Collect(ids.ITagSeqToTagStructSeq(cmd.Proto.Metadata.AllTags())),
		},
	}

	if err := localWorkingCopy.GetEnvWorkspace().CreateWorkspace(
		blob,
	); err != nil {
		req.Cancel(err)
	}
}

func (cmd InitWorkspace) runExperimentalRepo(req command.Request) {
	config := req.Utility.GetConfigAny().(*repo_config_cli.Config)

	if !repo_id.IsAuto(config.RepoId) {
		req.Cancel(
			errors.BadRequestf(
				"-repo_id cannot be used with -experimental-repo (workspace repos are always CWD-rooted)",
			),
		)
		return
	}

	config.RepoId = repo_id.CwdDefault()

	absParentPath, parentIsHomeRepo := cmd.resolveParentPath(req)
	cmd.validateParentRepo(req, absParentPath, parentIsHomeRepo)

	cmd.Genesis.BigBang.ExcludeDefaultType = true
	cmd.linkParentZettelIdProviders(absParentPath, parentIsHomeRepo)

	workspaceRepoIdString := req.PopArg("workspace repo id")
	cmd.setupParentPointerBlobStore(
		req,
		workspaceRepoIdString,
		absParentPath,
		parentIsHomeRepo,
	)

	local := cmd.OnTheFirstDay(req)

	remote := cmd.makeParentRemote(req, local, absParentPath, parentIsHomeRepo)

	queryArgs := req.PopArgs()

	// Build the query against `remote` (the parent repo with all the tag
	// definitions), not `local` (the brand-new empty workspace). Tag-name
	// queries depend on the tag's typed-blob being readable through the
	// objectProbeIndex (queries/build_state.go:466) — without it the tag
	// expression collapses to a permissive bare ObjectId match that
	// matches by partial tag-path comparison and pulls in unrelated
	// objects. The new workspace has no tag objects yet so it cannot
	// resolve `project-X` to its tag definition.
	queryGroup := cmd.MakeQuery(
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

	if cmd.EmitInventoryListPath != "" {
		cmd.emitInventoryList(req, local, remote, queryGroup)
	}

	if err := local.PullQueryGroupFromRemote(
		remote,
		queryGroup,
		cmd.WithPrintCopies(true),
	); err != nil {
		req.Cancel(err)
		return
	}

	var parentPathForConfig string
	if !parentIsHomeRepo {
		parentPathForConfig = absParentPath
	}

	// #287b: pin the parent repo's identity so later push/pull can verify the
	// resolved parent is the same repo, not a different one that later occupied
	// the path. Stored in StringWithFormat() form (the `ed25519_pub-...` shape).
	parentPubkey := remote.GetImmutableConfigPublic().GetPublicKey().StringWithFormat()

	v1 := workspace_config_blobs.V1{
		V0: workspace_config_blobs.V0{
			Query: cmd.DefaultQueryGroup.String(),
			Defaults: repo_configs.DefaultsV1OmitEmpty{
				Type: cmd.Proto.Metadata.GetType().ToType(),
				Tags: slices.Collect(
					ids.ITagSeqToTagStructSeq(cmd.Proto.Metadata.AllTags()),
				),
			},
		},
		ParentPath:   parentPathForConfig,
		ParentPubkey: parentPubkey,
	}

	var blob workspace_config_blobs.Config

	if cmd.Haustoria != "" {
		blob = cmd.makeHaustoriaConfig(req, v1)
	} else {
		blob = &v1
	}

	if err := local.GetEnvWorkspace().CreateWorkspace(blob); err != nil {
		req.Cancel(err)
	}

	if err := local.GetEnvWorkspace().UpdateSyncBaseline(
		local.GetInventoryListStore(),
	); err != nil {
		req.Cancel(err)
	}
}

func (cmd InitWorkspace) resolveParentPath(
	req command.Request,
) (absPath string, isHomeRepo bool) {
	if cmd.ParentPath != "" {
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

// parentRepoMetadataDir returns the parent repo's dodder-metadata
// directory. FDR-0019: dodder metadata nests under repos/<name>/, and a
// path-addressed or implicit (home) parent carries no -repo_id so it
// resolves to the default repo (matching makeParentRemote). For a home
// parent, absParentPath is already the dodder data dir
// (<dataHome>/dodder); for a -parent path it is the repo root containing
// .dodder/local/share.
func parentRepoMetadataDir(absParentPath string, isHomeRepo bool) string {
	dataRoot := absParentPath
	if !isHomeRepo {
		dataRoot = filepath.Join(
			absParentPath,
			"."+dodder_env.XDGUtilityName,
			"local", "share",
		)
	}

	return filepath.Join(dataRoot, "repos", repo_id.DefaultName)
}

func (cmd InitWorkspace) validateParentRepo(
	req command.Request,
	absPath string,
	isHomeRepo bool,
) {
	inventoryListLog := filepath.Join(
		parentRepoMetadataDir(absPath, isHomeRepo),
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

func (cmd InitWorkspace) makeParentRemote(
	req command.Request,
	local *local_working_copy.Repo,
	absParentPath string,
	isHomeRepo bool,
) repo.Repo {
	if isHomeRepo {
		config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

		home, err := os.UserHomeDir()
		if err != nil {
			req.Cancel(err)
		}

		ownDir := env_dir.MakeWithHomeAndInitialize(
			req,
			dodder_env.XDGUtilityName,
			home,
			config.Debug,
			repo_id.DefaultName,
		)

		madderDir := env_dir.MakeWithHomeAndInitialize(
			req,
			command_components_dodder.XDGUtilityNameMadder,
			home,
			config.Debug,
			repo_id.DefaultName,
		)

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

	var remote command_components_dodder.Remote
	remote.DirectPath = absParentPath
	return remote.MakeDirectRemoteFromPath(req, local)
}

// linkParentZettelIdProviders sets BigBang.Yin and BigBang.Yang to the parent
// repo's word list files when neither flag was explicitly provided. This allows
// workspace repos to create new zettels using the parent's ID space without
// requiring the user to maintain separate word lists.
func (cmd InitWorkspace) makeHaustoriaConfig(
	req command.Request,
	v1 workspace_config_blobs.V1,
) *workspace_config_blobs.V2 {
	switch cmd.Haustoria {
	case "caldav":
		cfg := workspace_config_blobs.CalDAVConfig{}

		// Read from env vars as defaults; the config can override later.
		resolved, err := cfg.Resolve()
		if err != nil {
			req.Cancel(
				errors.BadRequestf(
					"CalDAV configuration incomplete: %s", err,
				),
			)
			return nil
		}

		// Store URL and username in config; password stays in env only.
		cfg.URL = resolved.URL
		cfg.Username = resolved.Username

		return &workspace_config_blobs.V2{
			V1: v1,
			Haustoria: workspace_config_blobs.HaustoriaConfig{
				Type:   "caldav",
				CalDAV: &cfg,
				Calendars: map[string]workspace_config_blobs.CalendarConfig{
					"default": {
						URL:  resolved.URL,
						Type: "!task",
					},
				},
			},
		}

	case "orgmode":
		orgCfg := workspace_config_blobs.OrgmodeConfig{}

		resolved, err := orgCfg.ResolveOrgmode()
		if err != nil {
			req.Cancel(
				errors.BadRequestf(
					"Orgmode configuration incomplete: %s", err,
				),
			)
			return nil
		}

		switch resolved.Transport {
		case "webdav":
			orgCfg.Transport = "webdav"
			orgCfg.WebDAV = &workspace_config_blobs.OrgmodeWebDAV{
				URL:      resolved.WebDAVURL,
				Username: resolved.WebDAVUsername,
			}

		case "sftp":
			orgCfg.Transport = "sftp"
			orgCfg.SFTP = &workspace_config_blobs.OrgmodeSFTP{
				Host:           resolved.SFTPHost,
				Port:           resolved.SFTPPort,
				User:           resolved.SFTPUser,
				PrivateKeyPath: resolved.SFTPPrivateKeyPath,
			}
		}

		defaultPath := resolved.WebDAVURL
		if resolved.Transport == "sftp" {
			defaultPath = "/org"
		}

		return &workspace_config_blobs.V2{
			V1: v1,
			Haustoria: workspace_config_blobs.HaustoriaConfig{
				Type:    "orgmode",
				Orgmode: &orgCfg,
				Folders: map[string]workspace_config_blobs.FolderConfig{
					"default": {
						Path: defaultPath,
						Type: "!md",
					},
				},
			},
		}

	default:
		req.Cancel(
			errors.BadRequestf(
				"unknown haustoria type: %s (supported: caldav, orgmode)",
				cmd.Haustoria,
			),
		)
		return nil
	}
}

// setupParentPointerBlobStore configures Genesis.BigBang so that
// init-workspace -experimental-repo writes a TomlPointerV1 instead of
// a freshly-initialized local-hash-bucketed store (#200). The pointer
// resolves to the parent repo's default blob store, so blob reads
// (e.g. parent's konfig) flow through to where the parent actually
// stores them.
//
// The pointer id is "." + workspaceRepoIdString (CWD-scoped, prefixed
// with "." per dodder's blob_store_id convention). The base path is
// the parent's <madder>/blob_stores/default directory — computed from
// absParentPath and the madder XDG utility name. For the home-repo
// parent, absParentPath is <dataHome>/dodder; the madder sibling is
// <dataHome>/madder. For the -parent path, the parent dir contains
// .madder/local/share/ alongside .dodder/local/share/.
//
// TODO https://github.com/amarbel-llc/dodder/issues/219
// construct a parent env_dir for the madder utility and ask it
// for the blob_stores path, rather than hardcoding the layout
// here — the hardcoded form may diverge from what
// env_dir.MakeDefaultAndInitialize produces under non-default
// XDG env vars (notably the bats sandbox's XDG_DATA_HOME override).
func (cmd *InitWorkspace) setupParentPointerBlobStore(
	req command.Request,
	workspaceRepoIdString string,
	absParentPath string,
	isHomeRepo bool,
) {
	var parentBlobStoreBasePath string
	if isHomeRepo {
		// absParentPath = <dataHome>/dodder; sibling = <dataHome>/madder.
		parentBlobStoreBasePath = filepath.Join(
			filepath.Dir(absParentPath),
			command_components_dodder.XDGUtilityNameMadder,
			"blob_stores", "default",
		)
	} else {
		parentBlobStoreBasePath = filepath.Join(
			absParentPath,
			"."+command_components_dodder.XDGUtilityNameMadder,
			"local", "share",
			"blob_stores", "default",
		)
	}

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

func (cmd *InitWorkspace) linkParentZettelIdProviders(
	absParentPath string,
	isHomeRepo bool,
) {
	if cmd.Genesis.BigBang.Yin != "" || cmd.Genesis.BigBang.Yang != "" {
		return
	}

	parentObjectIdDir := filepath.Join(
		parentRepoMetadataDir(absParentPath, isHomeRepo),
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

func (cmd InitWorkspace) emitInventoryList(
	req command.Request,
	local *local_working_copy.Repo,
	remote repo.Repo,
	queryGroup *queries.Query,
) {
	list, err := remote.MakeInventoryList(queryGroup)
	if err != nil {
		req.Cancel(err)
		return
	}

	file, err := os.Create(cmd.EmitInventoryListPath)
	if err != nil {
		req.Cancel(err)
		return
	}

	defer errors.ContextMustClose(req, file)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(file)
	defer repoolBufferedWriter()
	defer errors.ContextMustFlush(req, bufferedWriter)

	closet := local.GetInventoryListCoderCloset()
	remoteBlobStore := remote.GetBlobStore()

	combined := func(yield func(*sku.Transacted, error) bool) {
		// Pass 1: pointer objects (the inventory_list-v2 manifests).
		for sk := range list.All() {
			if !yield(sk, nil) {
				return
			}
		}

		// Pass 2: dereferenced blob contents (the actual zettels, types,
		// tags inside each manifest). Each blob is the raw doddish stream
		// for the manifest's type; the type comes from the pointer object,
		// not a hyphence header. Decoding runs the type's afterDecoding
		// hook (which is FinalizeAndVerify for v2), so we recover from
		// ErrAfterDecoding and emit the object anyway — preserving the
		// failing-verify state for downstream debugging.
		//
		// Only inventory_list objects have a dereferenceable manifest
		// blob. Other genres (zettel, type, tag) have content blobs that
		// aren't object lists, so we skip them in this pass.
		for sk := range list.All() {
			if sk.GetGenre() != genres.InventoryList {
				continue
			}

			blobDigest := sk.GetBlobDigest()
			if blobDigest.IsNull() {
				continue
			}

			reader, openErr := remoteBlobStore.MakeBlobReader(blobDigest)
			if openErr != nil {
				if !yield(nil, errors.Wrapf(openErr, "blob: %s", blobDigest)) {
					return
				}
				continue
			}

			listCoder := closet.GetCoderForType(sk.GetType().ToType())
			bufferedReader, repoolBR := pool.GetBufferedReader(reader)

			for {
				inner, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

				_, decodeErr := listCoder.DecodeFrom(inner, bufferedReader)
				if decodeErr != nil {
					if errors.IsEOF(decodeErr) {
						break
					}
					if errors.Is(decodeErr, inventory_list_coders.ErrAfterDecoding{}) {
						if !yield(inner, nil) {
							repoolBR()
							reader.Close()
							return
						}
						continue
					}
					if !yield(nil, errors.Wrapf(decodeErr, "blob: %s", blobDigest)) {
						repoolBR()
						reader.Close()
						return
					}
					break
				}

				if !yield(inner, nil) {
					repoolBR()
					reader.Close()
					return
				}
			}

			repoolBR()
			reader.Close()
		}
	}

	if _, err := closet.WriteTypedBlobToWriter(
		req,
		ids.GetOrPanic(local.GetImmutableConfigPublic().GetInventoryListTypeId()).TypeStruct,
		combined,
		bufferedWriter,
	); err != nil {
		req.Cancel(err)
		return
	}
}
