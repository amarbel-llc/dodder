package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	charlie_bsc "code.linenisgreat.com/dodder/go/internal/charlie/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[Config]{
	Metadata: hyphence.TypedMetadataCoder[Config]{},
	Blob: hyphence.CoderTypeMapWithoutType[Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlBlobStoreConfigV1: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlLocalHashBucketedV1(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlLocalHashBucketedV1(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *TomlLocalHashBucketedV1:
						*doc.Data() = *v
					case TomlLocalHashBucketedV1:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigV2: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlLocalHashBucketedV2(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlLocalHashBucketedV2(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *TomlLocalHashBucketedV2:
						*doc.Data() = *v
					case TomlLocalHashBucketedV2:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigV3: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlV3(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlV3(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *TomlV3:
						*doc.Data() = *v
					case TomlV3:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigSftpExplicitV0: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlSFTPV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlSFTPV0(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlSFTPV0); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigSftpViaSSHConfigV0: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlSFTPViaSSHConfigV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlSFTPViaSSHConfigV0(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlSFTPViaSSHConfigV0); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigPointerV0: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlPointerV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlPointerV0(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *TomlPointerV0:
						*doc.Data() = *v
					case TomlPointerV0:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigInventoryArchiveV0: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlInventoryArchiveV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlInventoryArchiveV0(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *TomlInventoryArchiveV0:
						*doc.Data() = *v
					case TomlInventoryArchiveV0:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigInventoryArchiveV1: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlInventoryArchiveV1(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlInventoryArchiveV1(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *TomlInventoryArchiveV1:
						*doc.Data() = *v
					case TomlInventoryArchiveV1:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigInventoryArchiveV2: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := charlie_bsc.DecodeTomlInventoryArchiveV2(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := charlie_bsc.DecodeTomlInventoryArchiveV2(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *TomlInventoryArchiveV2:
						*doc.Data() = *v
					case TomlInventoryArchiveV2:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
		},
	),
}
