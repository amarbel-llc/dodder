package ui

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/src/_/stack_frame"
)

// TODO rename to comment printer
var todo todoPrinter

func init() {
	todo = todoPrinter{
		printer: printer{
			writer: os.Stderr,
			file:   os.Stderr,
		},
		includesStack: true,
	}
}

func SetTodoOn() {
	todo.on = true
}

type todoPrinter devPrinter

//go:generate stringer -type=Priority
type Priority int

const (
	P0 = Priority(iota)
	P1
	P2
	P3
	P4
	P5
)

func Todo(f string, a ...any) (err error) {
	return printerErr.printfStack(1, "TODO: "+f, a...)
}

func TodoP0(f string, a ...any) (err error) {
	return todo.printf(1, P0, f, a...)
}

func TodoP1(f string, a ...any) (err error) {
	return todo.printf(1, P1, f, a...)
}

func TodoP2(f string, a ...any) (err error) {
	return todo.printf(1, P2, f, a...)
}

func TodoP3(f string, a ...any) (err error) {
	return todo.printf(1, P3, f, a...)
}

func TodoP4(f string, a ...any) (err error) {
	return todo.printf(1, P4, f, a...)
}

func TodoP5(f string, a ...any) (err error) {
	return todo.printf(1, P5, f, a...)
}

func (printer todoPrinter) Printf(
	priority Priority,
	format string,
	aargs ...any,
) (err error) {
	return printer.printf(1, priority, format, aargs...)
}

func (printer todoPrinter) printf(
	skip int,
	priority Priority,
	format string,
	args ...any,
) (err error) {
	if !printer.on {
		return err
	}

	if printer.includesStack {
		si, _ := stack_frame.MakeFrame(1 + skip)
		format = "%s %s" + format
		args = append([]any{priority, si}, args...)
	}

	_, err = fmt.Fprintln(
		printer.writer,
		fmt.Sprintf(format, args...),
	)

	return err
}
