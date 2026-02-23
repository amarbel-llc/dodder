package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

type TomlInventoryArchiveV2 struct {
	HashTypeId      HashType                         `toml:"hash_type-id"`
	CompressionType compression_type.CompressionType `toml:"compression-type"`
	Encryption      markl.Id                         `toml:"encryption"`
	Delta           DeltaConfig                      `toml:"delta"`
	MaxPackSize     uint64                           `toml:"max-pack-size"`
}

var (
	_ ConfigInventoryArchiveDelta = TomlInventoryArchiveV2{}
	_ ConfigMutable               = &TomlInventoryArchiveV2{}
	_                             = registerToml[TomlInventoryArchiveV2](
		Coder.Blob,
		ids.TypeTomlBlobStoreConfigInventoryArchiveV2,
	)
)

func (TomlInventoryArchiveV2) GetBlobStoreType() string {
	return "local-inventory-archive"
}

func (config *TomlInventoryArchiveV2) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	config.CompressionType.SetFlagDefinitions(flagSet)

	config.HashTypeId = HashTypeDefault

	flagSet.Var(
		&config.HashTypeId,
		"hash_type-id",
		"hash type for archive checksums and blob hashes",
	)
}

func (config TomlInventoryArchiveV2) getBasePath() string {
	return ""
}

func (config TomlInventoryArchiveV2) SupportsMultiHash() bool {
	return false
}

func (config TomlInventoryArchiveV2) GetDefaultHashTypeId() string {
	return string(config.HashTypeId)
}

func (config TomlInventoryArchiveV2) GetBlobCompression() interfaces.IOWrapper {
	return &config.CompressionType
}

func (config TomlInventoryArchiveV2) GetBlobEncryption() domain_interfaces.MarklId {
	return config.Encryption
}

func (config TomlInventoryArchiveV2) GetLooseBlobStoreId() blob_store_id.Id {
	var zero blob_store_id.Id
	return zero
}

func (config TomlInventoryArchiveV2) GetCompressionType() compression_type.CompressionType {
	return config.CompressionType
}

func (config TomlInventoryArchiveV2) GetDeltaEnabled() bool {
	return config.Delta.Enabled
}

func (config TomlInventoryArchiveV2) GetDeltaAlgorithm() string {
	return config.Delta.Algorithm
}

func (config TomlInventoryArchiveV2) GetDeltaMinBlobSize() uint64 {
	return config.Delta.MinBlobSize
}

func (config TomlInventoryArchiveV2) GetDeltaMaxBlobSize() uint64 {
	return config.Delta.MaxBlobSize
}

func (config TomlInventoryArchiveV2) GetDeltaSizeRatio() float64 {
	return config.Delta.SizeRatio
}

func (config TomlInventoryArchiveV2) GetMaxPackSize() uint64 {
	return config.MaxPackSize
}
