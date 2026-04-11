package primordial

import (
	"os"

	"golang.org/x/term"
)

func IsTty(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
