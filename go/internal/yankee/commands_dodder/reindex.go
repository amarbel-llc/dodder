package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
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
