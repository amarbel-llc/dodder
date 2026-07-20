//go:build test

package import_plan

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestOmitTagsTransformRemovesMatchingTags(t1 *testing.T) {
	t := ui.MakeT(t1)
	transform, err := MakeOmitTagsTransform([]string{"^tag-[12]$"})
	t.AssertNoError(err)

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")
	object.GetMetadataMutable().AddTagString("tag-3")

	keep, err := transform(&object)
	t.AssertNoError(err)

	t.AssertTrue(keep, "expected keep=true")

	tags := collectTagStrings(&object)
	if len(tags) != 1 || tags[0] != "tag-3" {
		t.Fatalf("expected [tag-3], got %v", tags)
	}
}

func TestOmitTagsTransformKeepsAllWhenNoMatch(t1 *testing.T) {
	t := ui.MakeT(t1)
	transform, err := MakeOmitTagsTransform([]string{"^archived$"})
	t.AssertNoError(err)

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")

	keep, err := transform(&object)
	t.AssertNoError(err)

	t.AssertTrue(keep, "expected keep=true")

	tags := collectTagStrings(&object)
	t.AssertLen(2, tags, "tags")
}

func TestOmitTagsTransformMultiplePatterns(t1 *testing.T) {
	t := ui.MakeT(t1)
	transform, err := MakeOmitTagsTransform([]string{"^tag-1$", "^tag-3$"})
	t.AssertNoError(err)

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")
	object.GetMetadataMutable().AddTagString("tag-3")

	keep, err := transform(&object)
	t.AssertNoError(err)

	t.AssertTrue(keep, "expected keep=true")

	tags := collectTagStrings(&object)
	if len(tags) != 1 || tags[0] != "tag-2" {
		t.Fatalf("expected [tag-2], got %v", tags)
	}
}

func TestOmitTagsTransformRemovesAllTags(t1 *testing.T) {
	t := ui.MakeT(t1)
	transform, err := MakeOmitTagsTransform([]string{".*"})
	t.AssertNoError(err)

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")

	keep, err := transform(&object)
	t.AssertNoError(err)

	t.AssertTrue(keep, "expected keep=true even when all tags removed")

	tags := collectTagStrings(&object)
	t.AssertLen(0, tags, "tags")
}

func TestOmitTagsTransformInvalidRegex(t1 *testing.T) {
	t := ui.MakeT(t1)
	_, err := MakeOmitTagsTransform([]string{"[invalid"})
	t.AssertError(err)
}

func collectTagStrings(object *sku.Transacted) []string {
	var result []string

	for tag := range object.GetMetadata().AllTags() {
		result = append(result, tag.String())
	}

	return result
}
