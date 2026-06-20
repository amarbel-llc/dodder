package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/command_components"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type LocalWorkingCopy struct {
	command_components.Env
}

var _ interfaces.CommandComponentWriter = (*LocalWorkingCopy)(nil)

func (cmd *LocalWorkingCopy) SetFlagDefinitions(
	f interfaces.CLIFlagDefinitions,
) {
}

func (cmd LocalWorkingCopy) MakeLocalWorkingCopy(
	req command.Request,
) *local_working_copy.Repo {
	return cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsEmpty,
	)
}

func (cmd LocalWorkingCopy) MakeLocalWorkingCopyWithOptions(
	req command.Request,
	envOptions env_ui.Options,
	repoOptions local_working_copy.Options,
) *local_working_copy.Repo {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	if err := repo_id.CheckSupported(config.RepoId); err != nil {
		req.Cancel(err)
	}

	// Like env_repo.go's MakeEnvRepo, a system-scoped id forces its scope
	// via MakeDefaultAndInitialize (which #280 wired to root //name at the
	// system root); EffectiveId preserves the location (system carries no
	// cwdDepth to drop). Everything else — the auto id, named user repos,
	// and single-dot cwd repos — keeps the MakeDefault cwd-walk-up path.
	var ownDir, madderDir mad_env_dir.Env

	if config.RepoId.GetLocationType() == scoped_id.LocationTypeXDGSystem {
		ownDir = env_dir.MakeDefaultAndInitialize(
			req,
			dodder_env.XDGUtilityName,
			config.Debug,
			repo_id.EffectiveId(config.RepoId),
		)

		madderDir = env_dir.MakeDefaultAndInitialize(
			req,
			XDGUtilityNameMadder,
			config.Debug,
			repo_id.EffectiveId(config.RepoId),
		)
	} else {
		ownDir = env_dir.MakeDefault(
			req,
			dodder_env.XDGUtilityName,
			config.Debug,
			repo_id.EffectiveName(config.RepoId),
		)

		madderDir = env_dir.MakeDefault(
			req,
			XDGUtilityNameMadder,
			config.Debug,
			repo_id.EffectiveName(config.RepoId),
		)
	}

	if envOptions.CustomOut == nil && config.CustomOut != nil {
		envOptions.CustomOut = config.CustomOut
	}

	if envOptions.CustomErr == nil && config.CustomErr != nil {
		envOptions.CustomErr = config.CustomErr
	}

	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		envOptions,
	)

	return local_working_copy.Make(
		env_local.Make(envUI, ownDir),
		env_local.Make(envUI, madderDir),
		repoOptions,
	)
}

// TODO modify to work with archives
func (cmd LocalWorkingCopy) MakeLocalWorkingCopyFromConfigAndXDGDotenvPath(
	req command.Request,
	xdgDotenvPath string,
	options env_ui.Options,
) (local *local_working_copy.Repo) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	ownDir := env_dir.MakeFromXDGDotenvPath(
		req,
		config.Debug,
		xdgDotenvPath,
		repo_id.EffectiveName(config.RepoId),
	)

	// FDR-0019: the blob-store slot must be a flat madder-scoped env --
	// madder blob pools are separate and never nest. Re-key the dotenv's
	// own XDG to the madder utility (CloneWithUtilityName re-derives the
	// category dirs from the dotenv base, dropping the repos/<name>/
	// nesting); configFor blanks RepoName for the madder utility. Do NOT
	// use ownDir.GetXDGForBlobStores() here: it keeps the dodder utility
	// name and the repos/<name>/ nesting, which would hide blobs from the
	// flat pool.
	madderDir := env_dir.MakeWithXDG(
		req,
		config.Debug,
		ownDir.GetXDG().CloneWithUtilityName(XDGUtilityNameMadder),
		"",
	)

	if options.CustomOut == nil && config.CustomOut != nil {
		options.CustomOut = config.CustomOut
	}

	if options.CustomErr == nil && config.CustomErr != nil {
		options.CustomErr = config.CustomErr
	}

	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		options,
	)

	local = local_working_copy.Make(
		env_local.Make(envUI, ownDir),
		env_local.Make(envUI, madderDir),
		local_working_copy.OptionsEmpty,
	)

	return local
}

// MakeLocalWorkingCopyFromEnvLocal serves callers that already hold an
// env_local.Env for the own (dodder) scope — serve, serve-proto, and the
// HTTP/websocket remote server. envLocal's metadata XDG nests under
// repos/<name>/ (FDR-0019), so it is correct for the own slot but wrong
// for the blob-store slot: madder blob pools are separate, content-
// addressed, and never nest (see XDGUtilityNameMadder / configFor). The
// second slot is therefore built the same way the two-env builders do
// (env_dir.MakeDefault blanks RepoName for the madder utility and
// resolves the blob-store base via the madder XDG override / cwd walk),
// so opening a repo here lands on the same flat blob pool that init wrote
// and that ordinary commands read.
func (cmd LocalWorkingCopy) MakeLocalWorkingCopyFromEnvLocal(
	req command.Request,
	envLocal env_local.Env,
) (local *local_working_copy.Repo) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	// Gate unwired scopes (multi-dot cwd depth, system) here too, so
	// serve / serve-proto / the remote server reject them with the same
	// clear error as the two-env builders instead of silently
	// mis-resolving to a user/cwd repo (#273). Keeps CheckSupported the
	// uniform FDR-0019 gate so the P2 pickup is a single-place relaxation.
	if err := repo_id.CheckSupported(config.RepoId); err != nil {
		req.Cancel(err)
	}

	madderDir := env_dir.MakeDefault(
		req,
		XDGUtilityNameMadder,
		config.Debug,
		repo_id.EffectiveName(config.RepoId),
	)

	local = local_working_copy.Make(
		envLocal,
		env_local.Make(envLocal, madderDir),
		local_working_copy.OptionsEmpty,
	)

	return local
}
