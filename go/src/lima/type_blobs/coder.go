package type_blobs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
)

type Coder struct {
	envRepo env_repo.Env
}

func MakeTypeStore(
	envRepo env_repo.Env,
) Coder {
	return Coder{
		envRepo: envRepo,
	}
}

func (store Coder) SaveBlobText(
	tipe domain_interfaces.ObjectId,
	blob Blob,
) (digest domain_interfaces.MarklId, n int64, err error) {
	if err = genres.Type.AssertGenre(tipe); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	var writer domain_interfaces.BlobWriter

	if writer, err = store.envRepo.GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	defer errors.DeferredCloser(&err, writer)

	tipeString := tipe.String()

	if tipeString == "" {
		tipeString = ids.TypeTomlTypeV0
	}

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(writer)
	defer repoolBufferedWriter()

	if n, err = CoderToTypedBlob.Blob.EncodeTo(
		&TypedBlob{
			Type: ids.MustTypeStruct(tipeString),
			Blob: blob,
		},
		bufferedWriter,
	); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	digest = writer.GetMarklId()

	return digest, n, err
}

func (store Coder) ParseTypedBlob(
	tipe domain_interfaces.ObjectId,
	blobId domain_interfaces.MarklId,
) (common Blob, repool interfaces.FuncRepool, n int64, err error) {
	repool = func() {}

	var reader domain_interfaces.BlobReader

	if reader, err = store.envRepo.GetDefaultBlobStore().MakeBlobReader(blobId); err != nil {
		err = errors.Wrap(err)
		return common, repool, n, err
	}

	defer errors.DeferredCloser(&err, reader)

	tipeString := tipe.String()

	if tipeString == "" {
		tipeString = ids.TypeTomlTypeV0
	}

	typedBlob := TypedBlob{
		Type: ids.MustTypeStruct(tipeString),
	}

	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(reader)
	defer repoolBufferedReader()

	if n, err = CoderToTypedBlob.Blob.DecodeFrom(
		&typedBlob,
		bufferedReader,
	); err != nil {
		err = errors.Wrap(err)
		return common, repool, n, err
	}

	common = typedBlob.Blob

	return common, repool, n, err
}
