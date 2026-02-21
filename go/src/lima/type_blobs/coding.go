package type_blobs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

type TypedBlob = triple_hyphen_io.TypedBlob[Blob]

var CoderToTypedBlob = triple_hyphen_io.CoderToTypedBlob[Blob]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Blob]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Blob](
		map[string]interfaces.CoderBufferedReadWriter[*Blob]{
			ids.TypeTomlTypeV0: triple_hyphen_io.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlV0{}
				},
			},
			ids.TypeTomlTypeV1: triple_hyphen_io.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlV1{}
				},
			},
		},
	),
}
