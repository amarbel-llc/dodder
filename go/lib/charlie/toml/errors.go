package toml

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"github.com/pelletier/go-toml/v2"
)

type (
	StrictMissingError = toml.StrictMissingError

	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type Error struct {
	error
}

func MakeError(err error) Error {
	return Error{
		error: err,
	}
}

func (err Error) Is(target error) (ok bool) {
	_, ok = target.(Error)
	return ok
}

func (err Error) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
