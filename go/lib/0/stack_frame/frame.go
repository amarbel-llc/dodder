package stack_frame

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	cwd          string
	maxCallDepth int
)

func init() {
	var err error

	if cwd, err = os.Getwd(); err != nil {
		log.Panic(err)
	}
}

type Frame struct {
	Package     string
	Function    string
	Filename    string
	RelFilename string
	Line        int

	Prefix string

	nonZero bool
}

func MakeFrameFromRuntimeFrame(runtimeFrame runtime.Frame) (frame Frame) {
	frame.Filename = filepath.Clean(runtimeFrame.File)
	frame.Line = runtimeFrame.Line
	frame.Function = runtimeFrame.Function
	frame.Package, frame.Function = getPackageAndFunctionName(frame.Function)

	frame.RelFilename, _ = filepath.Rel(cwd, frame.Filename)
	frame.nonZero = true

	return frame
}

//go:noinline
func MustFrame(skip int) Frame {
	frame, ok := MakeFrame(skip + 1)

	if !ok {
		panic("stack unavailable")
	}

	return frame
}

//go:noinline
func MakeFrames(skip, count int) (frames []Frame) {
	programCounters := make([]uintptr, count)
	writtenCounters := runtime.Callers(skip+1, programCounters) // 0 is self

	if writtenCounters == 0 {
		return frames
	}

	programCounters = programCounters[:writtenCounters]

	rawFrames := runtime.CallersFrames(programCounters)

	frames = make([]Frame, 0, len(programCounters))

	for {
		frame, more := rawFrames.Next()
		frames = append(frames, MakeFrameFromRuntimeFrame(frame))

		if !more {
			break
		}
	}

	return frames
}

//go:noinline
func MakeFrame(skip int) (frame Frame, ok bool) {
	var programCounter uintptr
	programCounter, _, _, ok = runtime.Caller(skip + 1) // 0 is self

	if !ok {
		return frame, ok
	}

	runtimeFrames := runtime.CallersFrames([]uintptr{programCounter})

	runtimeFrame, _ := runtimeFrames.Next()
	frame = MakeFrameFromRuntimeFrame(runtimeFrame)

	// TODO remove this ugly hack
	if frame.Function == "Wrap" {
		panic(fmt.Sprintf("Parent Wrap included in stack. Skip: %d", skip))
	}

	return frame, ok
}

func (frame Frame) IsEmpty() bool {
	return !frame.nonZero
}

func getPackageAndFunctionName(v string) (p string, f string) {
	p, f = filepath.Split(v)

	idx := strings.Index(f, ".")

	if idx == -1 {
		return p, f
	}

	p += f[:idx]

	if len(f) > idx+1 {
		f = f[idx+1:]
	}

	return p, f
}

func (frame Frame) FileNameLine() string {
	filename := frame.Filename

	if frame.RelFilename != "" {
		filename = frame.RelFilename
	}

	return fmt.Sprintf(
		"%s:%d",
		filename,
		frame.Line,
	)
}

func (frame Frame) String() string {
	filename := frame.Filename

	if frame.RelFilename != "" {
		filename = frame.RelFilename
	}

	// TODO-P3 determine if si.line is ever not valid
	return fmt.Sprintf(
		"# %s\n%s%s:%d",
		frame.Function,
		frame.Prefix,
		filename,
		frame.Line,
	)
}

func (frame Frame) StringLogLine() string {
	filename := frame.Filename

	if frame.RelFilename != "" {
		filename = frame.RelFilename
	}

	// TODO-P3 determine if si.line is ever not valid
	return fmt.Sprintf(
		"%s%s:%d: %s",
		frame.Prefix,
		filename,
		frame.Line,
		frame.Function,
	)
}

func (frame Frame) StringNoFunctionName() string {
	if frame.IsEmpty() {
		return "|0| # empty stack frame"
	}

	filename := frame.Filename

	if frame.RelFilename != "" {
		filename = frame.RelFilename
	}

	return fmt.Sprintf(
		"%s|%d|",
		filename,
		frame.Line,
	)
}

//  __        __                     _
//  \ \      / / __ __ _ _ __  _ __ (_)_ __   __ _
//   \ \ /\ / / '__/ _` | '_ \| '_ \| | '_ \ / _` |
//    \ V  V /| | | (_| | |_) | |_) | | | | | (_| |
//     \_/\_/ |_|  \__,_| .__/| .__/|_|_| |_|\__, |
//                      |_|   |_|            |___/

// If the frame is non-zero, return a wrapped error. Otherwise return the input
// error unwrapped.
func (frame Frame) Wrap(err error) error {
	if err == nil {
		return nil
	}

	if !frame.nonZero {
		return err
	}

	if existing, ok := err.(*ErrorTree); ok {
		existing.Append(nil, frame)
		return existing
	} else {
		tree := &ErrorTree{
			Root: err,
		}

		tree.Append(nil, frame)

		return tree
	}
}

func (frame Frame) Wrapf(err error, format string, args ...any) error {
	extra := fmt.Errorf(
		"%s: "+format,
		append([]any{err.Error()}, args...)...,
	)

	if !frame.nonZero {
		return extra
	}

	if existing, ok := err.(*ErrorTree); ok {
		existing.Append(extra, frame)
		return existing
	} else {
		tree := &ErrorTree{
			Root: err,
		}

		tree.Append(extra, frame)

		return tree
	}
}

func (frame Frame) Errorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)

	if !frame.nonZero {
		return err
	}

	tree := &ErrorTree{
		Root: err,
	}

	tree.Append(nil, frame)

	return tree
}
