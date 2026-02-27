package catgut

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

var ErrBufferEmpty = newPkgError("buffer empty")

type errLength struct {
	expected, actual int64
	err              error
}

func MakeErrLength(expected, actual int64, err error) error {
	switch {
	case expected != actual:
		return errLength{expected, actual, err}

	case err != nil:
		return errLength{expected, actual, err}

	default:
		return nil
	}
}

func (a errLength) Is(b error) (ok bool) {
	_, ok = b.(errLength)
	return ok
}

func (e errLength) Error() string {
	return fmt.Sprintf("expected %d but got %d. error: %s", e.expected, e.actual, e.err)
}

func (e errLength) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

type errInvalidSliceRange [2]int

func (a errInvalidSliceRange) Is(b error) (ok bool) {
	_, ok = b.(errInvalidSliceRange)
	return ok
}

func (e errInvalidSliceRange) Error() string {
	return fmt.Sprintf("invalid slice range: (%d, %d)", e[0], e[1])
}

func (e errInvalidSliceRange) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
