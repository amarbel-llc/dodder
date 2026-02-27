package doddish

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type (
	pkgErrDisamb struct{}
	Error        = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) Error {
	return errors.NewWithType[pkgErrDisamb](text)
}

var (
	ErrEmptySeq       = newPkgError("empty seq")
	ErrMoreThanOneSeq = newPkgError("more than one seq")
)

type ErrUnsupportedSeq struct {
	For string
	Seq
}

func IsErrEmptySeq(err error) bool {
	return errors.Is(err, ErrEmptySeq)
}

func IsErrUnsupportedSeq(err error) bool {
	return errors.Is(err, ErrUnsupportedSeq{})
}

func (err ErrUnsupportedSeq) Error() string {
	if err.For == "" {
		return fmt.Sprintf("unsupported seq: %q", err.Seq)
	} else {
		return fmt.Sprintf("unsupported seq %q for %q", err.Seq, err.For)
	}
}

func (err ErrUnsupportedSeq) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrUnsupportedSeq) Is(target error) bool {
	if _, ok := target.(ErrUnsupportedSeq); ok {
		return true
	}

	return false
}
