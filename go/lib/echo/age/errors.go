package age

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"filippo.io/age"
)

type NoIdentityMatchError = age.NoIdentityMatchError

func IsNoIdentityMatchError(err error) bool {
	_, ok := errors.Unwrap(err).(*NoIdentityMatchError)
	return ok
}
