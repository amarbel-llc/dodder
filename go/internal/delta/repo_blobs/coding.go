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
			ids.TypeTomlRepoLocalOverridePath: hyphence.CoderTommy[
				Blob,
				*Blob,
			]{
				Decode: func(b []byte) (Blob, error) {
					doc, err := DecodeTomlLocalOverridePathV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := DecodeTomlLocalOverridePathV0(nil)
					if err != nil {
						return nil, err
					}
					switch v := blob.(type) {
					case *TomlLocalOverridePathV0:
						*doc.Data() = *v
					case TomlLocalOverridePathV0:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlRepoDotenvXdgV0: hyphence.CoderTommy[
				Blob,
				*Blob,
			]{
				Decode: func(b []byte) (Blob, error) {
					doc, err := DecodeTomlXDGV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := DecodeTomlXDGV0(nil)
					if err != nil {
						return nil, err
					}
					switch v := blob.(type) {
					case *TomlXDGV0:
						*doc.Data() = *v
					case TomlXDGV0:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlRepoUri: hyphence.CoderTommy[
				Blob,
				*Blob,
			]{
				Decode: func(b []byte) (Blob, error) {
					doc, err := DecodeTomlUriV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := DecodeTomlUriV0(nil)
					if err != nil {
						return nil, err
					}
					switch v := blob.(type) {
					case *TomlUriV0:
						*doc.Data() = *v
					case TomlUriV0:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
		},
	),
}
