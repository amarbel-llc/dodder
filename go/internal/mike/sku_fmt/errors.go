package sku_fmt

import "code.linenisgreat.com/dodder/go/lib/alfa/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var errEmptySku = newPkgError("empty sku")
