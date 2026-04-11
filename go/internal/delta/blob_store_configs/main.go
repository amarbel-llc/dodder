package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	charlie_bsc "code.linenisgreat.com/dodder/go/internal/charlie/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

// Re-export all types from charlie/blob_store_configs
type (
	Config                      = charlie_bsc.Config
	ConfigUpgradeable           = charlie_bsc.ConfigUpgradeable
	ConfigMutable               = charlie_bsc.ConfigMutable
	ConfigHashType              = charlie_bsc.ConfigHashType
	ConfigLocalMutable          = charlie_bsc.ConfigLocalMutable
	ConfigLocalHashBucketed     = charlie_bsc.ConfigLocalHashBucketed
	ConfigInventoryArchive      = charlie_bsc.ConfigInventoryArchive
	DeltaConfigImmutable        = charlie_bsc.DeltaConfigImmutable
	SignatureConfigImmutable    = charlie_bsc.SignatureConfigImmutable
	SelectorConfigImmutable     = charlie_bsc.SelectorConfigImmutable
	ConfigInventoryArchiveDelta = charlie_bsc.ConfigInventoryArchiveDelta
	ConfigPointer               = charlie_bsc.ConfigPointer
	ConfigSFTPRemotePath        = charlie_bsc.ConfigSFTPRemotePath
	ConfigSFTPUri               = charlie_bsc.ConfigSFTPUri
	ConfigSFTPConfigExplicit    = charlie_bsc.ConfigSFTPConfigExplicit
	ErrUnsupportedHashType      = charlie_bsc.ErrUnsupportedHashType
	HashType                    = charlie_bsc.HashType
	EncryptionKeys              = charlie_bsc.EncryptionKeys
	SignatureConfig             = charlie_bsc.SignatureConfig
	SelectorConfig              = charlie_bsc.SelectorConfig
	DeltaConfig                 = charlie_bsc.DeltaConfig
	TomlLocalHashBucketedV1     = charlie_bsc.TomlLocalHashBucketedV1
	TomlLocalHashBucketedV2     = charlie_bsc.TomlLocalHashBucketedV2
	TomlV3                      = charlie_bsc.TomlV3
	TomlSFTPV0                  = charlie_bsc.TomlSFTPV0
	TomlSFTPViaSSHConfigV0      = charlie_bsc.TomlSFTPViaSSHConfigV0
	TomlPointerV0               = charlie_bsc.TomlPointerV0
	TomlUriV0                   = charlie_bsc.TomlUriV0
	TomlInventoryArchiveV0      = charlie_bsc.TomlInventoryArchiveV0
	TomlInventoryArchiveV1      = charlie_bsc.TomlInventoryArchiveV1
	TomlInventoryArchiveV2      = charlie_bsc.TomlInventoryArchiveV2
	TypedConfig                 = hyphence.TypedBlob[Config]
	TypedMutableConfig          = hyphence.TypedBlob[ConfigMutable]
)

// Re-export constants
const (
	HashTypeSha256     = charlie_bsc.HashTypeSha256
	HashTypeBlake2b256 = charlie_bsc.HashTypeBlake2b256
	HashTypeDefault    = charlie_bsc.HashTypeDefault
	DefaultHashTypeId  = charlie_bsc.DefaultHashTypeId
)

var (
	DefaultHashType    = charlie_bsc.DefaultHashType
	DefaultHashBuckets = charlie_bsc.DefaultHashBuckets
	ConfigKeyValues    = charlie_bsc.ConfigKeyValues
	ConfigKeyNames     = charlie_bsc.ConfigKeyNames
)

// Re-export generated Decode/Encode functions
var (
	DecodeTomlLocalHashBucketedV1 = charlie_bsc.DecodeTomlLocalHashBucketedV1
	DecodeTomlLocalHashBucketedV2 = charlie_bsc.DecodeTomlLocalHashBucketedV2
	DecodeTomlV3                  = charlie_bsc.DecodeTomlV3
	DecodeTomlSFTPV0              = charlie_bsc.DecodeTomlSFTPV0
	DecodeTomlSFTPViaSSHConfigV0  = charlie_bsc.DecodeTomlSFTPViaSSHConfigV0
	DecodeTomlPointerV0           = charlie_bsc.DecodeTomlPointerV0
	DecodeTomlUriV0               = charlie_bsc.DecodeTomlUriV0
	DecodeTomlInventoryArchiveV0  = charlie_bsc.DecodeTomlInventoryArchiveV0
	DecodeTomlInventoryArchiveV1  = charlie_bsc.DecodeTomlInventoryArchiveV1
	DecodeTomlInventoryArchiveV2  = charlie_bsc.DecodeTomlInventoryArchiveV2
)

// Interface satisfaction checks
var (
	_ ConfigSFTPRemotePath        = &TomlSFTPV0{}
	_ ConfigSFTPRemotePath        = &TomlSFTPViaSSHConfigV0{}
	_ ConfigMutable               = &TomlSFTPV0{}
	_ ConfigLocalHashBucketed     = TomlLocalHashBucketedV1{}
	_ ConfigUpgradeable           = TomlLocalHashBucketedV1{}
	_ ConfigLocalMutable          = &TomlLocalHashBucketedV1{}
	_ ConfigLocalHashBucketed     = TomlLocalHashBucketedV2{}
	_ ConfigUpgradeable           = TomlLocalHashBucketedV2{}
	_ ConfigLocalMutable          = &TomlLocalHashBucketedV2{}
	_ ConfigLocalHashBucketed     = TomlV3{}
	_ ConfigLocalMutable          = &TomlV3{}
	_ ConfigMutable               = &TomlV3{}
	_ ConfigPointer               = TomlPointerV0{}
	_ ConfigMutable               = &TomlPointerV0{}
	_ ConfigInventoryArchive      = TomlInventoryArchiveV0{}
	_ ConfigUpgradeable           = TomlInventoryArchiveV0{}
	_ ConfigMutable               = &TomlInventoryArchiveV0{}
	_ ConfigInventoryArchiveDelta = TomlInventoryArchiveV1{}
	_ ConfigUpgradeable           = TomlInventoryArchiveV1{}
	_ ConfigMutable               = &TomlInventoryArchiveV1{}
	_ SignatureConfigImmutable    = TomlInventoryArchiveV1{}
	_ SelectorConfigImmutable     = TomlInventoryArchiveV1{}
	_ ConfigInventoryArchiveDelta = TomlInventoryArchiveV2{}
	_ ConfigMutable               = &TomlInventoryArchiveV2{}
	_ SignatureConfigImmutable    = TomlInventoryArchiveV2{}
	_ SelectorConfigImmutable     = TomlInventoryArchiveV2{}
	_ ConfigSFTPRemotePath        = TomlSFTPViaSSHConfigV0{}
	_ ConfigMutable               = &TomlSFTPViaSSHConfigV0{}
)

type DefaultType = TomlV3

func Default() *TypedMutableConfig {
	return &TypedMutableConfig{
		Type: ids.GetOrPanic(ids.TypeTomlBlobStoreConfigVCurrent).TypeStruct,
		Blob: &DefaultType{
			HashBuckets:       DefaultHashBuckets,
			HashTypeId:        HashTypeDefault,
			CompressionType:   compression_type.CompressionTypeDefault,
			LockInternalFiles: true,
		},
	}
}
