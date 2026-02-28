package command_components_madder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Init struct{}

func (cmd Init) InitBlobStore(
	ctx interfaces.ActiveContext,
	envBlobStore env_repo.BlobStoreEnv,
	id blob_store_id.Id,
	config *blob_store_configs.TypedConfig,
) (path directory_layout.BlobStorePath) {
	var layout directory_layout.BlobStore = envBlobStore

	if id.GetLocationType() == blob_store_id.LocationTypeCwd {
		xdgForCwd := envBlobStore.GetXDGForBlobStores().CloneWithOverridePath(
			envBlobStore.GetCwd(),
		)

		var err error

		if layout, err = directory_layout.CloneBlobStoreWithXDG(
			envBlobStore,
			xdgForCwd,
		); err != nil {
			err = errors.Wrap(err)
			envBlobStore.Cancel(err)
			return path
		}
	}

	path = directory_layout.GetBlobStorePath(
		layout,
		id.GetName(),
	)

	if err := envBlobStore.MakeDirs(
		filepath.Dir(path.GetBase()),
		filepath.Dir(path.GetConfig()),
	); err != nil {
		envBlobStore.Cancel(err)
		return path
	}

	if err := triple_hyphen_io.EncodeToFile(
		blob_store_configs.Coder,
		config,
		path.GetConfig(),
	); err != nil {
		envBlobStore.Cancel(err)
		return path
	}

	return path
}
