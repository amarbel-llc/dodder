//go:build test

package import_plan

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
)

func TestOmitTagsTransformRemovesMatchingTags(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{"^tag-[12]$"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")
	object.GetMetadataMutable().AddTagString("tag-3")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}

	if !keep {
		t.Fatal("expected keep=true")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 1 || tags[0] != "tag-3" {
		t.Fatalf("expected [tag-3], got %v", tags)
	}
}

func TestOmitTagsTransformKeepsAllWhenNoMatch(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{"^archived$"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}

	if !keep {
		t.Fatal("expected keep=true")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", tags)
	}
}

func TestOmitTagsTransformMultiplePatterns(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{"^tag-1$", "^tag-3$"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")
	object.GetMetadataMutable().AddTagString("tag-2")
	object.GetMetadataMutable().AddTagString("tag-3")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}

	if !keep {
		t.Fatal("expected keep=true")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 1 || tags[0] != "tag-2" {
		t.Fatalf("expected [tag-2], got %v", tags)
	}
}

func TestOmitTagsTransformRemovesAllTags(t *testing.T) {
	transform, err := MakeOmitTagsTransform([]string{".*"})
	if err != nil {
		t.Fatal(err)
	}

	var object sku.Transacted
	object.GetMetadataMutable().AddTagString("tag-1")

	keep, err := transform(&object)
	if err != nil {
		t.Fatal(err)
	}

	if !keep {
		t.Fatal("expected keep=true even when all tags removed")
	}

	tags := collectTagStrings(&object)
	if len(tags) != 0 {
		t.Fatalf("expected no tags, got %v", tags)
	}
}

func TestOmitTagsTransformInvalidRegex(t *testing.T) {
	_, err := MakeOmitTagsTransform([]string{"[invalid"})
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func collectTagStrings(object *sku.Transacted) []string {
	var result []string

	for tag := range object.GetMetadata().AllTags() {
		result = append(result, tag.String())
	}

	return result
}
