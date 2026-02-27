package ids

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

func wrapAsPkgError(err error) pkgError {
	return errors.WrapWithType[pkgErrDisamb](err)
}

type ErrInvalidId string

func (e ErrInvalidId) Error() string {
	return fmt.Sprintf("invalid object id: %q", string(e))
}

func (e ErrInvalidId) Is(err error) (ok bool) {
	_, ok = err.(ErrInvalidId)
	return ok
}

func (e ErrInvalidId) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrInvalid(err error) bool {
	return errors.Is(err, ErrInvalidId(""))
}

type errInvalidSigil string

func (e errInvalidSigil) Error() string {
	return fmt.Sprintf("invalid sigil: %q", string(e))
}

func (e errInvalidSigil) Is(err error) (ok bool) {
	_, ok = err.(errInvalidSigil)
	return ok
}

func (e errInvalidSigil) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrInvalidSigil(err error) bool {
	return errors.Is(err, errInvalidSigil(""))
}

var ErrEmptyTag = newPkgError("empty tag")
