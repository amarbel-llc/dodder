package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[Entry]{
	Metadata: hyphence.TypedMetadataCoder[Entry]{},
	Blob: hyphence.CoderTypeMapWithoutType[Entry](
		map[string]interfaces.CoderBufferedReadWriter[*Entry]{
			ids.TypeZettelIdLogV1: hyphence.CoderToml[
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
