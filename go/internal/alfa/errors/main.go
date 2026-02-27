package errors

import (
	"fmt"
	"io"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/_/stack_frame"
)

func Join(es ...error) error {
	switch {
	case len(es) == 2 && es[0] == nil && es[1] == nil:
		return nil

	case len(es) == 2 && es[0] == nil:
		return es[1]

	case len(es) == 2 && es[1] == nil:
		return es[0]

	default:
		groupBuilder := MakeGroupBuilder(es...)
		return groupBuilder.GetError()
	}
}

func PanicIfError(err any) {
	if err == nil {
		return
	}

	switch errOrFunc := err.(type) {
	case func() error:
		PanicIfError(errOrFunc())

	case error:
		panic(WrapSkip(1, errOrFunc))
	}
}

//go:noinline
func WrapSkip(
	skip int,
	err error,
) error {
	if shouldNotWrap(err) {
		return err
	}

	var stackFrame stack_frame.Frame
	var ok bool

	if stackFrame, ok = stack_frame.MakeFrame(skip + 1); !ok {
		panic("failed to get stack info")
	}

	return stackFrame.Wrap(err)
}

const thisSkip = 1

//go:noinline
func Errorf(format string, args ...any) (err error) {
	err = fmt.Errorf(format, args...)
	return err
}

//go:noinline
func ErrorWithStackf(format string, args ...any) error {
	var stackFrame stack_frame.Frame
	var ok bool

	if stackFrame, ok = stack_frame.MakeFrame(thisSkip); !ok {
		panic("failed to get stack info")
	}

	return stackFrame.Errorf(format, args...)
}

func shouldNotWrap(err error) bool {
	switch err {
	case nil:
		return true

	case io.EOF:
		panic("trying to wrap io.EOF")
	}

	var errStackTracer stack_frame.ErrorStackTracer

	if As(err, &errStackTracer) && !errStackTracer.ShouldShowStackTrace() {
		return true
	} else {
		return false
	}
}

//go:noinline
func Wrap(err error) error {
	if shouldNotWrap(err) {
		return err
	}

	var stackFrame stack_frame.Frame
	var ok bool

	if stackFrame, ok = stack_frame.MakeFrame(thisSkip); !ok {
		panic("failed to get stack info")
	}

	if group, ok := err.(Group); ok && group.Len() == 1 {
		return stackFrame.Wrap(group[0])
	}

	return stackFrame.Wrap(err)
}

//go:noinline
func Wrapf(err error, format string, values ...any) error {
	if shouldNotWrap(err) {
		return err
	}

	var stackFrame stack_frame.Frame
	var ok bool

	if stackFrame, ok = stack_frame.MakeFrame(thisSkip); !ok {
		panic("failed to get stack info")
	}

	return stackFrame.Wrapf(err, format, values...)
}

type funcGetNext func() (error, funcGetNext)

// TODO remove / rewrite the below

// Wrap the error with stack info unless it's one of the provided `except`
// errors, in which case return nil. Direct value comparison is
// performed (`in == except`) rather than errors.Is.
//
//go:noinline
func WrapExceptSentinelAsNil(in error, except ...error) (err error) {
	if in == nil {
		return err
	}

	if slices.Contains(except, in) {
		return nil
	}

	err = WrapSkip(thisSkip, in)

	return err
}

// Wrap the error with stack info unless it's one of the provided `except`
// errors, in which case return that bare error. Direct value comparison is
// performed (`in == except`) rather than errors.Is.
//
//go:noinline
func WrapExceptSentinel(in error, except ...error) (err error) {
	if in == nil {
		return err
	}

	if slices.Contains(except, in) {
		return in
	}

	err = WrapSkip(thisSkip, in)

	return err
}
