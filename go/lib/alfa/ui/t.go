//go:build test

package ui

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/0/test_ui"
)

type T = test_ui.T

func MakeT(t *testing.T) T {
	return T{
		T:            t,
		Printer:      Err(),
		ErrorEncoder: CLIErrorTreeEncoder,
	}
}
