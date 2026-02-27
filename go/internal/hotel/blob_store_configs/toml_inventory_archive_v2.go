package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
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
	_ SignatureConfigImmutable    = TomlInventoryArchiveV2{}
	_ SelectorConfigImmutable     = TomlInventoryArchiveV2{}
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

	setEncryptionFlagDefinition(flagSet, &config.Encryption)

	flagSet.BoolVar(
		&config.Delta.Enabled,
		"delta",
		false,
		"enable delta compression",
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

// DeltaConfigImmutable implementation

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

// SignatureConfigImmutable implementation

func (config TomlInventoryArchiveV2) GetSignatureType() string {
	return config.Delta.Signature.Type
}

func (config TomlInventoryArchiveV2) GetSignatureLen() int {
	return config.Delta.Signature.SignatureLen
}

func (config TomlInventoryArchiveV2) GetAvgChunkSize() int {
	return config.Delta.Signature.AvgChunkSize
}

func (config TomlInventoryArchiveV2) GetMinChunkSize() int {
	return config.Delta.Signature.MinChunkSize
}

func (config TomlInventoryArchiveV2) GetMaxChunkSize() int {
	return config.Delta.Signature.MaxChunkSize
}

// SelectorConfigImmutable implementation

func (config TomlInventoryArchiveV2) GetSelectorType() string {
	return config.Delta.Selector.Type
}

func (config TomlInventoryArchiveV2) GetSelectorBands() int {
	return config.Delta.Selector.Bands
}

func (config TomlInventoryArchiveV2) GetSelectorRowsPerBand() int {
	return config.Delta.Selector.RowsPerBand
}

func (config TomlInventoryArchiveV2) GetSelectorMinBlobSize() uint64 {
	return config.Delta.Selector.MinBlobSize
}

func (config TomlInventoryArchiveV2) GetSelectorMaxBlobSize() uint64 {
	return config.Delta.Selector.MaxBlobSize
}

func (config TomlInventoryArchiveV2) GetMaxPackSize() uint64 {
	return config.MaxPackSize
}
