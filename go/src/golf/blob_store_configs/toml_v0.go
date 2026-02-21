package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/markl_age_id"
)

// TomlV0 is the V0 configuration for the local hash-bucketed blob store.
// TODO rename to TomlLocalHashBucketedV0 for disambiguation from other
// blob store config types (e.g., TomlInventoryArchiveV0).
type TomlV0 struct {
	BasePath          string                           `toml:"base-path,omitempty"`
	AgeEncryption     markl_age_id.Id                  `toml:"age-encryption,omitempty"`
	CompressionType   compression_type.CompressionType `toml:"compression-type"`
	LockInternalFiles bool                             `toml:"lock-internal-files"`
}

var (
	_ ConfigLocalHashBucketed = TomlV0{}
	_ ConfigLocalMutable      = &TomlV0{}
)

func (TomlV0) GetBlobStoreType() string {
	return "local"
}

func (blobStoreConfig *TomlV0) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	blobStoreConfig.CompressionType.SetFlagDefinitions(flagSet)

	flagSet.BoolVar(
		&blobStoreConfig.LockInternalFiles,
		"lock-internal-files",
		blobStoreConfig.LockInternalFiles,
		"",
	)

	flagSet.Var(
		&blobStoreConfig.AgeEncryption,
		"age-identity",
		"add an age identity",
	)
}

func (blobStoreConfig TomlV0) getBasePath() string {
	return blobStoreConfig.BasePath
}

func (blobStoreConfig TomlV0) GetHashBuckets() []int {
	return []int{2}
}

func (blobStoreConfig TomlV0) GetBlobCompression() interfaces.IOWrapper {
	return &blobStoreConfig.CompressionType
}

func (blobStoreConfig TomlV0) GetBlobEncryption() domain_interfaces.MarklId {
	return &blobStoreConfig.AgeEncryption
}

func (blobStoreConfig TomlV0) GetLockInternalFiles() bool {
	return blobStoreConfig.LockInternalFiles
}

func (blobStoreConfig TomlV0) SupportsMultiHash() bool {
	return false
}

func (blobStoreConfig TomlV0) GetDefaultHashTypeId() string {
	return markl.FormatIdHashSha256
}

func (blobStoreConfig *TomlV0) setBasePath(value string) {
	blobStoreConfig.BasePath = value
}
