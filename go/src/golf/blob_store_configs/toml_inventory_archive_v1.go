package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

// DeltaConfig holds configuration for delta compression in inventory archives.
type DeltaConfig struct {
	Enabled     bool    `toml:"enabled"`
	Algorithm   string  `toml:"algorithm"`
	MinBlobSize uint64  `toml:"min-blob-size"`
	MaxBlobSize uint64  `toml:"max-blob-size"`
	SizeRatio   float64 `toml:"size-ratio"`
}

// TomlInventoryArchiveV1 is the V1 configuration for the inventory archive
// blob store. Adds delta compression settings.
type TomlInventoryArchiveV1 struct {
	HashTypeId       string                           `toml:"hash_type-id"`
	CompressionType  compression_type.CompressionType `toml:"compression-type"`
	LooseBlobStoreId blob_store_id.Id                 `toml:"loose-blob-store-id"`
	Encryption       markl.Id                         `toml:"encryption"`
	Delta            DeltaConfig                      `toml:"delta"`
}

var (
	_ ConfigInventoryArchiveDelta = TomlInventoryArchiveV1{}
	_ ConfigMutable               = &TomlInventoryArchiveV1{}
	_                             = registerToml[TomlInventoryArchiveV1](
		Coder.Blob,
		ids.TypeTomlBlobStoreConfigInventoryArchiveV1,
	)
)

func (TomlInventoryArchiveV1) GetBlobStoreType() string {
	return "local-inventory-archive"
}

func (config *TomlInventoryArchiveV1) SetFlagDefinitions(
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

func (config TomlInventoryArchiveV1) getBasePath() string {
	return ""
}

func (config TomlInventoryArchiveV1) SupportsMultiHash() bool {
	return false
}

func (config TomlInventoryArchiveV1) GetDefaultHashTypeId() string {
	return config.HashTypeId
}

func (config TomlInventoryArchiveV1) GetBlobCompression() interfaces.IOWrapper {
	return &config.CompressionType
}

func (config TomlInventoryArchiveV1) GetBlobEncryption() domain_interfaces.MarklId {
	return config.Encryption
}

func (config TomlInventoryArchiveV1) GetLooseBlobStoreId() blob_store_id.Id {
	return config.LooseBlobStoreId
}

func (config TomlInventoryArchiveV1) GetCompressionType() compression_type.CompressionType {
	return config.CompressionType
}

// DeltaConfigImmutable implementation

func (config TomlInventoryArchiveV1) GetDeltaEnabled() bool {
	return config.Delta.Enabled
}

func (config TomlInventoryArchiveV1) GetDeltaAlgorithm() string {
	return config.Delta.Algorithm
}

func (config TomlInventoryArchiveV1) GetDeltaMinBlobSize() uint64 {
	return config.Delta.MinBlobSize
}

func (config TomlInventoryArchiveV1) GetDeltaMaxBlobSize() uint64 {
	return config.Delta.MaxBlobSize
}

func (config TomlInventoryArchiveV1) GetDeltaSizeRatio() float64 {
	return config.Delta.SizeRatio
}
