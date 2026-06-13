package queries

import (
	"fmt"

	mad_blob_io "github.com/amarbel-llc/madder/go/pkgs/blob_io"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

// ErrConfigNotQueryable is returned when a query explicitly names the config
// object (token "konfig" or "config"). Config is no longer a queryable object;
// it lives in a repo-local log read via show-config and written via
// edit-config. This error fires only on an explicit konfig/config token during
// query building; broad genre queries (e.g. ":") are unaffected, and
// config-log entry decode (which parses the "konfig" object id) is untouched.
var ErrConfigNotQueryable = errors.BadRequestf(
	"config is no longer an object; use show-config / edit-config",
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
