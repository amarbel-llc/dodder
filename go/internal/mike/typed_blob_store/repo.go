package typed_blob_store

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/repo_blobs"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
)

type RepoStore struct {
	envRepo env_repo.Env
}

func MakeRepoStore(
	envRepo env_repo.Env,
) RepoStore {
	return RepoStore{
		envRepo: envRepo,
	}
}

func (store RepoStore) ReadTypedBlob(
	tipe ids.Type,
	blobSha domain_interfaces.MarklId,
) (common repo_blobs.Blob, n int64, err error) {
	var reader domain_interfaces.BlobReader

	if reader, err = store.envRepo.GetDefaultBlobStore().MakeBlobReader(blobSha); err != nil {
		err = errors.Wrap(err)
		return common, n, err
	}

	defer errors.DeferredCloser(&err, reader)

	typedBlob := repo_blobs.TypedBlob{
		Type: tipe.ToType(),
	}

	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(reader)
	defer repoolBufferedReader()

	if n, err = repo_blobs.Coder.DecodeFrom(
		&typedBlob,
		bufferedReader,
	); err != nil {
		err = errors.Wrap(err)
		return common, n, err
	}

	common = typedBlob.Blob

	return common, n, err
}

func (store RepoStore) WriteTypedBlob(
	tipe ids.Type,
	blob repo_blobs.Blob,
) (sh domain_interfaces.MarklId, n int64, err error) {
	var writer domain_interfaces.BlobWriter

	if writer, err = store.envRepo.GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return sh, n, err
	}

	defer errors.DeferredCloser(&err, writer)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(writer)
	defer repoolBufferedWriter()

	if n, err = repo_blobs.Coder.EncodeTo(
		&repo_blobs.TypedBlob{
			Type: tipe.ToType(),
			Blob: blob,
		},
		bufferedWriter,
	); err != nil {
		err = errors.Wrap(err)
		return sh, n, err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return sh, n, err
	}

	sh = writer.GetMarklId()

	return sh, n, err
}
