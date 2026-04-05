package type_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	golf_tb "code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
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
					doc, err := golf_tb.DecodeTomlV0(b)
					if err != nil {
						return &TomlV0{}, nil
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := golf_tb.DecodeTomlV0(nil)
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
					doc, err := golf_tb.DecodeTomlV1(b)
					if err != nil {
						return &TomlV1{}, nil
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := golf_tb.DecodeTomlV1(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := blob.(*TomlV1); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
			ids.TypeTomlTypeV2: hyphence.CoderTommy[
				Blob,
				*Blob,
			]{
				Decode: func(b []byte) (Blob, error) {
					doc, err := golf_tb.DecodeTomlV2(b)
					if err != nil {
						return &TomlV2{}, nil
					}
					return doc.Data(), nil
				},
				Encode: func(blob Blob) ([]byte, error) {
					doc, err := golf_tb.DecodeTomlV2(nil)
					if err != nil {
						return nil, err
					}
					if v, ok := blob.(*TomlV2); ok {
						*doc.Data() = *v
					}
					return doc.Encode()
				},
			},
		},
	),
}
