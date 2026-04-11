package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

// TomlLocalHashBucketedV2 is the V2 configuration for the local hash-bucketed blob store.
//
//go:generate tommy generate
type TomlLocalHashBucketedV2 struct {
	HashBuckets values.IntSlice `toml:"hash_buckets"`
	BasePath    string          `toml:"base_path,omitempty"`
	HashTypeId  HashType        `toml:"hash_type-id"`

	// cannot use `omitempty`, as markl.Id's empty value equals its non-empty
	// value due to unexported fields
	Encryption markl.Id `toml:"encryption"`

	CompressionType   compression_type.CompressionType `toml:"compression-type"`
	LockInternalFiles bool                             `toml:"lock-internal-files"`
}

func (TomlLocalHashBucketedV2) GetBlobStoreType() string {
	return "local"
}

func (blobStoreConfig *TomlLocalHashBucketedV2) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	blobStoreConfig.CompressionType.SetFlagDefinitions(flagSet)

	blobStoreConfig.HashBuckets = DefaultHashBuckets

	flagSet.Var(
		&blobStoreConfig.HashBuckets,
		"hash_buckets",
		"determines hash bucketing directory structure",
	)

	blobStoreConfig.HashTypeId = HashTypeDefault

	flagSet.Var(
		&blobStoreConfig.HashTypeId,
		"hash_type-id",
		"determines the hash type used for new blobs written to the store",
	)

	setEncryptionFlagDefinition(flagSet, &blobStoreConfig.Encryption)

	flagSet.BoolVar(
		&blobStoreConfig.LockInternalFiles,
		"lock-internal-files",
		blobStoreConfig.LockInternalFiles,
		"",
	)
}

func (blobStoreConfig TomlLocalHashBucketedV2) getBasePath() string {
	return blobStoreConfig.BasePath
}

func (blobStoreConfig TomlLocalHashBucketedV2) GetHashBuckets() []int {
	return blobStoreConfig.HashBuckets
}

func (blobStoreConfig TomlLocalHashBucketedV2) GetBlobCompression() interfaces.IOWrapper {
	return &blobStoreConfig.CompressionType
}

func (blobStoreConfig TomlLocalHashBucketedV2) GetBlobEncryption() domain_interfaces.MarklId {
	return blobStoreConfig.Encryption
}

func (blobStoreConfig TomlLocalHashBucketedV2) GetLockInternalFiles() bool {
	return blobStoreConfig.LockInternalFiles
}

func (blobStoreConfig TomlLocalHashBucketedV2) SupportsMultiHash() bool {
	return true
}

func (blobStoreConfig TomlLocalHashBucketedV2) GetDefaultHashTypeId() string {
	return string(blobStoreConfig.HashTypeId)
}

func (blobStoreConfig *TomlLocalHashBucketedV2) setBasePath(value string) {
	blobStoreConfig.BasePath = value
}

func (blobStoreConfig TomlLocalHashBucketedV2) Upgrade() (Config, ids.TypeStruct) {
	upgraded := &TomlV3{
		HashBuckets:       blobStoreConfig.HashBuckets,
		BasePath:          blobStoreConfig.BasePath,
		HashTypeId:        blobStoreConfig.HashTypeId,
		CompressionType:   blobStoreConfig.CompressionType,
		LockInternalFiles: blobStoreConfig.LockInternalFiles,
	}

	if !blobStoreConfig.Encryption.IsNull() {
		upgraded.Encryption = []markl.Id{blobStoreConfig.Encryption}
	}

	return upgraded, ids.GetOrPanic(ids.TypeTomlBlobStoreConfigV3).TypeStruct
}
