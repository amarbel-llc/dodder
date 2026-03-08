//go:build test

package store

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestParseReferenceOutputEmpty(t1 *testing.T) {
	t := ui.T{T: t1}

	refs, err := parseReferenceOutput("")
	t.AssertNoError(err)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d", len(refs))
	}
}

func TestParseReferenceOutputSimpleRefs(t1 *testing.T) {
	t := ui.T{T: t1}

	input := "one/dos\ntwo/uno\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}

	t.AssertEqualStrings("one/dos", refs[0].ObjectId)
	t.AssertEqualStrings("", refs[0].Alias)
	t.AssertEqualStrings("two/uno", refs[1].ObjectId)
	t.AssertEqualStrings("", refs[1].Alias)
}

func TestParseReferenceOutputWithAliases(t1 *testing.T) {
	t := ui.T{T: t1}

	input := "blog-template = one/uno\none/dos\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}

	t.AssertEqualStrings("one/uno", refs[0].ObjectId)
	t.AssertEqualStrings("blog-template", refs[0].Alias)
	t.AssertEqualStrings("one/dos", refs[1].ObjectId)
	t.AssertEqualStrings("", refs[1].Alias)
}

func TestParseReferenceOutputSkipsCommentsAndBlanks(t1 *testing.T) {
	t := ui.T{T: t1}

	input := "# this is a comment\n\none/dos\n  \n# another comment\n"
	refs, err := parseReferenceOutput(input)
	t.AssertNoError(err)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}

	t.AssertEqualStrings("one/dos", refs[0].ObjectId)
}
