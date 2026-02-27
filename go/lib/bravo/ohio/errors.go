package ohio

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var (
	ErrBoundaryNotFound      = newPkgError("boundary not found")
	ErrExpectedContentRead   = newPkgError("expected content read")
	ErrExpectedBoundaryRead  = newPkgError("expected boundary read")
	ErrReadFromSmallOverflow = newPkgError(
		"reader provided more bytes than max int",
	)
	ErrInvalidBoundaryReaderState = newPkgError("invalid boundary reader state")
)

type ErrExhaustedFuncSetStringersLine struct {
	error
	string
}

func (e ErrExhaustedFuncSetStringersLine) Error() string {
	return fmt.Sprintf("exhausted FuncSetString'ers at segment: %q", e.string)
}

func (e ErrExhaustedFuncSetStringersLine) Is(target error) (ok bool) {
	_, ok = target.(ErrExhaustedFuncSetStringersLine)
	return ok
}

func (e ErrExhaustedFuncSetStringersLine) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrExhaustedFuncSetStringers(err error) bool {
	return errors.Is(err, ErrExhaustedFuncSetStringersLine{})
}

func ErrExhaustedFuncSetStringersGetString(in error) (ok bool, v string) {
	var err ErrExhaustedFuncSetStringersLine

	if errors.As(in, &err) {
		v = err.string
	}

	return ok, v
}

func ErrExhaustedFuncSetStringersWithDelim(in error, delim byte) (out error) {
	var err ErrExhaustedFuncSetStringersLine

	if errors.As(in, &err) {
		err.string = fmt.Sprintf("%s%s", err.string, string([]byte{delim}))
		in = err
	}

	out = in

	return out
}
