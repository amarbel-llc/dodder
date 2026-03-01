package stream_index_fixed

import "code.linenisgreat.com/dodder/go/lib/bravo/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var (
	errConcurrentPageAccess = newPkgError("concurrent page access")
	errExpectedSigil        = newPkgError("expected sigil")
	errOverflowTooLarge     = newPkgError("overflow content exceeds uint16 max")
)
