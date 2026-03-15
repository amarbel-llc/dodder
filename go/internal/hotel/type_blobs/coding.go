package type_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type TypedBlob = hyphence.TypedBlob[Blob]

var CoderToTypedBlob = hyphence.CoderToTypedBlob[Blob]{
	Metadata: hyphence.TypedMetadataCoder[Blob]{},
	Blob: hyphence.CoderTypeMapWithoutType[Blob](
		map[string]interfaces.CoderBufferedReadWriter[*Blob]{
			ids.TypeTomlTypeV0: hyphence.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlV0{}
				},
				IgnoreDecodeErrors: true,
			},
			ids.TypeTomlTypeV1: hyphence.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlV1{}
				},
				IgnoreDecodeErrors: true,
			},
		},
	),
}
