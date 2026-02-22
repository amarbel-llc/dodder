package fd

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

type Std struct {
	ui.Printer
}

func MakeStd(f *os.File) Std {
	return Std{
		Printer: ui.MakePrinter(f),
	}
}

func MakeStdFromWriter(w io.Writer) Std {
	return Std{
		Printer: ui.MakePrinterFromWriter(w),
	}
}
