package typed_blob_store

import (
	"io"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type nopBlobParseSaver[
	OBJECT any,
	OBJECT_PTR interfaces.Ptr[OBJECT],
] struct {
	blobStore mad_domain_interfaces.BlobWriterFactory
}

func MakeNopBlobParseSaver[
	OBJECT any,
	OBJECT_PTR interfaces.Ptr[OBJECT],
](awf mad_domain_interfaces.BlobWriterFactory,
) nopBlobParseSaver[OBJECT, OBJECT_PTR] {
	return nopBlobParseSaver[OBJECT, OBJECT_PTR]{
		blobStore: awf,
	}
}

func (parseSaver nopBlobParseSaver[OBJECT, OBJECT_PTR]) ParseBlob(
	reader io.Reader,
	object OBJECT_PTR,
) (n int64, err error) {
	var blobWriter mad_domain_interfaces.BlobWriter

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
