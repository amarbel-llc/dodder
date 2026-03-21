package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/markl_age_id"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

// TomlLocalHashBucketedV0 is the V0 configuration for the local hash-bucketed blob store.
type TomlLocalHashBucketedV0 struct {
	BasePath          string                           `toml:"base-path,omitempty"`
	AgeEncryption     markl_age_id.Id                  `toml:"age-encryption,omitempty"`
	CompressionType   compression_type.CompressionType `toml:"compression-type"`
	LockInternalFiles bool                             `toml:"lock-internal-files"`
}

var (
	_ ConfigLocalHashBucketed = TomlLocalHashBucketedV0{}
	_ ConfigLocalMutable      = &TomlLocalHashBucketedV0{}
)

func (TomlLocalHashBucketedV0) GetBlobStoreType() string {
	return "local"
}

func (blobStoreConfig *TomlLocalHashBucketedV0) SetFlagDefinitions(
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

func (blobStoreConfig TomlLocalHashBucketedV0) getBasePath() string {
	return blobStoreConfig.BasePath
}

func (blobStoreConfig TomlLocalHashBucketedV0) GetHashBuckets() []int {
	return []int{2}
}

func (blobStoreConfig TomlLocalHashBucketedV0) GetBlobCompression() interfaces.IOWrapper {
	return &blobStoreConfig.CompressionType
}

func (blobStoreConfig TomlLocalHashBucketedV0) GetBlobEncryption() domain_interfaces.MarklId {
	return &blobStoreConfig.AgeEncryption
}

func (blobStoreConfig TomlLocalHashBucketedV0) GetLockInternalFiles() bool {
	return blobStoreConfig.LockInternalFiles
}

func (blobStoreConfig TomlLocalHashBucketedV0) SupportsMultiHash() bool {
	return false
}

func (blobStoreConfig TomlLocalHashBucketedV0) GetDefaultHashTypeId() string {
	return markl.FormatIdHashSha256
}

func (blobStoreConfig *TomlLocalHashBucketedV0) setBasePath(value string) {
	blobStoreConfig.BasePath = value
}
