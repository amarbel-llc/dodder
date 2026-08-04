package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	genesis_config_blobs "code.linenisgreat.com/dodder/go/internal/bravo/genesis_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

var CoderPrivate = hyphence.CoderToTypedBlob[ConfigPrivate]{
	Metadata: hyphence.TypedMetadataCoder[ConfigPrivate]{},
	Blob: hyphence.CoderTypeMapWithoutType[ConfigPrivate](
		map[string]interfaces.CoderBufferedReadWriter[*ConfigPrivate]{
			ids.TypeTomlConfigImmutableV2: hyphence.CoderTommy[
				ConfigPrivate,
				*ConfigPrivate,
			]{
				Decode: func(b []byte) (ConfigPrivate, error) {
					doc, err := genesis_config_blobs.DecodeTomlV2Private(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigPrivate) ([]byte, error) {
					doc, err := genesis_config_blobs.DecodeTomlV2Private(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlV2Private); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlConfigImmutableV3: hyphence.CoderTommy[
				ConfigPrivate,
				*ConfigPrivate,
			]{
				Decode: func(b []byte) (ConfigPrivate, error) {
					doc, err := genesis_config_blobs.DecodeTomlV3Private(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigPrivate) ([]byte, error) {
					doc, err := genesis_config_blobs.DecodeTomlV3Private(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlV3Private); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
		},
	),
}

var CoderPublic = hyphence.CoderToTypedBlob[ConfigPublic]{
	Metadata: hyphence.TypedMetadataCoder[ConfigPublic]{},
	Blob: hyphence.CoderTypeMapWithoutType[ConfigPublic](
		map[string]interfaces.CoderBufferedReadWriter[*ConfigPublic]{
			ids.TypeTomlConfigImmutableV2: hyphence.CoderTommy[
				ConfigPublic,
				*ConfigPublic,
			]{
				Decode: func(b []byte) (ConfigPublic, error) {
					doc, err := genesis_config_blobs.DecodeTomlV2Public(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigPublic) ([]byte, error) {
					doc, err := genesis_config_blobs.DecodeTomlV2Public(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlV2Public); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlConfigImmutableV3: hyphence.CoderTommy[
				ConfigPublic,
				*ConfigPublic,
			]{
				Decode: func(b []byte) (ConfigPublic, error) {
					doc, err := genesis_config_blobs.DecodeTomlV3Public(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigPublic) ([]byte, error) {
					doc, err := genesis_config_blobs.DecodeTomlV3Public(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlV3Public); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
		},
	),
}
