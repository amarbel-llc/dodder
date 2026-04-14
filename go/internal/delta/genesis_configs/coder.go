package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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
					doc, err := DecodeTomlV2Private(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlV2Private); ok {
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
					doc, err := DecodeTomlV2Public(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigPublic) ([]byte, error) {
					doc, err := DecodeTomlV2Public(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := cfg.(*TomlV2Public); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
		},
	),
}
