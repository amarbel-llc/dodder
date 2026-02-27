package checkout_mode

import "code.linenisgreat.com/dodder/go/internal/alfa/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type errInvalidCheckoutMode error

func MakeErrInvalidCheckoutModeMode(mode Mode) errInvalidCheckoutMode {
	return errors.WrapSkip(
		1,
		errInvalidCheckoutMode(
			errors.ErrorWithStackf("invalid checkout mode: %s", mode),
		),
	)
}

func MakeErrInvalidCheckoutMode(err error) errInvalidCheckoutMode {
	return errInvalidCheckoutMode(err)
}

func IsErrInvalidCheckoutMode(err error) bool {
	return errors.Is(err, errInvalidCheckoutMode(nil))
}
