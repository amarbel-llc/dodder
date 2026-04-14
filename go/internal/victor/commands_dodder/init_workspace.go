package commands_dodder

import (
	"os"
	"path/filepath"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/echo/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/quebec/repo"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/values"
)

func init() {
	utility.AddCmd(
		"init-workspace",
		&InitWorkspace{})
}

type InitWorkspace struct {
	command_components_dodder.Genesis
	repo.ImporterOptions
	command_components_dodder.Query

	complete command_components_dodder.Complete

	ExperimentalRepo  bool
	ParentPath        string
	Haustoria         string
	DefaultQueryGroup values.String
	Proto             sku.Proto
}

var _ interfaces.CommandComponentWriter = (*InitWorkspace)(nil)

func (cmd InitWorkspace) GetDescription() command.Description {
	return command.Description{
		Short: "initialize a workspace directory",
	}
}

func (cmd *InitWorkspace) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(
		&cmd.ExperimentalRepo,
		"experimental-repo",
		true,
		"create a repo-backed workspace with independent store and commit history",
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

	if !config.RepoId.IsEmpty() {
		req.Cancel(
			errors.BadRequestf(
				"-repo_id cannot be used with -experimental-repo (workspace repos are always CWD-rooted)",
			),
		)
		return
	}

	if err := config.RepoId.Set("."); err != nil {
		req.Cancel(err)
		return
	}

	absParentPath, parentIsHomeRepo := cmd.resolveParentPath(req)
	cmd.validateParentRepo(req, absParentPath, parentIsHomeRepo)

	cmd.Genesis.BigBang.ExcludeDefaultType = true
	cmd.linkParentZettelIdProviders(absParentPath, parentIsHomeRepo)

	local := cmd.OnTheFirstDay(req, req.PopArg("workspace repo id"))

	remote := cmd.makeParentRemote(req, local, absParentPath, parentIsHomeRepo)

	queryArgs := req.PopArgs()

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		local,
		queryArgs,
	)

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
		ParentPath: parentPathForConfig,
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

	absPath = filepath.Join(dataHome, env_dir.XDGUtilityNameDodder)
	return absPath, true
}

func (cmd InitWorkspace) validateParentRepo(
	req command.Request,
	absPath string,
	isHomeRepo bool,
) {
	var inventoryListLog string
	if isHomeRepo {
		inventoryListLog = filepath.Join(absPath, "inventory_lists_log")
	} else {
		inventoryListLog = filepath.Join(
			absPath,
			"."+env_dir.XDGUtilityNameDodder,
			"local", "share",
			"inventory_lists_log",
		)
	}

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

		envDir := env_dir.MakeWithHomeAndInitialize(
			req,
			env_dir.XDGUtilityNameDodder,
			home,
			config.Debug,
		)

		envUI := env_ui.Make(
			req,
			config,
			config.Debug,
			env_ui.Options{},
		)

		return local_working_copy.Make(
			env_local.Make(envUI, envDir),
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

func (cmd *InitWorkspace) linkParentZettelIdProviders(
	absParentPath string,
	isHomeRepo bool,
) {
	if cmd.Genesis.BigBang.Yin != "" || cmd.Genesis.BigBang.Yang != "" {
		return
	}

	var parentObjectIdDir string
	if isHomeRepo {
		parentObjectIdDir = filepath.Join(absParentPath, "object_ids")
	} else {
		parentObjectIdDir = filepath.Join(
			absParentPath,
			"."+env_dir.XDGUtilityNameDodder,
			"local", "share",
			"object_ids",
		)
	}

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
