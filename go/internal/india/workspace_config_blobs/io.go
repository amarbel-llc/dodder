package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var Coder = triple_hyphen_io.CoderToTypedBlob[Config]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Config]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlWorkspaceConfigV0: triple_hyphen_io.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &V0{}
				},
			},
		},
	),
}
