package env_repo

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_configs"
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

// MakeNamedBlobWriter returns an atomic-rename overwrite writer.
// Named blobs (e.g. zettel_id_index bitset cache) need overwrite
// semantics, distinct from madder's content-addressed publish
// (blob_io.NewMover) which treats os.Link ErrExist as a benign
// same-digest collision. See named_blob_writer.go for the rationale.
func (env Env) MakeNamedBlobWriter(
	path string,
) (mad_domain_interfaces.BlobWriter, error) {
	return makeNamedBlobWriter(
		path,
		blob_store_configs.DefaultHashType,
		env.GetTempLocal(),
	)
}
