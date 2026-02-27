package blob_library

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type savedBlobFormatter struct {
	blobReaderFactory domain_interfaces.BlobReaderFactory
}

func MakeSavedBlobFormatter(
	blobReaderFactory domain_interfaces.BlobReaderFactory,
) savedBlobFormatter {
	return savedBlobFormatter{
		blobReaderFactory: blobReaderFactory,
	}
}

func (formatter savedBlobFormatter) FormatSavedBlob(
	writer io.Writer,
	digest domain_interfaces.MarklId,
) (n int64, err error) {
	var blobReader domain_interfaces.BlobReader

	if blobReader, err = formatter.blobReaderFactory.MakeBlobReader(
		digest,
	); err != nil {
		if errors.IsNotExist(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return n, err
	}

	defer errors.DeferredCloser(&err, blobReader)

	if n, err = io.Copy(writer, blobReader); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
