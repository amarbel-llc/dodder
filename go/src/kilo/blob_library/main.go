package blob_library

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
)

type Library[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct {
	envRepo env_repo.Env
	pool    interfaces.PoolPtr[BLOB, BLOB_PTR]
	domain_interfaces.Format[BLOB, BLOB_PTR]
	resetFunc func(BLOB_PTR)
}

func MakeBlobStore[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	envRepo env_repo.Env,
	format domain_interfaces.Format[BLOB, BLOB_PTR],
	resetFunc func(BLOB_PTR),
) (blobStore *Library[BLOB, BLOB_PTR]) {
	blobStore = &Library[BLOB, BLOB_PTR]{
		envRepo:   envRepo,
		pool:      pool.Make(nil, resetFunc),
		Format:    format,
		resetFunc: resetFunc,
	}

	return blobStore
}

func (library *Library[BLOB, BLOB_PTR]) GetBlob(
	blobId domain_interfaces.MarklId,
) (blobPtr BLOB_PTR, repool interfaces.FuncRepool, err error) {
	var readCloser domain_interfaces.BlobReader

	if readCloser, err = library.envRepo.GetDefaultBlobStore().MakeBlobReader(
		blobId,
	); err != nil {
		err = errors.Wrap(err)
		return blobPtr, repool, err
	}

	defer errors.DeferredCloser(&err, readCloser)

	blobPtr, repool = library.pool.GetWithRepool()

	if _, err = library.DecodeFrom(blobPtr, readCloser); err != nil {
		err = errors.Wrapf(err, "BlobReader: %q", readCloser)
		return blobPtr, repool, err
	}

	actual := readCloser.GetMarklId()

	if err = markl.AssertEqual(blobId, actual); err != nil {
		err = errors.Wrap(err)
		return blobPtr, repool, err
	}

	return blobPtr, repool, err
}

func (library *Library[BLOB, BLOB_PTR]) PutBlob(blob BLOB_PTR) {
	// TODO-P2 implement pool
}

// TODO re-evaluate this strategy
func (library *Library[BLOB, BLOB_PTR]) SaveBlobText(
	blob BLOB_PTR,
) (digest domain_interfaces.MarklId, n int64, err error) {
	var writeCloser domain_interfaces.BlobWriter

	if writeCloser, err = library.envRepo.GetDefaultBlobStore().MakeBlobWriter(
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	defer errors.DeferredCloser(&err, writeCloser)

	if n, err = library.EncodeTo(blob, writeCloser); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	digest = writeCloser.GetMarklId()

	return digest, n, err
}
