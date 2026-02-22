package command_components_madder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
)

type Init struct{}

func (cmd Init) InitBlobStore(
	ctx interfaces.ActiveContext,
	envBlobStore env_repo.BlobStoreEnv,
	id blob_store_id.Id,
	config *blob_store_configs.TypedConfig,
) (path directory_layout.BlobStorePath) {
	path = directory_layout.GetBlobStorePath(
		envBlobStore,
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
