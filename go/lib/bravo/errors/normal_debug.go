//go:build debug

package errors

import (
	"code.linenisgreat.com/dodder/go/lib/0/stack_frame"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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
