//go:build !debug

package errors

import (
	"code.linenisgreat.com/dodder/go/lib/0/stack_frame"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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
