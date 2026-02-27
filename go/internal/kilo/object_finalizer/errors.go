package object_finalizer

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

func IsTypeLockError(err error) bool {
	return errors.IsTyped[pkgErrDisamb](err)
}

var (
	ErrFailedToReadCurrentLockObject = newPkgError("failed to read current lock object")
	ErrEmptyLockKey                  = newPkgError("empty type")
	ErrBuiltinType                   = newPkgError("builtin type")
)
