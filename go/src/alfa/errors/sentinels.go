package errors

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

// Stop iteration sentinel
type stopIterationDisamb struct{}

var errStopIteration = NewWithType[stopIterationDisamb]("stop iteration")

func MakeErrStopIteration() error {
	return errStopIteration
}

func IsStopIteration(err error) bool {
	return IsTyped[stopIterationDisamb](err)
}

type (
	Typed[DISAMB any] interface {
		error
		GetErrorType() DISAMB
	}

	errorString[DISAMB any] struct {
		value string
	}

	errorTypedWrapped[DISAMB any] struct {
		wrapped error
	}
)

func IsTyped[DISAMB any](err error) bool {
	var typed Typed[DISAMB]
	if As(err, &typed) {
		return true
	}
	return false
}

// MakeTypedSentinel creates a typed sentinel error and its checker function.
// This is a convenience helper to reduce boilerplate when creating package errors.
//
// Usage:
//
//	type pkgErrDisamb struct{}
//	var (
//	    ErrMyError, IsMyError = errors.MakeTypedSentinel[pkgErrDisamb]("my error")
//	)
//
// The returned sentinel implements errors.Typed[DISAMB] and can be checked with
// either the returned checker function or errors.IsTyped[DISAMB](err).
func MakeTypedSentinel[DISAMB any](text string) (
	sentinel Typed[DISAMB],
	check func(error) bool,
) {
	sentinel = NewWithType[DISAMB](text)
	check = func(err error) bool {
		return IsTyped[DISAMB](err)
	}
	return sentinel, check
}

func NewWithType[DISAMB any](text string) Typed[DISAMB] {
	return &errorString[DISAMB]{text}
}

func WrapWithType[DISAMB any](err error) Typed[DISAMB] {
	return &errorTypedWrapped[DISAMB]{wrapped: err}
}

func (err *errorTypedWrapped[TYPE]) Error() string {
	return err.wrapped.Error()
}

func (err *errorTypedWrapped[TYPE]) GetErrorType() TYPE {
	var disamb TYPE
	return disamb
}

func (err *errorTypedWrapped[_]) Unwrap() error {
	return err.wrapped
}

func (err *errorString[_]) Error() string {
	return err.value
}

func (err *errorString[TYPE]) GetErrorType() TYPE {
	var disamb TYPE
	return disamb
}

func (err *errorString[DISAMB]) Is(target error) bool {
	_, ok := target.(*errorString[DISAMB])
	return ok
}

// Exists sentinel
type errExistsDisamb struct{}

var ErrExists = NewWithType[errExistsDisamb]("exists")

// Not found sentinel
type errNotFoundDisamb struct{}

type ErrNotFound struct {
	Value string
}

func (err ErrNotFound) Error() string {
	if err.Value == "" {
		return "not found"
	}
	return fmt.Sprintf("not found: %q", err.Value)
}

func (err ErrNotFound) Is(target error) (ok bool) {
	_, ok = target.(ErrNotFound)
	return ok
}

func (err ErrNotFound) GetErrorType() errNotFoundDisamb {
	return errNotFoundDisamb{}
}

func MakeErrNotFound(value interfaces.Stringer) error {
	return ErrNotFound{Value: value.String()}
}

func MakeErrNotFoundString(s string) error {
	return ErrNotFound{Value: s}
}

func IsErrNotFound(err error) bool {
	return IsTyped[errNotFoundDisamb](err)
}

// GetErrNotFound extracts the ErrNotFound from an error chain.
// Returns the typed error and true if found, zero value and false otherwise.
func GetErrNotFound(err error) (ErrNotFound, bool) {
	var notFoundErr ErrNotFound
	if As(err, &notFoundErr) {
		return notFoundErr, true
	}
	return ErrNotFound{}, false
}
