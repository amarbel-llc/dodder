package env_repo

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

var _ mad_domain_interfaces.NamedBlobAccess = Env{}

func MakeNamedBlobReaderOrNullReader(
	blobAccess mad_domain_interfaces.NamedBlobAccess,
	path string,
) (blobReader mad_domain_interfaces.BlobReader, err error) {
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

func (env Env) MakeNamedBlobReader(path string) (mad_domain_interfaces.BlobReader, error) {
	return env_dir.NewFileReaderOrErrNotExist(env_dir.DefaultConfig, path)
}

func (env Env) MakeNamedBlobWriter(
	path string,
) (mad_domain_interfaces.BlobWriter, error) {
	return env_dir.NewMover(
		env_dir.DefaultConfig,
		env_dir.MoveOptions{
			FinalPathOrDir: path,
			TemporaryFS:    env.GetTempLocal(),
		},
	)
}
