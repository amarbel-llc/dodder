package key_bytes

import "code.linenisgreat.com/dodder/go/internal/alfa/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var ErrInvalid = newPkgError("invalid key")
