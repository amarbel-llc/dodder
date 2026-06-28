package repo_blobs

import (
	charlie_rb "code.linenisgreat.com/dodder/go/internal/alfa/repo_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/hyphence/go/hyphence"
	mad_ids "github.com/amarbel-llc/madder/go/pkgs/ids"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type TypedBlob = hyphence.TypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Blob]

var Coder = hyphence.CoderToTypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Blob]{
	Metadata: hyphence.TypedMetadataCoder[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Blob]{},
	Blob: hyphence.CoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Blob](
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
