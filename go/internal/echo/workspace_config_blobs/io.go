package workspace_config_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[Config]{
	Metadata: hyphence.TypedMetadataCoder[Config]{},
	Blob: hyphence.CoderTypeMapWithoutType[Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlWorkspaceConfigV0: hyphence.CoderTommy[
				Config,
				*Config,
			]{
				Decode: func(b []byte) (Config, error) {
					doc, err := DecodeV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := DecodeV0(nil)
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
					doc, err := DecodeV1(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := DecodeV1(nil)
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
					doc, err := DecodeV2(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg Config) ([]byte, error) {
					doc, err := DecodeV2(nil)
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
		},
	),
}
