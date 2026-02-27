//go:build !debug

package errors

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/_/stack_frame"
)

func PrintStackTracerIfNecessary(
	printer interfaces.Printer,
	name string,
	err error,
	_ ...any,
) {
	var normalError stack_frame.ErrorStackTracer

	if As(err, &normalError) && !normalError.ShouldShowStackTrace() {
		printer.Printf(
			"\n\n%s failed with error:\n%s",
			name,
			normalError.Error(),
		)
	} else {
		printer.Printf("\n\n%s failed with error:\n%s", name, err)
	}
}
