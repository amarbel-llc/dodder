package env_repo

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
)

var _ domain_interfaces.NamedBlobAccess = Env{}

func MakeNamedBlobReaderOrNullReader(
	blobAccess domain_interfaces.NamedBlobAccess,
	path string,
) (blobReader domain_interfaces.BlobReader, err error) {
	if blobReader, err = blobAccess.MakeNamedBlobReader(path); err != nil {
		if errors.IsNotExist(err) {
			return env_dir.NewNopReader()
		} else {
			err = errors.Wrap(err)
			return blobReader, err
		}
	}

	return blobReader, err
}

func (env Env) MakeNamedBlobReader(path string) (domain_interfaces.BlobReader, error) {
	return env_dir.NewFileReaderOrErrNotExist(env_dir.DefaultConfig, path)
}

func (env Env) MakeNamedBlobWriter(
	path string,
) (domain_interfaces.BlobWriter, error) {
	return env_dir.NewMover(
		env_dir.DefaultConfig,
		env_dir.MoveOptions{
			FinalPathOrDir: path,
			TemporaryFS:    env.GetTempLocal(),
		},
	)
}
