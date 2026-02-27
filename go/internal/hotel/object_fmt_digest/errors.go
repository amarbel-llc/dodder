package object_fmt_digest

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var ErrEmptyTai = newPkgError("empty tai")

type errUnknownFormatKey string

func (err errUnknownFormatKey) Error() string {
	return fmt.Sprintf("unknown format key: %q", string(err))
}

func (err errUnknownFormatKey) Is(target error) bool {
	_, ok := target.(errUnknownFormatKey)
	return ok
}

func (err errUnknownFormatKey) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
