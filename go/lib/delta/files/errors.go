package files

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var (
	ErrEmptyFileList = newPkgError("empty file list")
	errNotDirectory  ErrNotDirectory
)

func IsErrNotDirectory(err error) bool {
	return errors.Is(errNotDirectory, err)
}

type ErrNotDirectory string

func (err ErrNotDirectory) Is(target error) bool {
	_, ok := target.(ErrNotDirectory)
	return ok
}

func (err ErrNotDirectory) Error() string {
	return fmt.Sprintf("%q is not a directory", string(err))
}

func (err ErrNotDirectory) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
