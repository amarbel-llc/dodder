package env_dir

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func IsErrBlobAlreadyExists(err error) bool {
	return errors.Is(err, ErrBlobAlreadyExists{})
}

func MakeErrBlobAlreadyExists(
	blobId domain_interfaces.MarklId,
	path string,
) ErrBlobAlreadyExists {
	errors.PanicIfError(markl.AssertIdIsNotNull(blobId))

	clonedId, _ := markl.Clone(blobId)

	return ErrBlobAlreadyExists{
		Path:   path,
		BlobId: clonedId,
	}
}

type ErrBlobAlreadyExists struct {
	BlobId domain_interfaces.MarklId
	Path   string
}

func (err ErrBlobAlreadyExists) Error() string {
	return fmt.Sprintf(
		"File with blob_id %s already exists: %s",
		err.BlobId,
		err.Path,
	)
}

func (err ErrBlobAlreadyExists) Is(target error) bool {
	_, ok := target.(ErrBlobAlreadyExists)
	return ok
}

func (err ErrBlobAlreadyExists) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrBlobMissing(err error) bool {
	return errors.Is(err, ErrBlobMissing{})
}

// TODO create a constructor function to enable debugging
type ErrBlobMissing struct {
	// TODO add blob store
	BlobId domain_interfaces.MarklId
	Path   string
}

func (err ErrBlobMissing) Error() string {
	if err.Path == "" {
		return fmt.Sprintf(
			"Blob with id %q does not exist locally",
			err.BlobId,
		)
	} else {
		return fmt.Sprintf(
			"Blob with id %q does not exist locally: %q",
			err.BlobId,
			err.Path,
		)
	}
}

func (err ErrBlobMissing) Is(target error) bool {
	_, ok := target.(ErrBlobMissing)
	return ok
}

func (err ErrBlobMissing) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func MakeErrTempAlreadyExists(
	path string,
) (err ErrTempAlreadyExists) {
	err = ErrTempAlreadyExists{Path: path}
	return err
}

var _ errors.Helpful = ErrTempAlreadyExists{}

type ErrTempAlreadyExists struct {
	Path string
}

func (err ErrTempAlreadyExists) Error() string {
	return fmt.Sprintf("Local temporary directory already exists: %q", err.Path)
}

func (err ErrTempAlreadyExists) GetErrorCause() []string {
	return []string{
		"Another dodder previous process with the same PID likely terminated unexpectedly",
	}
}

func (err ErrTempAlreadyExists) GetErrorRecovery() []string {
	return []string{
		"Check if there are any relevant files in the directory, or possible delete it",
	}
}

func (err ErrTempAlreadyExists) Is(target error) bool {
	_, ok := target.(ErrTempAlreadyExists)
	return ok
}

func (err ErrTempAlreadyExists) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
