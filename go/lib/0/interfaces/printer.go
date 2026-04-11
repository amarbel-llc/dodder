package interfaces

import (
	"io"
	"os"
)

type Printer interface {
	io.Writer

	GetFile() *os.File
	IsTty() bool

	// TODO add "isOn" function
	Caller(skip int) Printer
	Print(a ...any) (err error)
	PrintDebug(a ...any) (err error)
	Printf(f string, a ...any) (err error)

	// TODO add below
	// Fatal(args ...any)
	// Fatalf(format string, args ...any)
}
