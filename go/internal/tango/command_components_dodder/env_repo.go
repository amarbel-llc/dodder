package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
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

	var ownDir, madderDir mad_env_dir.Env
	if config.RepoId.IsCwd() || config.RepoId.IsSystem() {
		ownDir = env_dir.MakeDefaultAndInitialize(
			req,
			dodder_env.XDGUtilityName,
			config.Debug,
			config.RepoId,
		)
		madderDir = env_dir.MakeDefaultAndInitialize(
			req,
			XDGUtilityNameMadder,
			config.Debug,
			config.RepoId,
		)
	} else {
		ownDir = env_dir.MakeDefault(
			req,
			dodder_env.XDGUtilityName,
			config.Debug,
		)
		madderDir = env_dir.MakeDefault(
			req,
			XDGUtilityNameMadder,
			config.Debug,
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
