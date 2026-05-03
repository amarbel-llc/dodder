package ui

import "fmt"

func MakePrefixPrinter(printer Printer, prefix string) prefixPrinter {
	return prefixPrinter{
		Printer: printer,
		prefix:  prefix,
	}
}

type prefixPrinter struct {
	Printer
	prefix string
}

func (printer prefixPrinter) Print(args ...any) error {
	return printer.Printer.Print(
		append([]any{printer.prefix}, args...)...,
	)
}

func (printer prefixPrinter) Printf(format string, args ...any) error {
	return printer.Printer.Printf(
		fmt.Sprintf("%s%s", printer.prefix, format),
		args...,
	)
}
