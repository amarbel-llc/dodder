package key_bytes

import "github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var ErrInvalid = newPkgError("invalid key")
