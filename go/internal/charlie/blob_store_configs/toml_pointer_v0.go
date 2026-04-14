package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
)

//go:generate tommy generate
type TomlPointerV0 struct {
	Id         blob_store_id.Id `toml:"id"`
	BasePath   string           `toml:"base-path"`
	ConfigPath string           `toml:"config-path"`
}

func (TomlPointerV0) GetBlobStoreType() string {
	return "local-pointer"
}

func (blobStoreConfig *TomlPointerV0) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.Var(
		&blobStoreConfig.Id,
		"id",
		"another blob store's id",
	)

	flagSet.StringVar(
		&blobStoreConfig.BasePath,
		"base-path",
		"",
		"path to another blob store base directory",
	)

	flagSet.StringVar(
		&blobStoreConfig.ConfigPath,
		"config-path",
		"",
		"path to another blob store config file",
	)
}

func (blobStoreConfig TomlPointerV0) GetPath() directory_layout.BlobStorePath {
	return directory_layout.MakeBlobStorePath(
		blobStoreConfig.Id,
		blobStoreConfig.BasePath,
		blobStoreConfig.ConfigPath,
	)
}
