//go:build test

package type_blobs

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// TestTomlV2EncodeDeterministic pins the workaround for
// https://github.com/amarbel-llc/tommy/issues/139: tommy's generated encoders
// iterate Go maps directly, so without the sorted sub-table skeleton
// (tomlV2EncodeSkeleton) a multi-formatter blob would serialize its
// [formatters.*] tables in random per-process order — giving every
// `dodder init` a different genesis type-blob digest for the same logical
// blob. Repeated encodes must be byte-identical and sorted by formatter name.
func TestTomlV2EncodeDeterministic(t1 *testing.T) {
	t := ui.MakeT(t1)

	encode := func() string {
		blob := DefaultWithPandocFormatter()

		typedBlob := TypedBlob{
			Type: ids.MustTypeStruct(ids.TypeTomlTypeV2).ToMadder(),
			Blob: &blob,
		}

		var buf bytes.Buffer
		bufferedWriter := bufio.NewWriter(&buf)

		if _, err := CoderToTypedBlob.Blob.EncodeTo(
			&typedBlob,
			bufferedWriter,
		); err != nil {
			t.Fatalf("encode failed: %s", err)
		}

		if err := bufferedWriter.Flush(); err != nil {
			t.Fatalf("flush failed: %s", err)
		}

		return buf.String()
	}

	first := encode()

	// 20 re-encodes: with six formatter keys and four multi-key uti-groups,
	// unsorted map iteration would produce a differing order with
	// near-certainty. This covers both levels of tommy#139: sub-table order
	// AND the inner key order of the map-valued uti-groups.
	for range 20 {
		t.AssertEqualStrings(first, encode())
	}

	// The uti-groups tables (with sorted inner keys) must precede the
	// formatter tables, and both must appear in sorted key order.
	expectedOrder := []string{
		"[uti-groups.default]",
		`"public.html" = "html"`,
		`"public.utf8-plain-text" = "text"`,
		"[uti-groups.gdoc]",
		"[uti-groups.pdf]",
		`"com.adobe.pdf" = "pdf-beamer"`,
		"[uti-groups.text-render]",
		"[formatters.html]",
		"[formatters.html-gdoc]",
		"[formatters.html-partial]",
		"[formatters.pdf-beamer]",
		"[formatters.text]",
		"[formatters.text-render]",
	}

	previousIndex := -1

	for _, line := range expectedOrder {
		index := strings.Index(first, line+"\n")

		if index < 0 {
			t.Fatalf("missing %q in encoded blob:\n%s", line, first)
		}

		if index <= previousIndex {
			t.Errorf("%q out of sorted order in encoded blob:\n%s", line, first)
		}

		previousIndex = index
	}
}

// TestTomlV2UTIGroupsRoundTrip pins the encode-side UTIGroups hiding trick:
// the V2 Encode wrapper seeds uti-groups entirely via the skeleton CST and
// nils UTIGroups on the data copy, so decode must still recover the full
// group set from the encoded bytes.
func TestTomlV2UTIGroupsRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	original := DefaultWithPandocFormatter()

	typedBlob := TypedBlob{
		Type: ids.MustTypeStruct(ids.TypeTomlTypeV2).ToMadder(),
		Blob: &original,
	}

	var buf bytes.Buffer
	bufferedWriter := bufio.NewWriter(&buf)

	if _, err := CoderToTypedBlob.Blob.EncodeTo(
		&typedBlob,
		bufferedWriter,
	); err != nil {
		t.Fatalf("encode failed: %s", err)
	}

	if err := bufferedWriter.Flush(); err != nil {
		t.Fatalf("flush failed: %s", err)
	}

	doc, err := DecodeTomlV2(buf.Bytes())
	if err != nil {
		t.Fatalf("decode failed: %s", err)
	}

	decoded := doc.Data()

	if len(decoded.UTIGroups) != len(original.UTIGroups) {
		t.Fatalf(
			"expected %d uti-groups after round-trip, got %d: %v",
			len(original.UTIGroups), len(decoded.UTIGroups), decoded.UTIGroups,
		)
	}

	for groupName, wantGroup := range original.UTIGroups {
		gotGroup, ok := decoded.UTIGroups[groupName]

		if !ok {
			t.Errorf("missing uti-group %q after round-trip", groupName)
			continue
		}

		if !gotGroup.Equals(wantGroup) {
			t.Errorf(
				"uti-group %q: expected %v after round-trip, got %v",
				groupName, wantGroup, gotGroup,
			)
		}
	}
}
