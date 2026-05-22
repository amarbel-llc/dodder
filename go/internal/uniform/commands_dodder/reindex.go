package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	utility.AddCmd("reindex", &Reindex{})
}

func (cmd Reindex) GetDescription() command.Description {
	return command.Description{
		Short: "rebuild store indices",
	}
}

type Reindex struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*Reindex)(nil)

// GetArgs returns nil: reindex explicitly rejects positional arguments.
func (cmd *Reindex) GetArgs() []command.ArgGroup { return nil }

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
