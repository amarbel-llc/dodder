package repo_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	charlie_rb "code.linenisgreat.com/dodder/go/internal/charlie/repo_blobs"
	"code.linenisgreat.com/dodder/go/lib/0/interfaces"
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
					doc, err := charlie_rb.DecodeTomlLocalOverridePathV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := charlie_rb.DecodeTomlLocalOverridePathV0(nil)
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
					doc, err := charlie_rb.DecodeTomlXDGV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := charlie_rb.DecodeTomlXDGV0(nil)
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
					doc, err := charlie_rb.DecodeTomlUriV0(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := charlie_rb.DecodeTomlUriV0(nil)
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
