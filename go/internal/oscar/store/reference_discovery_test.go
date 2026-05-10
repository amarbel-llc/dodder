//go:build test

package store

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestParseReferenceOutputEmpty(t1 *testing.T) {
	t := ui.MakeT(t1)

	refs, err := parseReferenceOutput("")
	t.AssertNoError(err)

	t.AssertLen(0, refs, "refs")
}

func TestParseReferenceOutputSimpleRefs(t1 *testing.T) {
	t := ui.MakeT(t1)

	input := "one/dos\ntwo/uno\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	t.AssertLen(2, refs, "refs")

	t.AssertEqualStrings("one/dos", refs[0].ObjectId)
	t.AssertEqualStrings("", refs[0].Alias)
	t.AssertEqualStrings("two/uno", refs[1].ObjectId)
	t.AssertEqualStrings("", refs[1].Alias)
}

func TestParseReferenceOutputWithAliases(t1 *testing.T) {
	t := ui.MakeT(t1)

	input := "blog-template = one/uno\none/dos\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	t.AssertLen(2, refs, "refs")

	t.AssertEqualStrings("one/uno", refs[0].ObjectId)
	t.AssertEqualStrings("blog-template", refs[0].Alias)
	t.AssertEqualStrings("one/dos", refs[1].ObjectId)
	t.AssertEqualStrings("", refs[1].Alias)
}

func TestParseReferenceOutputBinaryGarbage(t1 *testing.T) {
	t := ui.MakeT(t1)

	input := "one/dos\n\x00\xff\xfe binary garbage\ntwo/uno\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	// Parser should yield refs for lines it can parse; garbage lines become
	// object refs with the raw string (they'll fail downstream validation,
	// but parseReferenceOutput itself shouldn't panic or error).
	t.AssertLen(3, refs, "refs")

	t.AssertEqualStrings("one/dos", refs[0].ObjectId)
	t.AssertEqualStrings("two/uno", refs[2].ObjectId)
}

func TestParseReferenceOutputPartialBlobRef(t1 *testing.T) {
	t := ui.MakeT(t1)

	// Blob ref without digest — just "@"
	input := "@\none/dos\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	t.AssertLen(2, refs, "refs")

	// Empty blob ID from bare "@"
	t.AssertEqualStrings("", refs[0].BlobId)
	t.AssertEqualStrings("one/dos", refs[1].ObjectId)
}

func TestParseReferenceOutputBlobRefWithAlias(t1 *testing.T) {
	t := ui.MakeT(t1)

	input := "hero = @blake2b256-abc123 !image-png\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	t.AssertLen(1, refs, "refs")

	t.AssertEqualStrings("blake2b256-abc123", refs[0].BlobId)
	t.AssertEqualStrings("hero", refs[0].Alias)
	t.AssertEqualStrings("!image-png", refs[0].TypeId)
}

func TestParseReferenceOutputSkipsCommentsAndBlanks(t1 *testing.T) {
	t := ui.MakeT(t1)

	input := "# this is a comment\n\none/dos\n  \n# another comment\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	t.AssertLen(1, refs, "refs")

	t.AssertEqualStrings("one/dos", refs[0].ObjectId)
}
