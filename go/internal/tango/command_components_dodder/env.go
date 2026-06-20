package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
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
	// This env opens a repo (e.g. serve / serve-proto via
	// MakeLocalWorkingCopyFromEnvLocal), so its dodder metadata XDG must nest
	// under repos/<name>/ to find the repo init wrote (FDR-0019).
	// makeOperateEnvDir routes the cwd/system scopes: a system id roots at the
	// system root (#280), a multi-dot `..name` id resolves the Nth same-named
	// ancestor store-aware (#281), everything else keeps the nearest-ancestor
	// walk.
	layout := MakeOperateEnvDir(req, config, req.Utility.GetName())

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
		repo_id.EffectiveName(config.RepoId),
	)

	ui := env_ui.Make(
		req,
		config,
		config.Debug,
		options,
	)

	return env_local.Make(ui, dir)
}
