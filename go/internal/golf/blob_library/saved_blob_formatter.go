package blob_library

import (
	"io"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type savedBlobFormatter struct {
	blobReaderFactory mad_domain_interfaces.BlobReaderFactory
}

func MakeSavedBlobFormatter(
	blobReaderFactory mad_domain_interfaces.BlobReaderFactory,
) savedBlobFormatter {
	return savedBlobFormatter{
		blobReaderFactory: blobReaderFactory,
	}
}

func (formatter savedBlobFormatter) FormatSavedBlob(
	writer io.Writer,
	digest mad_domain_interfaces.MarklId,
) (n int64, err error) {
	var blobReader mad_domain_interfaces.BlobReader

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
