package stream_index

import "code.linenisgreat.com/dodder/go/lib/bravo/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var errConcurrentPageAccess = newPkgError("concurrent page access")

func MakeErrConcurrentPageAccess() error {
	return errors.WrapSkip(2, errConcurrentPageAccess)
}
