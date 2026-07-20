//go:build test

package store

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// makeRequiredURLTypeBlob returns a synthetic TomlV2 shaped like !bookmark:
// one required string field (url) plus one optional string field (notes).
func makeRequiredURLTypeBlob() *type_blobs.TomlV2 {
	return &type_blobs.TomlV2{
		Fields: []type_blobs.FieldDefinition{
			{Name: "url", Kind: "string", Required: true},
			{Name: "notes", Kind: "string"},
		},
	}
}

func projectFieldsForTest(
	blob *type_blobs.TomlV2,
	scriptOutput map[string]string,
) ([]fields.Field, error) {
	var appended []fields.Field

	err := projectFields(
		ids.MustType("bookmark"),
		blob.GetFieldDefinitions(),
		scriptOutput,
		markl.Id{},
		func(field fields.Field) {
			appended = append(appended, field)
		},
	)

	return appended, err
}

func assertErrorContains(t *ui.T, err error, want string) {
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", want)
	}

	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error to contain %q, got %q", want, err.Error())
	}
}

func TestProjectFieldsRequiredAbsentRejected(t1 *testing.T) {
	t := ui.MakeT(t1)

	_, err := projectFieldsForTest(
		makeRequiredURLTypeBlob(),
		map[string]string{"notes": "no url key at all"},
	)

	assertErrorContains(
		&t,
		err,
		`type !bookmark: field "url" is required but missing or empty`,
	)
}

func TestProjectFieldsRequiredEmptyRejected(t1 *testing.T) {
	t := ui.MakeT(t1)

	_, err := projectFieldsForTest(
		makeRequiredURLTypeBlob(),
		map[string]string{"url": ""},
	)

	assertErrorContains(
		&t,
		err,
		`type !bookmark: field "url" is required but missing or empty`,
	)
}

func TestProjectFieldsRequiredPresentAccepted(t1 *testing.T) {
	t := ui.MakeT(t1)

	appended, err := projectFieldsForTest(
		makeRequiredURLTypeBlob(),
		map[string]string{
			"url":   "https://example.com/page",
			"notes": "worth keeping",
		},
	)

	t.AssertNoError(err)
	t.AssertLen(2, appended, "appended fields")

	t.AssertEqualStrings("url", appended[0].Key)
	t.AssertEqualStrings("https://example.com/page", appended[0].Value)
	t.AssertEqualStrings("notes", appended[1].Key)
	t.AssertEqualStrings("worth keeping", appended[1].Value)
}

// A required field with a non-empty default is always satisfiable: an
// absent key falls back to the default before the required check runs.
func TestProjectFieldsRequiredWithDefaultAbsentAccepted(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := &type_blobs.TomlV2{
		Fields: []type_blobs.FieldDefinition{
			{
				Name:     "status",
				Kind:     "enum",
				Values:   []string{"todo", "done"},
				Default:  "todo",
				Required: true,
			},
		},
	}

	appended, err := projectFieldsForTest(blob, map[string]string{})

	t.AssertNoError(err)
	t.AssertLen(1, appended, "appended fields")
	t.AssertEqualStrings("todo", appended[0].Value)
}

func TestBloblessCommitRejectedWhenTypeHasRequiredFields(t1 *testing.T) {
	t := ui.MakeT(t1)

	err := validateBloblessAgainstRequiredFields(
		ids.MustType("bookmark"),
		makeRequiredURLTypeBlob().GetFieldDefinitions(),
	)

	assertErrorContains(
		&t,
		err,
		"type !bookmark requires fields (url) but object has no blob",
	)
}

func TestBloblessCommitAcceptedWhenTypeHasNoRequiredFields(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := &type_blobs.TomlV2{
		Fields: []type_blobs.FieldDefinition{
			{
				Name:    "status",
				Kind:    "enum",
				Values:  []string{"todo", "done"},
				Default: "todo",
			},
			{Name: "due", Kind: "string"},
		},
	}

	err := validateBloblessAgainstRequiredFields(
		ids.MustType("task"),
		blob.GetFieldDefinitions(),
	)

	t.AssertNoError(err)
}
