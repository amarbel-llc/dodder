package typed_blob_store

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type nopBlobParseSaver[
	OBJECT any,
	OBJECT_PTR interfaces.Ptr[OBJECT],
] struct {
	blobStore domain_interfaces.BlobWriterFactory
}

func MakeNopBlobParseSaver[
	OBJECT any,
	OBJECT_PTR interfaces.Ptr[OBJECT],
](awf domain_interfaces.BlobWriterFactory,
) nopBlobParseSaver[OBJECT, OBJECT_PTR] {
	return nopBlobParseSaver[OBJECT, OBJECT_PTR]{
		blobStore: awf,
	}
}

func (parseSaver nopBlobParseSaver[OBJECT, OBJECT_PTR]) ParseBlob(
	reader io.Reader,
	object OBJECT_PTR,
) (n int64, err error) {
	var blobWriter domain_interfaces.BlobWriter

	if blobWriter, err = parseSaver.blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if n, err = io.Copy(blobWriter, reader); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
