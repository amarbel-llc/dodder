package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type TypedBlob = hyphence.TypedBlob[Blob]

var Coder = hyphence.CoderToTypedBlob[Blob]{
	Metadata: hyphence.TypedMetadataCoder[Blob]{},
	Blob: hyphence.CoderTypeMapWithoutType[Blob](
		map[string]interfaces.CoderBufferedReadWriter[*Blob]{
			ids.TypeTomlRepoLocalOverridePath: hyphence.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlLocalOverridePathV0{}
				},
			},
			ids.TypeTomlRepoDotenvXdgV0: hyphence.CoderToml[
				Blob,
				*Blob,
			]{
				Progenitor: func() Blob {
					return &TomlXDGV0{}
				},
			},
			ids.TypeTomlRepoUri: hyphence.CoderToml[
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
