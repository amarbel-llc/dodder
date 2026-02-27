package command

import (
	"fmt"
	"syscall"

	"code.linenisgreat.com/dodder/go/internal/_/stack_frame"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
)

func extendNameIfNecessary(name string) string {
	if name == "dodder" {
		return name
	} else {
		return fmt.Sprintf("dodder (%s)", name)
	}
}

func handleMainErrors(
	ctx errors.Context,
	name string,
	err error,
) (exitStatus int) {
	exitStatus = 1

	var signal errors.Signal

	if errors.As(err, &signal) {
		if signal.Signal != syscall.SIGHUP {
			ui.Err().Printf(
				"%s aborting due to signal: %s",
				name,
				signal.Signal,
			)
		}

		return exitStatus
	}

	var helpful errors.Helpful

	if errors.As(err, &helpful) {
		errors.PrintHelpful(ui.Err(), helpful)
		return exitStatus
	}

	if errors.Is499ClientClosedRequest(err) {
		return exitStatus
	}

	_, frames := ctx.CauseWithStackFrames()
	err = stack_frame.MakeErrorTreeOrErr(err, frames...)

	ui.CLIErrorTreeEncoder.EncodeTo(err, ui.Err())

	return exitStatus
}
