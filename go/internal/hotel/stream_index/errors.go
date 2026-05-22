package stream_index

import "github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"

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
