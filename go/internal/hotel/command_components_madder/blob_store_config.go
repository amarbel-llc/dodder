package command_components_madder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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
