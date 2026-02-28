package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var CoderPrivate = triple_hyphen_io.CoderToTypedBlob[ConfigPrivate]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[ConfigPrivate]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[ConfigPrivate](
		map[string]interfaces.CoderBufferedReadWriter[*ConfigPrivate]{
			ids.TypeTomlConfigImmutableV2: triple_hyphen_io.CoderToml[
				ConfigPrivate,
				*ConfigPrivate,
			]{
				Progenitor: func() ConfigPrivate {
					return &TomlV2Private{}
				},
			},
			ids.TypeTomlConfigImmutableV1: triple_hyphen_io.CoderToml[
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

var CoderPublic = triple_hyphen_io.CoderToTypedBlob[ConfigPublic]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[ConfigPublic]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[ConfigPublic](
		map[string]interfaces.CoderBufferedReadWriter[*ConfigPublic]{
			ids.TypeTomlConfigImmutableV2: triple_hyphen_io.CoderToml[
				ConfigPublic,
				*ConfigPublic,
			]{
				Progenitor: func() ConfigPublic {
					return &TomlV2Public{}
				},
			},
			ids.TypeTomlConfigImmutableV1: triple_hyphen_io.CoderToml[
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
