package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	charlie_rc "code.linenisgreat.com/dodder/go/internal/charlie/repo_configs"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[ConfigOverlay]{
	Metadata: hyphence.TypedMetadataCoder[ConfigOverlay]{},
	Blob: hyphence.CoderTypeMapWithoutType[ConfigOverlay](
		map[string]interfaces.CoderBufferedReadWriter[*ConfigOverlay]{
			ids.TypeTomlConfigV0: hyphence.CoderTommy[
				ConfigOverlay,
				*ConfigOverlay,
			]{
				Decode: func(b []byte) (ConfigOverlay, error) {
					doc, err := charlie_rc.DecodeV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigOverlay) ([]byte, error) {
					doc, err := charlie_rc.DecodeV0(nil)
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
			ids.TypeTomlConfigV1: hyphence.CoderTommy[
				ConfigOverlay,
				*ConfigOverlay,
			]{
				Decode: func(b []byte) (ConfigOverlay, error) {
					doc, err := charlie_rc.DecodeV1(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigOverlay) ([]byte, error) {
					doc, err := charlie_rc.DecodeV1(nil)
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
			ids.TypeTomlConfigV2: hyphence.CoderTommy[
				ConfigOverlay,
				*ConfigOverlay,
			]{
				Decode: func(b []byte) (ConfigOverlay, error) {
					doc, err := charlie_rc.DecodeV2(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(cfg ConfigOverlay) ([]byte, error) {
					doc, err := charlie_rc.DecodeV2(nil)
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
