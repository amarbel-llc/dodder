package env_repo

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrNotInDodderDir struct {
	Expected string
}

func (err ErrNotInDodderDir) Error() string {
	if err.Expected == "" {
		return "not in a dodder directory."
	} else {
		return fmt.Sprintf("not in a dodder directory. Looking for %s", err.Expected)
	}
}

func (err ErrNotInDodderDir) ShouldShowStackTrace() bool {
	return false
}

func (err ErrNotInDodderDir) Is(target error) (ok bool) {
	_, ok = target.(ErrNotInDodderDir)
	return ok
}

func (err ErrNotInDodderDir) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
