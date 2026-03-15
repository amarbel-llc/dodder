package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[ConfigOverlay]{
	Metadata: hyphence.TypedMetadataCoder[ConfigOverlay]{},
	Blob: hyphence.CoderTypeMapWithoutType[ConfigOverlay](
		map[string]interfaces.CoderBufferedReadWriter[*ConfigOverlay]{
			ids.TypeTomlConfigV0: hyphence.CoderToml[
				ConfigOverlay,
				*ConfigOverlay,
			]{
				Progenitor: func() ConfigOverlay {
					return &V0{}
				},
			},
			ids.TypeTomlConfigV1: hyphence.CoderToml[
				ConfigOverlay,
				*ConfigOverlay,
			]{
				Progenitor: func() ConfigOverlay {
					return &V1{}
				},
			},
			ids.TypeTomlConfigV2: hyphence.CoderToml[
				ConfigOverlay,
				*ConfigOverlay,
			]{
				Progenitor: func() ConfigOverlay {
					return &V2{}
				},
			},
		},
	),
}
