package queries

import (
	"fmt"

	mad_blob_io "github.com/amarbel-llc/madder/go/pkgs/blob_io"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrBlobMissing struct {
	ObjectId
	mad_blob_io.ErrBlobMissing
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
