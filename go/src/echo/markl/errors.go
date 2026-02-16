package markl

import (
	"bytes"
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"golang.org/x/exp/constraints"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var ErrEmptyType = newPkgError("type is empty")

func MakeErrEmptyType(id domain_interfaces.MarklId) error {
	if id.GetMarklFormat() == nil {
		return errors.WrapSkip(1, ErrEmptyType)
	}

	return nil
}

func AssertIdIsNull(id domain_interfaces.MarklId) error {
	if !id.IsNull() {
		cloned, _ := Clone(id)
		return errors.WrapSkip(1, errIsNotNull{id: cloned})
	}

	return nil
}

type errIsNotNull struct {
	id domain_interfaces.MarklId
}

func (err errIsNotNull) Error() string {
	return fmt.Sprintf("blob id is not null %q", err.id)
}

func (err errIsNotNull) Is(target error) bool {
	_, ok := target.(errIsNotNull)
	return ok
}

func (err errIsNotNull) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func AssertIdIsNotNull(id domain_interfaces.MarklId) error {
	if id.IsNull() {
		return errors.WrapSkip(1, errIsNull{purpose: id.GetPurposeId()})
	}

	return nil
}

func AssertIdIsNotNullWithPurpose(id domain_interfaces.MarklId, purpose string) error {
	if id.IsNull() {
		return errors.WrapSkip(1, errIsNull{purpose: purpose})
	}

	return nil
}

func IsErrNull(target error) bool {
	return errors.Is(target, errIsNull{})
}

type errIsNull struct {
	purpose string
}

func (err errIsNull) Error() string {
	return fmt.Sprintf("markl id is null for purpose %q", err.purpose)
}

func (err errIsNull) Is(target error) bool {
	_, ok := target.(errIsNull)
	return ok
}

func (err errIsNull) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

type ErrNotEqual struct {
	Expected, Actual domain_interfaces.MarklId
}

func AssertEqual(expected, actual domain_interfaces.MarklId) (err error) {
	if Equals(expected, actual) {
		return err
	}

	err = ErrNotEqual{
		Expected: expected,
		Actual:   actual,
	}

	return err
}

func (err ErrNotEqual) Error() string {
	return fmt.Sprintf(
		"expected digest %q but got %q",
		err.Expected,
		err.Actual,
	)
}

func (err ErrNotEqual) Is(target error) bool {
	_, ok := target.(ErrNotEqual)
	return ok
}

func (err ErrNotEqual) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrNotEqual) IsDifferentHashTypes() bool {
	return err.Expected.GetMarklFormat() != err.Actual.GetMarklFormat()
}

type ErrNotEqualBytes struct {
	Expected, Actual []byte
}

func MakeErrNotEqualBytes(expected, actual []byte) (err error) {
	if bytes.Equal(expected, actual) {
		return err
	}

	err = ErrNotEqualBytes{
		Expected: expected,
		Actual:   actual,
	}

	return err
}

func (err ErrNotEqualBytes) Error() string {
	return fmt.Sprintf(
		"expected digest %x but got %x",
		err.Expected,
		err.Actual,
	)
}

func (err ErrNotEqualBytes) Is(target error) bool {
	_, ok := target.(ErrNotEqualBytes)
	return ok
}

func (err ErrNotEqualBytes) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

type errLength[INTEGER constraints.Integer] [2]INTEGER

// TODO add another "wrong hasher" error type
func MakeErrLength[INTEGER constraints.Integer](
	expected, actual INTEGER,
) error {
	if expected != actual {
		return errLength[INTEGER]{expected, actual}
	} else {
		return nil
	}
}

func (err errLength[_]) Error() string {
	return fmt.Sprintf(
		"expected digest to have length %d, but got %d",
		err[0],
		err[1],
	)
}

func MakeErrWrongType(expected, actual string) error {
	if expected != actual {
		return errWrongType{expected: expected, actual: actual}
	}

	return nil
}

type errWrongType struct {
	expected, actual string
}

func (err errWrongType) Error() string {
	return fmt.Sprintf(
		"wrong type. expected %q but got %q",
		err.expected,
		err.actual,
	)
}

func (err errWrongType) Is(target error) bool {
	_, ok := target.(errWrongType)
	return ok
}

func (err errWrongType) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

type ErrFormatOperationNotSupported struct {
	Format        domain_interfaces.MarklFormat
	FormatId      string
	OperationName string
}

func (err ErrFormatOperationNotSupported) Error() string {
	if err.Format == nil {
		return fmt.Sprintf(
			"nil format with id %q does not support operation %q",
			err.FormatId,
			err.OperationName,
		)
	} else {
		return fmt.Sprintf(
			"format (%T) with id %q does not support operation %q",
			err.Format,
			err.Format.GetMarklFormatId(),
			err.OperationName,
		)
	}
}

func (err ErrFormatOperationNotSupported) Is(target error) bool {
	_, ok := target.(ErrFormatOperationNotSupported)
	return ok
}

func (err ErrFormatOperationNotSupported) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
