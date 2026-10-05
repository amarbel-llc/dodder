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
			Type: ids.MustTypeStruct(ids.TypeTomlTypeVLatest).ToMadder(),
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

// TestTomlV3RejectsMalformedBlob pins the v3 strictness: where the v0-v2
// coders decode a malformed blob as an empty type, v3 surfaces the error so a
// broken type cannot silently lose its hooks, formatters, and fields.
func TestTomlV3RejectsMalformedBlob(t1 *testing.T) {
	// The lenient DecodeTomlV3 accepts all of these without error; the v3
	// coder rejects the first two via DecodeTomlV3Strict and the third by
	// refusing undecoded keys.
	for name, malformed := range map[string]string{
		"unterminated values": "file-extension = \"md\nhooks = [unterminated\n",
		"mistyped value":      "hooks = 42\n",
		"unknown key":         "no-such-key = \"x\"\n",
	} {
		typedBlob := TypedBlob{
			Type: ids.MustTypeStruct(ids.TypeTomlTypeV3).ToMadder(),
		}

		if _, err := CoderToTypedBlob.Blob.DecodeFrom(
			&typedBlob,
			bufio.NewReader(strings.NewReader(malformed)),
		); err == nil {
			t1.Errorf(
				"v3 coder accepted %s, decoded %#v",
				name,
				typedBlob.Blob,
			)
		}
	}

	// A well-formed blob still decodes.
	typedBlob := TypedBlob{
		Type: ids.MustTypeStruct(ids.TypeTomlTypeV3).ToMadder(),
	}

	if _, err := CoderToTypedBlob.Blob.DecodeFrom(
		&typedBlob,
		bufio.NewReader(strings.NewReader(
			"file-extension = \"md\"\nhooks = \"return {}\"\n",
		)),
	); err != nil {
		t1.Fatalf("v3 coder rejected a well-formed blob: %s", err)
	}
}

// Every built-in type blob must round-trip through the strict v3 decoder: the
// unconsumed-key check must not trip on the map-backed tables the encoder
// seeds (uti-groups, formatters) or on nested field definitions.
func TestTomlV3StrictDecodeAcceptsBuiltinShapes(t1 *testing.T) {
	for name, blob := range map[string]TomlV3{
		"md":        Default(),
		"md-pandoc": DefaultWithPandocFormatter(),
		"task":      DefaultTaskType(),
		"chore":     DefaultChoreType(),
		"habit":     DefaultHabitType(),
	} {
		encoded := TypedBlob{
			Type: ids.MustTypeStruct(ids.TypeTomlTypeV3).ToMadder(),
			Blob: &blob,
		}

		var buf bytes.Buffer
		bufferedWriter := bufio.NewWriter(&buf)

		if _, err := CoderToTypedBlob.Blob.EncodeTo(
			&encoded,
			bufferedWriter,
		); err != nil {
			t1.Fatalf("%s: encode failed: %s", name, err)
		}

		if err := bufferedWriter.Flush(); err != nil {
			t1.Fatalf("%s: flush failed: %s", name, err)
		}

		decoded := TypedBlob{
			Type: ids.MustTypeStruct(ids.TypeTomlTypeV3).ToMadder(),
		}

		if _, err := CoderToTypedBlob.Blob.DecodeFrom(
			&decoded,
			bufio.NewReader(bytes.NewReader(buf.Bytes())),
		); err != nil {
			t1.Errorf("%s: strict decode rejected its own encoding: %s\n%s", name, err, buf.String())
		}
	}
}

// TestTomlV3RefusesOtherVersionStruct pins the encode guard: a v2 struct under
// the v3 type string is an error, not an empty type blob.
func TestTomlV3RefusesOtherVersionStruct(t1 *testing.T) {
	blob := TomlV2{FileExtension: "md"}

	typedBlob := TypedBlob{
		Type: ids.MustTypeStruct(ids.TypeTomlTypeV3).ToMadder(),
		Blob: &blob,
	}

	var buf bytes.Buffer

	if _, err := CoderToTypedBlob.Blob.EncodeTo(
		&typedBlob,
		bufio.NewWriter(&buf),
	); err == nil {
		t1.Fatalf("v3 coder encoded a %T without error", typedBlob.Blob)
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
		Type: ids.MustTypeStruct(ids.TypeTomlTypeVLatest).ToMadder(),
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

	doc, err := DecodeTomlV3(buf.Bytes())
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
