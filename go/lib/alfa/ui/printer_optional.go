package ui

import "os"

type printerOptional struct {
	printer DevPrinter
	on      bool
}

var (
	_ Printer    = printerOptional{}
	_ DevPrinter = printerOptional{}
)

func (printer printerOptional) GetFile() *os.File {
	if printer.on {
		return printer.printer.GetFile()
	}

	return Null.GetFile()
}

func (printer printerOptional) IsTty() bool {
	if printer.on {
		return printer.printer.IsTty()
	}

	return false
}

func (printer printerOptional) Write(b []byte) (n int, err error) {
	if !printer.on {
		n = len(b)
		return n, err
	}

	return printer.printer.Write(b)
}

//go:noinline
func (printer printerOptional) PrintDebug(args ...any) error {
	if printer.on {
		return printer.printer.PrintDebug(args...)
	}

	return nil
}

//go:noinline
func (printer printerOptional) Print(args ...any) error {
	if printer.on {
		return printer.printer.Print(args...)
	}

	return nil
}

//go:noinline
func (printer printerOptional) Printf(format string, args ...any) error {
	if printer.on {
		return printer.printer.Printf(format, args...)
	}

	return nil
}

//go:noinline
func (printer printerOptional) Caller(skip int) Printer {
	if printer.on {
		return printer.printer.Caller(skip + 1)
	}

	return Null
}

//go:noinline
func (printer printerOptional) FunctionName(skip int) {
	if printer.on {
		printer.printer.FunctionName(skip + 1)
	}
}

//go:noinline
func (printer printerOptional) Stack(skip, count int) {
	if !printer.on {
		return
	}

	printer.printer.Stack(skip+1, count)
}
