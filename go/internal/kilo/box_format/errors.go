package box_format

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

type ErrBoxParse struct {
	error
}

func (err ErrBoxParse) Is(target error) bool {
	_, ok := target.(ErrBoxParse)
	return ok
}

func (err ErrBoxParse) Unwrap() error {
	return err.error
}

func (err ErrBoxParse) Error() string {
	return fmt.Sprintf("parsing box failed: %s", err.error.Error())
}

func (err ErrBoxParse) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

var ErrNotABox = newPkgError("not a box")

type ErrBoxReadSeq struct {
	expected string
	actual   doddish.Seq
}

func (err ErrBoxReadSeq) Is(target error) bool {
	_, ok := target.(ErrBoxReadSeq)
	return ok
}

func (err ErrBoxReadSeq) Error() string {
	return fmt.Sprintf(
		"box parse error: expected %s but got %s",
		err.expected,
		err.actual,
	)
}

func (err ErrBoxReadSeq) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

type ErrUnsupportedDodderTag struct {
	tag string
}

func (err ErrUnsupportedDodderTag) Error() string {
	return fmt.Sprintf("unsupported dodder tag: %q", err.tag)
}

func (err ErrUnsupportedDodderTag) Is(target error) bool {
	_, ok := target.(ErrUnsupportedDodderTag)
	return ok
}

func (err ErrUnsupportedDodderTag) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
