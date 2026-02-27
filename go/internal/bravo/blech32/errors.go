package blech32

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var (
	ErrEmptyHRP         = newPkgError("empty HRP")
	ErrSeparatorMissing = newPkgError(
		fmt.Sprintf("separator (%q) missing", string(separator)),
	)
)

type errInvalidCharacterInData struct {
	pos  int
	char rune
}

func (err errInvalidCharacterInData) Error() string {
	return fmt.Sprintf(
		"invalid character %q found at position %d. expected one of %q",
		string([]rune{err.char}),
		err.pos,
		charsetString,
	)
}

func (err errInvalidCharacterInData) Is(target error) bool {
	_, ok := target.(errInvalidCharacterInData)
	return ok
}

func (err errInvalidCharacterInData) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
