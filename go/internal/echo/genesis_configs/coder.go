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
			ids.TypeTomlConfigImmutableV2: hyphence.CoderToml[
				ConfigPrivate,
				*ConfigPrivate,
			]{
				Progenitor: func() ConfigPrivate {
					return &TomlV2Private{}
				},
			},
			ids.TypeTomlConfigImmutableV1: hyphence.CoderToml[
				ConfigPrivate,
				*ConfigPrivate,
			]{
				Progenitor: func() ConfigPrivate {
					return &TomlV1Private{}
				},
			},
		},
	),
}

var CoderPublic = hyphence.CoderToTypedBlob[ConfigPublic]{
	Metadata: hyphence.TypedMetadataCoder[ConfigPublic]{},
	Blob: hyphence.CoderTypeMapWithoutType[ConfigPublic](
		map[string]interfaces.CoderBufferedReadWriter[*ConfigPublic]{
			ids.TypeTomlConfigImmutableV2: hyphence.CoderToml[
				ConfigPublic,
				*ConfigPublic,
			]{
				Progenitor: func() ConfigPublic {
					return &TomlV2Public{}
				},
			},
			ids.TypeTomlConfigImmutableV1: hyphence.CoderToml[
				ConfigPublic,
				*ConfigPublic,
			]{
				Progenitor: func() ConfigPublic {
					return &TomlV1Public{}
				},
			},
		},
	),
}
