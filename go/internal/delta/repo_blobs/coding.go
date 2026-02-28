package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type TypedBlob = triple_hyphen_io.TypedBlob[Blob]

var Coder = triple_hyphen_io.CoderToTypedBlob[Blob]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Blob]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Blob](
		map[string]interfaces.CoderBufferedReadWriter[*Blob]{
			ids.TypeTomlRepoLocalOverridePath: triple_hyphen_io.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlLocalOverridePathV0{}
				},
			},
			ids.TypeTomlRepoDotenvXdgV0: triple_hyphen_io.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlXDGV0{}
				},
			},
			ids.TypeTomlRepoUri: triple_hyphen_io.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlUriV0{}
				},
			},
		},
	),
}
