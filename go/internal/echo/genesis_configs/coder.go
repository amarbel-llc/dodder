package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
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
					doc, err := DecodeTomlV2Private(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigPrivate) ([]byte, error) {
					concrete := cfg.(*TomlV2Private)
					doc, err := DecodeTomlV2Private([]byte{})
					if err != nil {
						return nil, err
					}
					*doc.Data() = *concrete
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
					doc, err := DecodeTomlV2Public(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigPublic) ([]byte, error) {
					concrete := cfg.(*TomlV2Public)
					doc, err := DecodeTomlV2Public([]byte{})
					if err != nil {
						return nil, err
					}
					*doc.Data() = *concrete
					return doc.Encode()
				},
			},
		},
	),
}
