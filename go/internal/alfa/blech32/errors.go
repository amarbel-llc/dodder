package blech32

import (
	"fmt"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
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
	ErrInvalidChecksum = newPkgError("invalid checksum")
	ErrMixedCase       = newPkgError("mixed case")
)

type errInvalidHRPCharacter struct {
	pos  int
	char rune
}

func (err errInvalidHRPCharacter) Error() string {
	return fmt.Sprintf(
		"invalid character in human-readable part: s[%d]=%d",
		err.pos,
		err.char,
	)
}

func (err errInvalidHRPCharacter) Is(target error) bool {
	_, ok := target.(errInvalidHRPCharacter)
	return ok
}

func (err errInvalidHRPCharacter) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

type errDataPortionTooShort struct {
	expected int
	actual   int
	data     string
}

func (err errDataPortionTooShort) Error() string {
	return fmt.Sprintf(
		"separator `-` at invalid position because data+checksum portion is too short. Should be at least %d but was %d (%q)",
		err.expected,
		err.actual,
		err.data,
	)
}

func (err errDataPortionTooShort) Is(target error) bool {
	_, ok := target.(errDataPortionTooShort)
	return ok
}

func (err errDataPortionTooShort) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

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
