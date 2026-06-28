package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	workspace_config_value_blobs "code.linenisgreat.com/dodder/go/internal/delta/workspace_config_value_blobs"
	"github.com/amarbel-llc/hyphence/go/hyphence"
	mad_ids "github.com/amarbel-llc/madder/go/pkgs/ids"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Config]{
	Metadata: hyphence.TypedMetadataCoder[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Config]{},
	Blob: hyphence.CoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlWorkspaceConfigV0: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := workspace_config_value_blobs.DecodeV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := workspace_config_value_blobs.DecodeV0(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *V0:
						*doc.Data() = *v
					case V0:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlWorkspaceConfigV1: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := workspace_config_value_blobs.DecodeV1(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := workspace_config_value_blobs.DecodeV1(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *V1:
						*doc.Data() = *v
					case V1:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlWorkspaceConfigV2: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := workspace_config_value_blobs.DecodeV2(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := workspace_config_value_blobs.DecodeV2(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *V2:
						*doc.Data() = *v
					case V2:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlWorkspaceConfigV3: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := workspace_config_value_blobs.DecodeV3(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := workspace_config_value_blobs.DecodeV3(nil)
					if err != nil {
						return nil, err
					}
					switch v := cfg.(type) {
					case *V3:
						*doc.Data() = *v
					case V3:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
		},
	),
}
