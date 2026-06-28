package genesis_configs

import (
	genesis_config_blobs "code.linenisgreat.com/dodder/go/internal/bravo/genesis_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/hyphence/go/hyphence"
	mad_ids "github.com/amarbel-llc/madder/go/pkgs/ids"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

var CoderPrivate = hyphence.CoderToTypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, ConfigPrivate]{
	Metadata: hyphence.TypedMetadataCoder[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, ConfigPrivate]{},
	Blob: hyphence.CoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, ConfigPrivate](
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
		},
	),
}

var CoderPublic = hyphence.CoderToTypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, ConfigPublic]{
	Metadata: hyphence.TypedMetadataCoder[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, ConfigPublic]{},
	Blob: hyphence.CoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, ConfigPublic](
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
		},
	),
}
