package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

type TomlInventoryArchiveV0 struct {
	HashTypeId       string                           `toml:"hash_type-id"`
	CompressionType  compression_type.CompressionType `toml:"compression-type"`
	LooseBlobStoreId blob_store_id.Id                 `toml:"loose-blob-store-id"`
	Encryption       markl.Id                         `toml:"encryption"`
}

var (
	_ ConfigInventoryArchive = TomlInventoryArchiveV0{}
	_ ConfigMutable          = &TomlInventoryArchiveV0{}
	_                        = registerToml[TomlInventoryArchiveV0](
		Coder.Blob,
		ids.TypeTomlBlobStoreConfigInventoryArchiveV0,
	)
)

func (TomlInventoryArchiveV0) GetBlobStoreType() string {
	return "local-inventory-archive"
}

func (config *TomlInventoryArchiveV0) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	config.CompressionType.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&config.HashTypeId,
		"hash_type-id",
		markl.FormatIdHashBlake2b256,
		"hash type for archive checksums and blob hashes",
	)

	flagSet.Var(
		&config.LooseBlobStoreId,
		"loose-blob-store-id",
		"id of the loose blob store to read from and write to",
	)
}

func (config TomlInventoryArchiveV0) getBasePath() string {
	return ""
}

func (config TomlInventoryArchiveV0) SupportsMultiHash() bool {
	return false
}

func (config TomlInventoryArchiveV0) GetDefaultHashTypeId() string {
	return config.HashTypeId
}

func (config TomlInventoryArchiveV0) GetBlobCompression() interfaces.IOWrapper {
	return &config.CompressionType
}

func (config TomlInventoryArchiveV0) GetBlobEncryption() domain_interfaces.MarklId {
	return config.Encryption
}

func (config TomlInventoryArchiveV0) GetLooseBlobStoreId() blob_store_id.Id {
	return config.LooseBlobStoreId
}

func (config TomlInventoryArchiveV0) GetCompressionType() compression_type.CompressionType {
	return config.CompressionType
}
