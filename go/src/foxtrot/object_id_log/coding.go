package object_id_log

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
)

var Coder = triple_hyphen_io.CoderToTypedBlob[Entry]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Entry]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Entry](
		map[string]interfaces.CoderBufferedReadWriter[*Entry]{
			ids.TypeObjectIdLogV1: triple_hyphen_io.CoderToml[
				Entry,
				*Entry,
			]{
				Progenitor: func() Entry {
					return &V1{}
				},
			},
		},
	),
}
