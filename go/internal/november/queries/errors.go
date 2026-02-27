package queries

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrBlobMissing struct {
	ObjectId
	env_dir.ErrBlobMissing
}

// TODO add recovery text
func (err ErrBlobMissing) Error() string {
	return fmt.Sprintf(
		"Blob for %q with sha %q does not exist locally.",
		err.ObjectId,
		err.BlobId,
	)
}

func (err ErrBlobMissing) Is(target error) bool {
	_, ok := target.(ErrBlobMissing)
	return ok
}

func (err ErrBlobMissing) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrBlobMissing(err error) bool {
	return errors.Is(err, ErrBlobMissing{})
}
