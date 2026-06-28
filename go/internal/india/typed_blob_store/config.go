package typed_blob_store

import (
	"io"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_library"
	"github.com/amarbel-llc/hyphence/go/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type Config struct {
	toml_v0 mad_domain_interfaces.TypedStore[repo_configs.V0, *repo_configs.V0]
	toml_v1 mad_domain_interfaces.TypedStore[repo_configs.V1, *repo_configs.V1]
}

func MakeConfigStore(
	envRepo env_repo.Env,
) Config {
	return Config{
		toml_v0: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat(
				hyphence.TommyBlobDecoder[repo_configs.V0, *repo_configs.V0]{
					Decode: func(b []byte) (repo_configs.V0, error) {
						doc, err := repo_configs.DecodeV0(b)
						if err != nil {
							return repo_configs.V0{}, err
						}
						return *doc.Data(), nil
					},
				},
				hyphence.TommyBlobEncoder[repo_configs.V0, *repo_configs.V0]{
					Encode: func(v repo_configs.V0) ([]byte, error) {
						doc, err := repo_configs.DecodeV0(nil)
						if err != nil {
							return nil, err
						}
						*doc.Data() = v
						return doc.Encode()
					},
				},
				envRepo.GetDefaultBlobStore(),
			),
			func(a *repo_configs.V0) {
				a.Reset()
			},
		),
		toml_v1: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat(
				hyphence.TommyBlobDecoder[repo_configs.V1, *repo_configs.V1]{
					Decode: func(b []byte) (repo_configs.V1, error) {
						doc, err := repo_configs.DecodeV1(b)
						if err != nil {
							return repo_configs.V1{}, err
						}
						return *doc.Data(), nil
					},
				},
				hyphence.TommyBlobEncoder[repo_configs.V1, *repo_configs.V1]{
					Encode: func(v repo_configs.V1) ([]byte, error) {
						doc, err := repo_configs.DecodeV1(nil)
						if err != nil {
							return nil, err
						}
						*doc.Data() = v
						return doc.Encode()
					},
				},
				envRepo.GetDefaultBlobStore(),
			),
			func(a *repo_configs.V1) {
				a.Reset()
			},
		),
	}
}

func (a Config) ParseTypedBlob(
	tipe ids.Type,
	blobId mad_domain_interfaces.MarklId,
) (common repo_configs.ConfigOverlay, repool interfaces.FuncRepool, n int64, err error) {
	switch tipe.String() {
	case "", ids.TypeTomlConfigV0:
		store := a.toml_v0
		var blob *repo_configs.V0

		if blob, repool, err = store.GetBlob(blobId); err != nil {
			err = errors.Wrap(err)
			return common, repool, n, err
		}

		common = blob

	case ids.TypeTomlConfigV1:
		store := a.toml_v1
		var blob *repo_configs.V1

		if blob, repool, err = store.GetBlob(blobId); err != nil {
			err = errors.Wrap(err)
			return common, repool, n, err
		}

		common = blob
	}

	return common, repool, n, err
}

func (a Config) FormatTypedBlob(
	objectGetter sku.TransactedGetter,
	writer io.Writer,
) (n int64, err error) {
	object := objectGetter.GetSku()

	tipe := object.GetType()
	blobSha := object.GetBlobDigest()

	var store mad_domain_interfaces.SavedBlobFormatter
	switch tipe.String() {
	case "", ids.TypeTomlConfigV0:
		store = a.toml_v0

	case ids.TypeTomlConfigV1:
		store = a.toml_v1
	}

	if n, err = store.FormatSavedBlob(writer, blobSha); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
