package tridex

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
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
		t.AssertNoError(err)

		t.AssertNoError(binary.Write(&buf, binary.BigEndian, uint32(len(bs))))

		_, err = buf.Write(bs)
		t.AssertNoError(err)
	}

	// Read: same framing as store_abbr readIfNecessary
	reader := bytes.NewReader(buf.Bytes())

	restored := make([]interfaces.TridexMutable, len(originals))
	for i := range restored {
		restored[i] = Make()
	}

	for _, tri := range restored {
		var length uint32

		t.AssertNoError(binary.Read(reader, binary.BigEndian, &length))

		bs := make([]byte, length)

		_, err := io.ReadFull(reader, bs)
		t.AssertNoError(err)

		t.AssertNoError(tri.(*Tridex).UnmarshalBinary(bs))
	}

	restoredHeads := restored[5]
	restoredTails := restored[6]

	// This is the exact operation that fails in the bats test:
	// show -format object-id o/u should expand to one/uno
	expandedHead := restoredHeads.Expand("o")
	expandedTail := restoredTails.Expand("u")

	t.AssertEqualStrings("one", expandedHead)
	t.AssertEqualStrings("uno", expandedTail)

	// Verify all tridexes round-tripped correctly
	names := []string{"repo", "tags", "types", "zettels", "marklIds", "heads", "tails"}

	for i := range names {
		t.AssertEqual(originals[i].Len(), restored[i].Len())
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
		t.AssertEqualStrings(et.expects, actual)
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
		t.AssertNoError(err)

		t.AssertNoError(binary.Write(&buf, binary.BigEndian, uint32(len(bs))))

		_, err = buf.Write(bs)
		t.AssertNoError(err)
	}

	reader := bytes.NewReader(buf.Bytes())

	for i := 0; i < 7; i++ {
		var length uint32

		t.AssertNoError(binary.Read(reader, binary.BigEndian, &length))

		bs := make([]byte, length)

		_, err := io.ReadFull(reader, bs)
		t.AssertNoError(err)

		restored := Make()

		t.AssertNoError(restored.(*Tridex).UnmarshalBinary(bs))

		t.AssertEqual(0, restored.Len())
	}
}
