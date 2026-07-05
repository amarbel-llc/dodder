package type_blobs

import (
	"fmt"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	golf_tb "code.linenisgreat.com/dodder/go/internal/alfa/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/tommy/pkg/cst"
)

type TypedBlob = hyphence.TypedBlob[Blob]

// tomlV2EncodeSkeleton returns a TOML skeleton pre-seeding blob's map-backed
// sub-table headers ([uti-groups.X], [formatters.X]) in sorted key order,
// with a single leading blank line so root scalars render above the first
// header separated by a blank, as a from-scratch encode would place them.
// The headers themselves are back-to-back: a blank line between headers
// would be parsed into the previous table's scope and end up interleaved
// with that table's key-values on encode.
//
// WORKAROUND for https://github.com/amarbel-llc/tommy/issues/139: tommy's
// generated encoders iterate Go maps directly, so a blob with more than one
// formatter serializes its sub-tables in random per-process order — giving
// every `dodder init` a different genesis type-blob digest for the same
// logical blob and breaking cross-repo dedup of the builtin types (pull and
// clone between fresh repos conflict on !md). Decoding this skeleton before
// encoding pins the order: the generated encoder's EnsureChildSubTable fills
// the seeded tables in place instead of appending in map-iteration order.
// Delete once tommy sorts map keys on encode.
func tomlV2EncodeSkeleton(blob *TomlV2) []byte {
	var sb strings.Builder

	writeSortedTableHeaders(&sb, "uti-groups", blob.UTIGroups)
	writeSortedTableHeaders(&sb, "formatters", blob.Formatters)

	return []byte(sb.String())
}

func writeSortedTableHeaders[V any](
	sb *strings.Builder,
	prefix string,
	tables map[string]V,
) {
	keys := make([]string, 0, len(tables))

	for key := range tables {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if sb.Len() == 0 {
			sb.WriteString("\n")
		}

		fmt.Fprintf(sb, "[%s.%s]\n", prefix, cst.QuoteKey(key))
	}
}

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
					// Seed the document with the map-backed sub-table
					// headers in sorted order so encode output is
					// deterministic (see tomlV2EncodeSkeleton).
					var skeleton []byte
					if v, ok := blob.(*TomlV2); ok {
						skeleton = tomlV2EncodeSkeleton(v)
					}
					doc, err := golf_tb.DecodeTomlV2(skeleton)
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
