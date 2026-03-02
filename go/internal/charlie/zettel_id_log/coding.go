package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var Coder = triple_hyphen_io.CoderToTypedBlob[Entry]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Entry]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Entry](
		map[string]interfaces.CoderBufferedReadWriter[*Entry]{
			ids.TypeZettelIdLogV1: triple_hyphen_io.CoderToml[
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
