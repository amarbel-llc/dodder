package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	mad_env_dir "code.linenisgreat.com/madder/go/pkgs/env_dir"
	env_local "code.linenisgreat.com/madder/go/pkgs/env_local"
	"code.linenisgreat.com/madder/go/pkgs/scoped_id"
)

// XDGUtilityNameMadder is the literal scope segment for madder's XDG
// namespace. Used as the second-env scope in env_repo.Make's two-env
// composition.
const XDGUtilityNameMadder = "madder"

// TODO move to command_components
type EnvRepo struct{}

func (cmd EnvRepo) MakeEnvRepo(
	req command.Request,
	permitNoDodderDirectory bool,
) env_repo.Env {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	if err := repo_id.CheckSupported(config.RepoId); err != nil {
		req.Cancel(err)
	}

	// Cwd, system, and explicit user ids force their scope via
	// MakeDefaultAndInitialize (it preserves the scoped_id LocationType);
	// only the auto/no-selector id falls through to MakeDefault so the
	// cwd-walk-up override still applies for it. An explicit XDG-user bare
	// name must NOT take that walk-up (FDR-0019: bare name is XDG-user scope
	// unconditionally), or an ancestor .dodder/ would hijack it to a
	// workspace-local repo. System never actually reaches here (CheckSupported
	// gates it above), but routing it to MakeDefaultAndInitialize keeps the
	// explicit 501 rather than silently resolving to the user tree if the
	// gate is ever relaxed.
	var ownDir, madderDir mad_env_dir.Env
	if loc := config.RepoId.GetLocationType(); loc == scoped_id.LocationTypeCwd ||
		loc == scoped_id.LocationTypeXDGSystem ||
		loc == scoped_id.LocationTypeXDGUser {
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

	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		env_ui.Options{},
	)

	var envRepo env_repo.Env

	envRepoOptions := env_repo.Options{
		BasePath:                config.BasePath,
		PermitNoDodderDirectory: permitNoDodderDirectory,
	}

	{
		var err error

		if envRepo, err = env_repo.Make(
			env_local.Make(envUI, ownDir),
			env_local.Make(envUI, madderDir),
			envRepoOptions,
		); err != nil {
			envUI.Cancel(err)
		}
	}

	return envRepo
}

// TODO move to command_components
