package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

// TomlV1 is the V1 configuration for the local hash-bucketed blob store.
// TODO rename to TomlLocalHashBucketedV1 for disambiguation from other
// blob store config types (e.g., TomlInventoryArchiveV1).
type TomlV1 struct {
	HashBuckets values.IntSlice `toml:"hash-buckets"`
	BasePath    string          `toml:"base-path,omitempty"`
	HashTypeId  HashType        `toml:"hash_type-id"`

	// cannot use `omitempty`, as markl.Id's empty value equals its non-empty
	// value due to unexported fields
	Encryption markl.Id `toml:"encryption"`

	CompressionType   compression_type.CompressionType `toml:"compression-type"`
	LockInternalFiles bool                             `toml:"lock-internal-files"`
}

var (
	_ ConfigLocalHashBucketed = TomlV1{}
	_ ConfigUpgradeable       = TomlV1{}
	_ ConfigLocalMutable      = &TomlV1{}
)

func (TomlV1) GetBlobStoreType() string {
	return "local"
}

func (blobStoreConfig *TomlV1) SetFlagDefinitions(
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

func (blobStoreConfig TomlV1) getBasePath() string {
	return blobStoreConfig.BasePath
}

func (blobStoreConfig TomlV1) GetHashBuckets() []int {
	return blobStoreConfig.HashBuckets
}

func (blobStoreConfig TomlV1) GetBlobCompression() interfaces.IOWrapper {
	return &blobStoreConfig.CompressionType
}

func (blobStoreConfig TomlV1) GetBlobEncryption() domain_interfaces.MarklId {
	return blobStoreConfig.Encryption
}

func (blobStoreConfig TomlV1) GetLockInternalFiles() bool {
	return blobStoreConfig.LockInternalFiles
}

func (blobStoreConfig TomlV1) SupportsMultiHash() bool {
	return true
}

func (blobStoreConfig TomlV1) GetDefaultHashTypeId() string {
	return string(blobStoreConfig.HashTypeId)
}

func (blobStoreConfig *TomlV1) setBasePath(value string) {
	blobStoreConfig.BasePath = value
}

func (blobStoreConfig TomlV1) Upgrade() (Config, ids.TypeStruct) {
	upgraded := &TomlV2{
		HashBuckets:       blobStoreConfig.HashBuckets,
		BasePath:          blobStoreConfig.BasePath,
		HashTypeId:        HashTypeSha256,
		CompressionType:   blobStoreConfig.CompressionType,
		LockInternalFiles: blobStoreConfig.LockInternalFiles,
	}

	upgraded.Encryption.ResetWithMarklId(blobStoreConfig.Encryption)

	return upgraded, ids.GetOrPanic(ids.TypeTomlBlobStoreConfigV2).TypeStruct
}
