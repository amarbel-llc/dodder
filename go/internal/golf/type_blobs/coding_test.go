//go:build test

package type_blobs

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
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

	// 20 re-encodes: with four formatter keys, unsorted map iteration would
	// produce a differing order with near-certainty.
	for range 20 {
		t.AssertEqualStrings(first, encode())
	}

	// The formatter tables must appear in sorted key order.
	expectedOrder := []string{
		"[formatters.html]",
		"[formatters.html-gdoc]",
		"[formatters.pdf-beamer]",
		"[formatters.text]",
	}

	previousIndex := -1

	for _, header := range expectedOrder {
		index := strings.Index(first, header+"\n")

		if index < 0 {
			t.Fatalf("missing %q in encoded blob:\n%s", header, first)
		}

		if index <= previousIndex {
			t.Errorf("%q out of sorted order in encoded blob:\n%s", header, first)
		}

		previousIndex = index
	}
}
