package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/primordial"
	"code.linenisgreat.com/dodder/go/lib/_/stack_frame"
)

func MakePrinter(file *os.File) printer {
	return MakePrinterOn(file, true)
}

func MakePrinterOn(file *os.File, on bool) printer {
	return printer{
		writer: file,
		file:   file,
		isTty:  primordial.IsTty(file),
		on:     on,
	}
}

func MakePrinterFromWriter(w io.Writer) printer {
	return printer{
		writer: w,
		file:   nil,
		isTty:  false,
		on:     true,
	}
}

type printer struct {
	writer io.Writer
	file   *os.File
	isTty  bool
	on     bool
}

var _ Printer = printer{}

// Returns a copy of this printer with a modified `on` setting
func (printer printer) withOn(on bool) printer {
	printer.on = on
	return printer
}

func (printer printer) GetPrinter() Printer {
	return printer
}

func (printer printer) Write(b []byte) (n int, err error) {
	if !printer.on {
		n = len(b)
		return n, err
	}

	return printer.writer.Write(b)
}

func (printer printer) GetFile() *os.File {
	return printer.file
}

func (printer printer) IsTty() bool {
	return printer.isTty
}

//go:noinline
func (printer printer) Caller(skip int) Printer {
	if !printer.on {
		return Null
	}

	stackFrame, _ := stack_frame.MakeFrame(skip + 1)

	return prefixPrinter{
		Printer: printer,
		prefix:  stackFrame.StringNoFunctionName() + " ",
	}
}

func (printer printer) PrintDebug(args ...any) (err error) {
	if !printer.on {
		return err
	}

	_, err = fmt.Fprintf(
		printer.writer,
		strings.Repeat("%#v ", len(args))+"\n",
		args...,
	)

	return err
}

func (printer printer) Print(args ...any) (err error) {
	if !printer.on {
		return err
	}

	_, err = fmt.Fprintln(
		printer.writer,
		args...,
	)

	return err
}

//go:noinline
func (printer printer) printfStack(
	depth int,
	format string,
	args ...any,
) (err error) {
	if !printer.on {
		return err
	}

	stackFrame, _ := stack_frame.MakeFrame(1 + depth)
	format = "%s" + format
	args = append([]any{stackFrame}, args...)

	_, err = fmt.Fprintln(
		printer.writer,
		fmt.Sprintf(format, args...),
	)

	return err
}

func (printer printer) Printf(format string, args ...any) (err error) {
	if !printer.on {
		return err
	}

	_, err = fmt.Fprintln(
		printer.writer,
		fmt.Sprintf(format, args...),
	)

	return err
}

// Fatal(args ...any)
// Fatalf(format string, args ...any)
