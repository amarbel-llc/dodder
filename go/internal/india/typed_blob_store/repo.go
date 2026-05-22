package typed_blob_store

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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
	blobSha mad_domain_interfaces.MarklId,
) (common repo_blobs.Blob, n int64, err error) {
	var reader mad_domain_interfaces.BlobReader

	if reader, err = store.envRepo.GetReadBlobStore().MakeBlobReader(blobSha); err != nil {
		err = errors.Wrap(err)
		return common, n, err
	}

	defer errors.DeferredCloser(&err, reader)

	typedBlob := repo_blobs.TypedBlob{
		Type: tipe.ToType().ToMadder(),
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
) (sh mad_domain_interfaces.MarklId, n int64, err error) {
	var writer mad_domain_interfaces.BlobWriter

	if writer, err = store.envRepo.GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return sh, n, err
	}

	defer errors.DeferredCloser(&err, writer)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(writer)
	defer repoolBufferedWriter()

	if n, err = repo_blobs.Coder.EncodeTo(
		&repo_blobs.TypedBlob{
			Type: tipe.ToType().ToMadder(),
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
