package sku_fmt

import "code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var errEmptySku = newPkgError("empty sku")
