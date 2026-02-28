package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type TomlPointerV0 struct {
	Id         blob_store_id.Id `toml:"id"`
	BasePath   string           `toml:"base-path"`
	ConfigPath string           `toml:"config-path"`
}

var (
	_ ConfigPointer = TomlPointerV0{}
	_ ConfigMutable = &TomlPointerV0{}
	_               = registerToml[TomlPointerV0](
		Coder.Blob,
		ids.TypeTomlBlobStoreConfigPointerV0,
	)
)

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
