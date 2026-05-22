package age

import (
	"filippo.io/age"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type NoIdentityMatchError = age.NoIdentityMatchError

func IsNoIdentityMatchError(err error) bool {
	_, ok := errors.Unwrap(err).(*NoIdentityMatchError)
	return ok
}
