package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

const DefaultHashTypeId = string(HashTypeSha256)

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
		GetMaxPackSize() uint64
	}

	DeltaConfigImmutable interface {
		GetDeltaEnabled() bool
		GetDeltaAlgorithm() string
		GetDeltaMinBlobSize() uint64
		GetDeltaMaxBlobSize() uint64
		GetDeltaSizeRatio() float64
	}

	SignatureConfigImmutable interface {
		GetSignatureType() string
		GetSignatureLen() int
		GetAvgChunkSize() int
		GetMinChunkSize() int
		GetMaxChunkSize() int
	}

	SelectorConfigImmutable interface {
		GetSelectorType() string
		GetSelectorBands() int
		GetSelectorRowsPerBand() int
		GetSelectorMinBlobSize() uint64
		GetSelectorMaxBlobSize() uint64
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
		GetKnownHostsFile() string
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
)

var DefaultHashBuckets []int = []int{2}
