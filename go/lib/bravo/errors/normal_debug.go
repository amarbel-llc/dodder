//go:build debug

package errors

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/0/stack_frame"
)

func PrintWithStackFramesIfNecessary(
	printer interfaces.Printer,
	message string,
	stackFrames []stack_frame.Frame,
) {
	if len(stackFrames) > 0 && debugBuild {
		printer.Printf("\n\n%s\n", stackFrames, message)
	} else {
		printer.Printf("\n\n%s", message)
	}
}
