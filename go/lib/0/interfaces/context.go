package interfaces

import (
	"context"
)

//go:generate stringer -type=ContextState
type ContextState uint8

const (
	ContextStateUnknown = ContextState(iota)
	ContextStateUnstarted
	// all states that are > than `ContextStateStarted` are considered terminal,
	// so the order here is important
	ContextStateStarted
	ContextStateSucceeded
	ContextStateFailed
	ContextStateAborted
)

func (state ContextState) IsComplete() bool {
	return state > ContextStateStarted
}

func (state ContextState) Error() string {
	return state.String()
}

func (state ContextState) Is(target error) bool {
	_, ok := target.(ContextState)
	return ok
}

type (
	FuncActiveContext = func(ActiveContext) error

	// TODO think about how to separate "consumers" of context, and "managers or
	// supervisors"

	ActiveContext interface {
		context.Context

		Cause() error
		GetState() ContextState

		// TODO disambiguate between errors and exceptions
		// TODO rename this to Complete
		Cancel(error)

		// `After` runs a function after the context is complete (regardless of
		// any errors). `After`s are run in the reverse order of when they are
		// called,
		// like
		// defers but on a whole-program level.
		After(FuncActiveContext)

		// `Must` executes a function even if the context has been cancelled. If
		// the function returns an error, `Must` cancels the context. It is
		// meant for
		// defers that must be executed, like closing files, flushing buffers,
		// releasing locks.
		Must(FuncActiveContext)
	}

	ActiveContextGetter interface {
		GetActiveContext() ActiveContext
	}

	FuncRetry        func()
	FuncRetryAborted func(err error)

	ErrorRetryable interface {
		error
		Recover(ActiveContext, FuncRetry, FuncRetryAborted)
	}
)
