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
// The uti-groups tables are seeded COMPLETELY (headers plus sorted
// key-values), not just as headers: each UTIGroup is itself a
// map[string]string, and the generated encoder rewrites those inner keys via
// DeleteAllValues + per-map-iteration SetAny — which would scramble the
// inner-key order even inside a seeded table. The V2 Encode wrapper in
// CoderToTypedBlob hides UTIGroups from the generated encoder so the seeded
// key-values survive verbatim. Formatter tables only need headers: their
// values are struct-backed (script_config.WithOutputFormat), encoded in
// fixed field order.
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

	writeSortedUTIGroupTables(&sb, blob.UTIGroups)
	writeSortedTableHeaders(&sb, "formatters", blob.Formatters)

	return []byte(sb.String())
}

// writeSortedUTIGroupTables seeds each [uti-groups.X] table with its
// key-values in sorted key order. The values are plain strings (formatter
// names), rendered as TOML basic strings.
func writeSortedUTIGroupTables(
	sb *strings.Builder,
	groups map[string]UTIGroup,
) {
	for _, groupName := range sortedKeys(groups) {
		if sb.Len() == 0 {
			sb.WriteString("\n")
		}

		fmt.Fprintf(sb, "[uti-groups.%s]\n", cst.QuoteKey(groupName))

		group := groups[groupName]

		for _, key := range sortedKeys(group) {
			fmt.Fprintf(
				sb,
				"%s = \"%s\"\n",
				cst.QuoteKey(key),
				cst.EscapeString(group[key]),
			)
		}
	}
}

func writeSortedTableHeaders[V any](
	sb *strings.Builder,
	prefix string,
	tables map[string]V,
) {
	for _, key := range sortedKeys(tables) {
		if sb.Len() == 0 {
			sb.WriteString("\n")
		}

		fmt.Fprintf(sb, "[%s.%s]\n", prefix, cst.QuoteKey(key))
	}
}

func sortedKeys[V any](tables map[string]V) []string {
	keys := make([]string, 0, len(tables))

	for key := range tables {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
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
					// Seed the document with the map-backed sub-tables in
					// sorted order so encode output is deterministic (see
					// tomlV2EncodeSkeleton).
					var skeleton []byte
					v, isV2 := blob.(*TomlV2)
					if isV2 {
						skeleton = tomlV2EncodeSkeleton(v)
					}
					doc, err := golf_tb.DecodeTomlV2(skeleton)
					if err != nil {
						return nil, err
					}
					if isV2 {
						data := *v
						// The uti-groups tables were seeded completely
						// (headers plus sorted key-values) by the skeleton;
						// hide them from the generated encoder, whose
						// DeleteAllValues + map-iteration SetAny would
						// re-add the inner keys in random order (tommy#139
						// one level deeper). The encoder leaves the seeded
						// CST tables untouched when UTIGroups is empty.
						data.UTIGroups = nil
						*doc.Data() = data
					}
					return doc.Encode()
				},
			},
		},
	),
}
