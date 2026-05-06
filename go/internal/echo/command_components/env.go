package command_components

import (
	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/env_local"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
)

type Env struct{}

func (cmd *Env) MakeEnv(req command.Request) env_local.Env {
	return cmd.MakeEnvWithOptions(
		req,
		env_ui.Options{},
	)
}

func (cmd *Env) MakeEnvWithOptions(
	req command.Request,
	options env_ui.Options,
) env_local.Env {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	layout := env_dir.MakeDefault(
		req,
		dodder_env.XDGUtilityName,
		config.Debug,
	)

	if options.CustomOut == nil && config.CustomOut != nil {
		options.CustomOut = config.CustomOut
	}

	if options.CustomErr == nil && config.CustomErr != nil {
		options.CustomErr = config.CustomErr
	}

	return env_local.Make(
		env_ui.Make(
			req,
			config,
			config.Debug,
			options,
		),
		layout,
	)
}

func (cmd *Env) MakeEnvWithXDGLayoutAndOptions(
	req command.Request,
	xdgDotenvPath string,
	options env_ui.Options,
) env_local.Env {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	dir := env_dir.MakeFromXDGDotenvPath(
		req,
		config.Debug,
		xdgDotenvPath,
	)

	if options.CustomOut == nil && config.CustomOut != nil {
		options.CustomOut = config.CustomOut
	}

	if options.CustomErr == nil && config.CustomErr != nil {
		options.CustomErr = config.CustomErr
	}

	ui := env_ui.Make(
		req,
		config,
		config.Debug,
		options,
	)

	return env_local.Make(ui, dir)
}
