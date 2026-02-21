package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/bravo/values"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

const DefaultHashTypeId = markl.FormatIdHashSha256

var DefaultHashType markl.FormatHash = markl.FormatHashSha256

type (
	Config = interface {
		GetBlobStoreType() string
	}

	ConfigUpgradeable interface {
		Config
		Upgrade() (Config, ids.TypeStruct)
	}

	ConfigMutable interface {
		Config
		interfaces.CommandComponentWriter
	}

	ConfigHashType interface {
		SupportsMultiHash() bool
		GetDefaultHashTypeId() string
	}

	configLocal interface {
		Config
		getBasePath() string
	}

	configLocalMutable interface {
		configLocal
		setBasePath(string)
	}

	ConfigLocalMutable interface {
		configLocalMutable
	}

	ConfigLocalHashBucketed interface {
		configLocal
		ConfigHashType
		domain_interfaces.BlobIOWrapper
		GetHashBuckets() []int
		GetLockInternalFiles() bool
	}

	ConfigInventoryArchive interface {
		configLocal
		ConfigHashType
		domain_interfaces.BlobIOWrapper
		GetLooseBlobStoreId() blob_store_id.Id
		GetCompressionType() compression_type.CompressionType
	}

	DeltaConfigImmutable interface {
		GetDeltaEnabled() bool
		GetDeltaAlgorithm() string
		GetDeltaMinBlobSize() uint64
		GetDeltaMaxBlobSize() uint64
		GetDeltaSizeRatio() float64
	}

	ConfigInventoryArchiveDelta interface {
		ConfigInventoryArchive
		DeltaConfigImmutable
	}

	ConfigPointer interface {
		Config
		GetPath() directory_layout.BlobStorePath
	}

	ConfigSFTPRemotePath interface {
		Config
		GetRemotePath() string
	}

	ConfigSFTPUri interface {
		ConfigSFTPRemotePath

		GetUri() values.Uri
	}

	ConfigSFTPConfigExplicit interface {
		ConfigSFTPRemotePath

		GetHost() string
		GetPort() int
		GetUser() string
		GetPassword() string
		GetPrivateKeyPath() string
	}

	TypedConfig        = triple_hyphen_io.TypedBlob[Config]
	TypedMutableConfig = triple_hyphen_io.TypedBlob[ConfigMutable]
)

var (
	_ ConfigSFTPRemotePath = &TomlSFTPV0{}
	_ ConfigSFTPRemotePath = &TomlSFTPViaSSHConfigV0{}
	_ ConfigMutable        = &TomlV0{}
	_ ConfigMutable        = &TomlSFTPV0{}
)

var DefaultHashBuckets []int = []int{2}

type DefaultType = TomlV2

func Default() *TypedMutableConfig {
	return &TypedMutableConfig{
		Type: ids.GetOrPanic(ids.TypeTomlBlobStoreConfigVCurrent).TypeStruct,
		Blob: &DefaultType{
			HashBuckets:       DefaultHashBuckets,
			HashTypeId:        markl.FormatIdHashBlake2b256,
			CompressionType:   compression_type.CompressionTypeDefault,
			LockInternalFiles: true,
		},
	}
}
