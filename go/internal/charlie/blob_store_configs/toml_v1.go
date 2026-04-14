package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

// TomlLocalHashBucketedV1 is the V1 configuration for the local hash-bucketed blob store.
//
//go:generate tommy generate
type TomlLocalHashBucketedV1 struct {
	HashBuckets values.IntSlice `toml:"hash-buckets"`
	BasePath    string          `toml:"base-path,omitempty"`
	HashTypeId  HashType        `toml:"hash_type-id"`

	// cannot use `omitempty`, as markl.Id's empty value equals its non-empty
	// value due to unexported fields
	Encryption markl.Id `toml:"encryption"`

	CompressionType   compression_type.CompressionType `toml:"compression-type"`
	LockInternalFiles bool                             `toml:"lock-internal-files"`
}


func (TomlLocalHashBucketedV1) GetBlobStoreType() string {
	return "local"
}

func (blobStoreConfig *TomlLocalHashBucketedV1) SetFlagDefinitions(
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

func (blobStoreConfig TomlLocalHashBucketedV1) getBasePath() string {
	return blobStoreConfig.BasePath
}

func (blobStoreConfig TomlLocalHashBucketedV1) GetHashBuckets() []int {
	return blobStoreConfig.HashBuckets
}

func (blobStoreConfig TomlLocalHashBucketedV1) GetBlobCompression() interfaces.IOWrapper {
	return &blobStoreConfig.CompressionType
}

func (blobStoreConfig TomlLocalHashBucketedV1) GetBlobEncryption() domain_interfaces.MarklId {
	return blobStoreConfig.Encryption
}

func (blobStoreConfig TomlLocalHashBucketedV1) GetLockInternalFiles() bool {
	return blobStoreConfig.LockInternalFiles
}

func (blobStoreConfig TomlLocalHashBucketedV1) SupportsMultiHash() bool {
	return true
}

func (blobStoreConfig TomlLocalHashBucketedV1) GetDefaultHashTypeId() string {
	return string(blobStoreConfig.HashTypeId)
}

func (blobStoreConfig *TomlLocalHashBucketedV1) setBasePath(value string) {
	blobStoreConfig.BasePath = value
}

func (blobStoreConfig TomlLocalHashBucketedV1) Upgrade() (Config, ids.TypeStruct) {
	upgraded := &TomlLocalHashBucketedV2{
		HashBuckets:       blobStoreConfig.HashBuckets,
		BasePath:          blobStoreConfig.BasePath,
		HashTypeId:        HashTypeSha256,
		CompressionType:   blobStoreConfig.CompressionType,
		LockInternalFiles: blobStoreConfig.LockInternalFiles,
	}

	upgraded.Encryption.ResetWithMarklId(blobStoreConfig.Encryption)

	return upgraded, ids.GetOrPanic(ids.TypeTomlBlobStoreConfigV2).TypeStruct
}
