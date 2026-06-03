package type_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
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
) (digest mad_domain_interfaces.MarklId, n int64, err error) {
	if err = genres.Type.AssertGenre(tipe); err != nil {
		err = errors.Wrap(err)
		return digest, n, err
	}

	var writer mad_domain_interfaces.BlobWriter

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
			Type: ids.MustTypeStruct(tipeString).ToMadder(),
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
	blobId mad_domain_interfaces.MarklId,
) (common Blob, repool interfaces.FuncRepool, n int64, err error) {
	repool = func() {}

	var reader mad_domain_interfaces.BlobReader

	if reader, err = store.envRepo.GetReadBlobStore().MakeBlobReader(blobId); err != nil {
		err = errors.Wrap(err)
		return common, repool, n, err
	}

	defer errors.DeferredCloser(&err, reader)

	tipeString := tipe.String()

	if tipeString == "" {
		tipeString = ids.TypeTomlTypeV0
	}

	typedBlob := TypedBlob{
		Type: ids.MustTypeStruct(tipeString).ToMadder(),
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
