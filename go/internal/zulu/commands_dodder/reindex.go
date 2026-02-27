package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/whiskey/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("reindex", &Reindex{})
}

type Reindex struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd Reindex) Run(req command.Request) {
	args := req.PopArgs()

	if len(args) > 0 {
		errors.ContextCancelWithErrorf(
			req,
			"reindex does not support arguments",
		)
	}

	localWorkingCopy := cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsAllowConfigReadError,
	)

	localWorkingCopy.Reindex()
}
