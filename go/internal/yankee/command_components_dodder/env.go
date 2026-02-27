package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_local"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
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
		req.Utility.GetName(),
		config.Debug,
	)

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

	ui := env_ui.Make(
		req,
		config,
		config.Debug,
		options,
	)

	return env_local.Make(ui, dir)
}
