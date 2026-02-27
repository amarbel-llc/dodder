package remote_transfer

import "code.linenisgreat.com/dodder/go/lib/alfa/errors"

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var (
	ErrSkipped = newPkgError("skipped due to exclude objects option")

	ErrNeedsMerge = errors.Err409Conflict.Errorf(
		"import failed with conflicts, merging required",
	)
)
