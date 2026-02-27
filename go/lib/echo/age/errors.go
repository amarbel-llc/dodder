package age

import (
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"filippo.io/age"
)

type NoIdentityMatchError = age.NoIdentityMatchError

func IsNoIdentityMatchError(err error) bool {
	_, ok := errors.Unwrap(err).(*NoIdentityMatchError)
	return ok
}
