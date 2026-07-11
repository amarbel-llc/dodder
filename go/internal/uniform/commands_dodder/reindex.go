package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
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

	AllowLockFailures bool
}

var (
	_ command.CommandWithArgs           = (*Reindex)(nil)
	_ interfaces.CommandComponentWriter = (*Reindex)(nil)
)

// GetArgs returns nil: reindex explicitly rejects positional arguments.
func (cmd *Reindex) GetArgs() []command.ArgGroup { return nil }

func (cmd *Reindex) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

	flagSet.BoolVar(
		&cmd.AllowLockFailures,
		"allow_lock_failures",
		false,
		"tolerate objects whose type/tag/referenced-object/blob-reference lock target no longer exists (e.g. a type object lost to a prior import/export bug) instead of reporting them as reindex errors",
	)
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

	localWorkingCopy.Reindex(cmd.AllowLockFailures)
}
