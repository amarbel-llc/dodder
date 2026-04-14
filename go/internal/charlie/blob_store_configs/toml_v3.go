package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/compression_type"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/values"
)

//go:generate tommy generate
type TomlV3 struct {
	HashBuckets values.IntSlice `toml:"hash_buckets"`
	BasePath    string          `toml:"base_path,omitempty"`
	HashTypeId  HashType        `toml:"hash_type-id"`

	Encryption []markl.Id `toml:"encryption"`

	CompressionType   compression_type.CompressionType `toml:"compression-type"`
	LockInternalFiles bool                             `toml:"lock-internal-files"`
}

func (TomlV3) GetBlobStoreType() string {
	return "local"
}

func (blobStoreConfig *TomlV3) SetFlagDefinitions(
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

	setMultiEncryptionFlagDefinition(flagSet, &blobStoreConfig.Encryption)

	flagSet.BoolVar(
		&blobStoreConfig.LockInternalFiles,
		"lock-internal-files",
		blobStoreConfig.LockInternalFiles,
		"",
	)
}

func (blobStoreConfig TomlV3) getBasePath() string {
	return blobStoreConfig.BasePath
}

func (blobStoreConfig TomlV3) GetHashBuckets() []int {
	return blobStoreConfig.HashBuckets
}

func (blobStoreConfig TomlV3) GetBlobCompression() interfaces.IOWrapper {
	return &blobStoreConfig.CompressionType
}

func (blobStoreConfig TomlV3) GetBlobEncryption() domain_interfaces.MarklId {
	return EncryptionKeys(blobStoreConfig.Encryption)
}

func (blobStoreConfig TomlV3) GetLockInternalFiles() bool {
	return blobStoreConfig.LockInternalFiles
}

func (blobStoreConfig TomlV3) SupportsMultiHash() bool {
	return true
}

func (blobStoreConfig TomlV3) GetDefaultHashTypeId() string {
	return string(blobStoreConfig.HashTypeId)
}

func (blobStoreConfig *TomlV3) setBasePath(value string) {
	blobStoreConfig.BasePath = value
}
