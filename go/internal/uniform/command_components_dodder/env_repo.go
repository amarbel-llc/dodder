package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/echo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
)

// TODO move to command_components
type EnvRepo struct{}

func (cmd EnvRepo) MakeEnvRepo(
	req command.Request,
	permitNoDodderDirectory bool,
) env_repo.Env {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	dir := env_dir.MakeDefault(
		req,
		env_dir.XDGUtilityNameDodder,
		config.Debug,
	)

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
			env_local.Make(envUI, dir),
			envRepoOptions,
		); err != nil {
			envUI.Cancel(err)
		}
	}

	return envRepo
}

func (cmd EnvRepo) MakeEnvRepoFromEnvLocal(
	envLocal env_local.Env,
) env_repo.Env {
	var envRepo env_repo.Env

	var basePath string
	if repoConfig, ok := envLocal.GetCLIConfig().(domain_interfaces.RepoCLIConfigProvider); ok {
		basePath = repoConfig.GetBasePath()
	}

	layoutOptions := env_repo.Options{
		BasePath: basePath,
	}

	{
		var err error

		if envRepo, err = env_repo.Make(
			envLocal,
			layoutOptions,
		); err != nil {
			envLocal.Cancel(err)
		}
	}

	return envRepo
}

// TODO move to command_components
