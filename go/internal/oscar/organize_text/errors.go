package organize_text

import "code.linenisgreat.com/dodder/go/internal/alfa/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrorRead struct {
	error

	line, column int
}

func (err ErrorRead) Is(target error) bool {
	_, ok := target.(ErrorRead)
	return ok
}

func (err ErrorRead) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
