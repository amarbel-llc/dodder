package type_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type TypedBlob = hyphence.TypedBlob[Blob]

var CoderToTypedBlob = hyphence.CoderToTypedBlob[Blob]{
	Metadata: hyphence.TypedMetadataCoder[Blob]{},
	Blob: hyphence.CoderTypeMapWithoutType[Blob](
		map[string]interfaces.CoderBufferedReadWriter[*Blob]{
			ids.TypeTomlTypeV0: hyphence.CoderTommy[
				Blob,
				*Blob,
			]{
				Decode: func(b []byte) (Blob, error) {
					doc, err := DecodeTomlV0(b)
					if err != nil {
						return &TomlV0{}, nil
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := DecodeTomlV0(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := blob.(*TomlV0); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlTypeV1: hyphence.CoderTommy[
				Blob,
				*Blob,
			]{
				Decode: func(b []byte) (Blob, error) {
					doc, err := DecodeTomlV1(b)
					if err != nil {
						return &TomlV1{}, nil
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := DecodeTomlV1(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := blob.(*TomlV1); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
		},
	),
}
