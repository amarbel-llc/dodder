package checkout_mode

import "github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"

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
