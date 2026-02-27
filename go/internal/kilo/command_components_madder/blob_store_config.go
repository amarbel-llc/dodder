package command_components_madder

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_store_configs"
)

type BlobStoreConfig struct{}

// This method temporarily modifies the config with a resolved base path
func (BlobStoreConfig) PrintBlobStoreConfig(
	ctx interfaces.ActiveContext,
	config *blob_store_configs.TypedConfig,
	out io.Writer,
) (err error) {
	if _, err = blob_store_configs.Coder.EncodeTo(
		&blob_store_configs.TypedConfig{
			Type: config.Type,
			Blob: config.Blob,
		},
		out,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
