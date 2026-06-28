package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	charlie_zil "code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"github.com/amarbel-llc/hyphence/go/hyphence"
	mad_ids "github.com/amarbel-llc/madder/go/pkgs/ids"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Entry]{
	Metadata: hyphence.TypedMetadataCoder[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Entry]{},
	Blob: hyphence.CoderTypeMapWithoutType[mad_ids.TypeStruct, *mad_ids.TypeStruct, markl.Id, *markl.Id, Entry](
		map[string]interfaces.CoderBufferedReadWriter[*Entry]{
			ids.TypeZettelIdLogV1: hyphence.CoderTommy[
				Entry,
				*Entry,
			]{
				Decode: func(b []byte) (Entry, error) {
					doc, err := charlie_zil.DecodeV1(b)
					if err != nil {
						return nil, err
					}
					return doc.Data(), nil
				},
				Encode: func(entry Entry) ([]byte, error) {
					doc, err := charlie_zil.DecodeV1(nil)
					if err != nil {
						return nil, err
					}
					switch v := entry.(type) {
					case *V1:
						*doc.Data() = *v
					case V1:
						*doc.Data() = v
					}
					return doc.Encode()
				},
			},
		},
	),
}
