package hyphence

import "github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var errMissingNewlineAfterBoundary = newPkgError(
	"missing blank line after closing boundary --- before blob body",
)
