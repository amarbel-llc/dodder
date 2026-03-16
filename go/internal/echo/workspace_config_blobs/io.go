package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[Config]{
	Metadata: hyphence.TypedMetadataCoder[Config]{},
	Blob: hyphence.CoderTypeMapWithoutType[Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlWorkspaceConfigV0: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &V0{}
				},
			},
			ids.TypeTomlWorkspaceConfigV1: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &V1{}
				},
			},
		},
	),
}
