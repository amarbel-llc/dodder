package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/env_local"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/command_components"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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

	ownDir := env_dir.MakeDefault(
		req,
		env_dir.XDGUtilityNameDodder,
		config.Debug,
	)

	madderDir := env_dir.MakeDefault(
		req,
		XDGUtilityNameMadder,
		config.Debug,
	)

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
	)

	madderDir := env_dir.MakeWithXDG(
		req,
		config.Debug,
		ownDir.GetXDGForBlobStores(),
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

// MakeLocalWorkingCopyFromEnvLocal preserves the legacy single-env
// shape for callers that already hold an env_local.Env they want to
// reuse. The same env is passed to both Make slots; for own scope
// it's correct, and for madder scope it relies on the existing
// dodder env_dir.GetXDGForBlobStores bridge mapping blob-store XDG
// to "madder". Once the dodder env_dir fork is dropped, this helper
// must be deleted (#151 bucket B Stage B follow-up): there will no
// longer be a bridge, and the second env must be a properly madder-
// scoped env_local.
func (cmd LocalWorkingCopy) MakeLocalWorkingCopyFromEnvLocal(
	envLocal env_local.Env,
) (local *local_working_copy.Repo) {
	local = local_working_copy.Make(
		envLocal,
		envLocal,
		local_working_copy.OptionsEmpty,
	)

	return local
}
