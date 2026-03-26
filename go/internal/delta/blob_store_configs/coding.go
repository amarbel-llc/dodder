package blob_store_configs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func registerTommy(
	typeMap hyphence.CoderTypeMapWithoutType[Config],
	typeString string,
	decode func([]byte) (Config, error),
	encode func(Config) ([]byte, error),
) struct{} {
	if existing, ok := typeMap[typeString]; ok {
		panic(
			fmt.Sprintf(
				"coder for type %q registered more than once! first registration: %#v",
				typeString,
				existing,
			),
		)
	}

	typeMap[typeString] = hyphence.CoderTommy[
		Config,
		*Config,
	]{
		Decode: decode,
		Encode: encode,
	}

	return struct{}{}
}

var Coder = hyphence.CoderToTypedBlob[Config]{
	Metadata: hyphence.TypedMetadataCoder[Config]{},
	Blob: hyphence.CoderTypeMapWithoutType[Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlBlobStoreConfigV1: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := DecodeTomlLocalHashBucketedV1(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := DecodeTomlLocalHashBucketedV1(nil)
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
					doc, err := DecodeTomlLocalHashBucketedV2(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := DecodeTomlLocalHashBucketedV2(nil)
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
					doc, err := DecodeTomlV3(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := DecodeTomlV3(nil)
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
					doc, err := DecodeTomlSFTPV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := DecodeTomlSFTPV0(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlSFTPV0); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlBlobStoreConfigSftpViaSSHConfigV0: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlSFTPViaSSHConfigV0{}
				},
			},
		},
	),
}
