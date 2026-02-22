package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"code.linenisgreat.com/dodder/go/src/_/stack_frame"
)

type devPrinter struct {
	printer
	includesTime  bool
	includesStack bool
}

var _ DevPrinter = devPrinter{}

//go:noinline
func (printer devPrinter) PrintDebug(args ...any) (err error) {
	if !printer.on {
		return err
	}

	var sb strings.Builder

	if printer.includesTime {
		fmt.Fprintf(&sb, "%s ", time.Now())
	}

	if printer.includesStack {
		stackFrame, _ := stack_frame.MakeFrame(1)
		fmt.Fprintf(&sb, "%s ", stackFrame.StringNoFunctionName())
	}

	for range args {
		sb.WriteString("%#v ")
	}

	sb.WriteString("\n")

	_, err = fmt.Fprintf(
		printer.writer,
		sb.String(),
		args...,
	)

	return err
}

//go:noinline
func (printer devPrinter) Print(args ...any) (err error) {
	if !printer.on {
		return err
	}

	if printer.includesTime {
		args = append([]any{time.Now()}, args...)
	}

	if printer.includesStack {
		stackFrame, _ := stack_frame.MakeFrame(1)
		args = append([]any{stackFrame.StringNoFunctionName()}, args...)
	}

	return printer.printer.Print(args...)
}

//go:noinline
func (printer devPrinter) Printf(format string, args ...any) (err error) {
	if !printer.on {
		return err
	}

	if printer.includesTime {
		format = "%s " + format
		args = append([]any{time.Now()}, args...)
	}

	if printer.includesStack {
		stackFrame, _ := stack_frame.MakeFrame(1)
		format = "%s " + format
		args = append([]any{stackFrame.StringNoFunctionName()}, args...)
	}

	return printer.printer.Printf(format, args...)
}

//go:noinline
func (printer devPrinter) FunctionName(skip int) {
	if !printer.on {
		return
	}

	stackFrame, _ := stack_frame.MakeFrame(skip + 1)
	io.WriteString(
		printer.writer,
		fmt.Sprintf("%s %s\n", stackFrame, stackFrame.Function),
	)
}

//go:noinline
func (printer devPrinter) Stack(skip, count int) {
	if !printer.on {
		return
	}

	frames := stack_frame.MakeFrames(skip+1, count)

	io.WriteString(
		printer.writer,
		fmt.Sprintf(
			"Printing Stack (skip: %d, count requested: %d, count actual: %d):\n\n",
			skip,
			count,
			len(frames),
		),
	)

	for i, frame := range frames {
		io.WriteString(
			printer.writer,
			fmt.Sprintf("%s (%d)\n", frame.StringLogLine(), i),
		)
	}
}
