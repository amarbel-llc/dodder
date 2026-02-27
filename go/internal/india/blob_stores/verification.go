package blob_stores

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

// TODO offer options like just checking the existence of the blob, getting its
// size, or full verification
func VerifyBlob(
	ctx errors.Context,
	blobStore domain_interfaces.BlobStore,
	expected domain_interfaces.MarklId,
	progressWriter io.Writer,
) (err error) {
	// TODO check if `blobStore` implements a `VerifyBlob` method and call that
	// instead (for expensive blob stores that may implement their own remote
	// verification, such as ssh, sftp, or something else)

	var readCloser domain_interfaces.BlobReader

	if readCloser, err = blobStore.MakeBlobReader(expected); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = io.Copy(progressWriter, readCloser); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertEqual(
		expected,
		readCloser.GetMarklId(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = readCloser.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
