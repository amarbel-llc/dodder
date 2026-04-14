package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	charlie_zil "code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

var Coder = hyphence.CoderToTypedBlob[Entry]{
	Metadata: hyphence.TypedMetadataCoder[Entry]{},
	Blob: hyphence.CoderTypeMapWithoutType[Entry](
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
