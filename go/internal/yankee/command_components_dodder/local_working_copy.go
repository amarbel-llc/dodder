package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_local"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/lima/command_components"
	"code.linenisgreat.com/dodder/go/internal/whiskey/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
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
	env := cmd.MakeEnvWithOptions(req, envOptions)

	return local_working_copy.Make(env, repoOptions)
}

// TODO modify to work with archives
func (cmd LocalWorkingCopy) MakeLocalWorkingCopyFromConfigAndXDGDotenvPath(
	req command.Request,
	xdgDotenvPath string,
	options env_ui.Options,
) (local *local_working_copy.Repo) {
	envLocal := cmd.MakeEnvWithXDGLayoutAndOptions(
		req,
		xdgDotenvPath,
		options,
	)

	local = local_working_copy.Make(
		envLocal,
		local_working_copy.OptionsEmpty,
	)

	return local
}

func (cmd LocalWorkingCopy) MakeLocalWorkingCopyFromEnvLocal(
	envLocal env_local.Env,
) (local *local_working_copy.Repo) {
	local = local_working_copy.Make(
		envLocal,
		local_working_copy.OptionsEmpty,
	)

	return local
}
