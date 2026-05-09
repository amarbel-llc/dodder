package tridex

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

// TestFramedMultiTridexRoundTrip replicates the store_abbr serialization
// pattern: 7 tridexes written as length-prefixed binary blobs, then read
// back. The content mirrors what the bats fixture setup creates (yin=one/two/
// three/.., yang=uno/dos/tres/.., two zettels with tags).
func TestFramedMultiTridexRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	// Build tridexes matching bats fixture state after create_test_zettels
	repo := Make(".")
	tags := Make("tag-1", "tag-2", "tag-3", "tag-4")
	types := Make("md")
	zettels := Make("one/uno", "two/dos")
	marklIds := Make(
		"blake2b256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"blake2b256-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	heads := Make("one", "two")
	tails := Make("uno", "dos")

	originals := []interfaces.TridexMutable{
		repo, tags, types, zettels, marklIds, heads, tails,
	}

	// Write: same framing as store_abbr Flush
	var buf bytes.Buffer

	for _, tri := range originals {
		bs, err := tri.(*Tridex).MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary failed: %s", err)
		}

		if err := binary.Write(&buf, binary.BigEndian, uint32(len(bs))); err != nil {
			t.Fatalf("write length failed: %s", err)
		}

		if _, err := buf.Write(bs); err != nil {
			t.Fatalf("write data failed: %s", err)
		}
	}

	// Read: same framing as store_abbr readIfNecessary
	reader := bytes.NewReader(buf.Bytes())

	restored := make([]interfaces.TridexMutable, len(originals))
	for i := range restored {
		restored[i] = Make()
	}

	for i, tri := range restored {
		var length uint32

		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			t.Fatalf("read length %d failed: %s", i, err)
		}

		bs := make([]byte, length)

		if _, err := io.ReadFull(reader, bs); err != nil {
			t.Fatalf("read data %d failed: %s", i, err)
		}

		if err := tri.(*Tridex).UnmarshalBinary(bs); err != nil {
			t.Fatalf("UnmarshalBinary %d failed: %s", i, err)
		}
	}

	restoredHeads := restored[5]
	restoredTails := restored[6]

	// This is the exact operation that fails in the bats test:
	// show -format object-id o/u should expand to one/uno
	expandedHead := restoredHeads.Expand("o")
	expandedTail := restoredTails.Expand("u")

	if expandedHead != "one" {
		t.Errorf("head expansion: expected %q, got %q", "one", expandedHead)
	}

	if expandedTail != "uno" {
		t.Errorf("tail expansion: expected %q, got %q", "uno", expandedTail)
	}

	// Verify all tridexes round-tripped correctly
	names := []string{"repo", "tags", "types", "zettels", "marklIds", "heads", "tails"}

	for i, name := range names {
		if originals[i].Len() != restored[i].Len() {
			t.Errorf("%s: Len mismatch: expected %d, got %d",
				name, originals[i].Len(), restored[i].Len())
		}
	}

	// Verify specific expansions work
	expandTests := []struct {
		name    string
		tridex  interfaces.TridexMutable
		abbr    string
		expects string
	}{
		{"heads o→one", restoredHeads, "o", "one"},
		{"heads t→two", restoredHeads, "t", "two"},
		{"tails u→uno", restoredTails, "u", "uno"},
		{"tails d→dos", restoredTails, "d", "dos"},
		{"zettels one/→one/uno", restored[3], "one/", "one/uno"},
		{"types m→md", restored[2], "m", "md"},
		{"tags tag-1→tag-1", restored[1], "tag-1", "tag-1"},
	}

	for _, et := range expandTests {
		actual := et.tridex.Expand(et.abbr)
		if actual != et.expects {
			t.Errorf("%s: expected %q, got %q", et.name, et.expects, actual)
		}
	}
}

// TestFramedEmptyTridexRoundTrip verifies that empty tridexes survive the
// framing protocol (important for fresh repos where no objects exist yet).
func TestFramedEmptyTridexRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	originals := make([]interfaces.TridexMutable, 7)
	for i := range originals {
		originals[i] = Make()
	}

	var buf bytes.Buffer

	for _, tri := range originals {
		bs, err := tri.(*Tridex).MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary failed: %s", err)
		}

		if err := binary.Write(&buf, binary.BigEndian, uint32(len(bs))); err != nil {
			t.Fatalf("write length failed: %s", err)
		}

		if _, err := buf.Write(bs); err != nil {
			t.Fatalf("write data failed: %s", err)
		}
	}

	reader := bytes.NewReader(buf.Bytes())

	for i := 0; i < 7; i++ {
		var length uint32

		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			t.Fatalf("read length %d failed: %s", i, err)
		}

		bs := make([]byte, length)

		if _, err := io.ReadFull(reader, bs); err != nil {
			t.Fatalf("read data %d failed: %s", i, err)
		}

		restored := Make()

		if err := restored.(*Tridex).UnmarshalBinary(bs); err != nil {
			t.Fatalf("UnmarshalBinary %d failed: %s", i, err)
		}

		if restored.Len() != 0 {
			t.Errorf("tridex %d: expected empty, got Len=%d", i, restored.Len())
		}
	}
}
