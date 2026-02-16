package object_metadata_fmt_triple_hyphen

import (
	"fmt"
	"os/exec"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func MakeErrBlobFormatterFailed(
	err *exec.ExitError,
) ErrBlobFormatterFailed {
	return ErrBlobFormatterFailed{ExitError: err}
}

type ErrBlobFormatterFailed struct {
	*exec.ExitError
}

func (err ErrBlobFormatterFailed) Error() string {
	return fmt.Sprintf(
		"blob formatter failed (exit status: %d): %q",
		err.ExitCode(),
		err.Stderr,
	)
}

func (err ErrBlobFormatterFailed) Is(target error) bool {
	_, ok := target.(ErrBlobFormatterFailed)
	return ok
}

func (err ErrBlobFormatterFailed) ShouldShowStackTrace() bool {
	return false
}

func (err ErrBlobFormatterFailed) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func MakeErrHasInlineBlobAndFilePath(
	blobFD *fd.FD,
	inlineBlobDigest domain_interfaces.MarklId,
) (err *ErrHasInlineBlobAndFilePath) {
	err = &ErrHasInlineBlobAndFilePath{}
	err.BlobFD.ResetWith(blobFD)
	err.InlineDigest.SetDigest(inlineBlobDigest)
	return err
}

type ErrHasInlineBlobAndFilePath struct {
	BlobFD       fd.FD
	InlineDigest markl.Id
}

func (err *ErrHasInlineBlobAndFilePath) Error() string {
	return fmt.Sprintf(
		"text has inline blob and file: \nexternal path: %s\nexternal sha: %s\ninline sha: %s",
		err.BlobFD.GetPath(),
		err.BlobFD.GetDigest(),
		&err.InlineDigest,
	)
}

func (err *ErrHasInlineBlobAndFilePath) Is(target error) bool {
	_, ok := target.(*ErrHasInlineBlobAndFilePath)
	return ok
}

func (err *ErrHasInlineBlobAndFilePath) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func MakeErrHasInlineBlobAndMetadataBlobId(
	inline, metadata domain_interfaces.MarklId,
) (err *ErrHasInlineBlobAndMetadataDigest) {
	err = &ErrHasInlineBlobAndMetadataDigest{}
	err.metadata, _ = markl.Clone(metadata)
	err.Inline, _ = markl.Clone(inline)
	return err
}

type ErrHasInlineBlobAndMetadataDigest struct {
	Inline   domain_interfaces.MarklId
	metadata domain_interfaces.MarklId
}

func (err *ErrHasInlineBlobAndMetadataDigest) Error() string {
	return fmt.Sprintf(
		"text has inline blob and metadata blob id: \ninline blob id: %s\n metadata blob id: %s",
		err.Inline,
		err.metadata,
	)
}

func (err *ErrHasInlineBlobAndMetadataDigest) Is(target error) bool {
	_, ok := target.(*ErrHasInlineBlobAndMetadataDigest)
	return ok
}

func (err *ErrHasInlineBlobAndMetadataDigest) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}
