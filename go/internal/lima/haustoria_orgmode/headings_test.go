package haustoria_orgmode

import (
	"strings"
	"testing"
)

const sampleOrg = `* Nuisance distractions
:PROPERTIES:
:ID:       e794126b-b510-45af-a233-ee4c9e4879f1
:END:

One of my biggest focus inhibitors is frustration.

* Spinclass :work:health:
- explore session state
- explore session ids

* TODO Today important
SCHEDULED: <2025-08-25 Mon>
:PROPERTIES:
:ID:       05a231ff-d627-486f-9373-3b98b9f83878
:END:

body content here
`

func TestParseFile_Counts(t *testing.T) {
	headings, err := parseFile([]byte(sampleOrg))
	if err != nil {
		t.Fatal(err)
	}
	if len(headings) != 3 {
		t.Fatalf("want 3 headings, got %d", len(headings))
	}
}

func TestParseFile_Titles(t *testing.T) {
	headings, _ := parseFile([]byte(sampleOrg))
	expected := []string{"Nuisance distractions", "Spinclass", "TODO Today important"}
	for i, h := range headings {
		if h.Title != expected[i] {
			t.Errorf("heading %d: want title %q, got %q", i, expected[i], h.Title)
		}
	}
}

func TestParseFile_Tags(t *testing.T) {
	headings, _ := parseFile([]byte(sampleOrg))
	if len(headings[1].Tags) != 2 {
		t.Fatalf("heading 1: want 2 tags, got %d", len(headings[1].Tags))
	}
	if headings[1].Tags[0] != "work" || headings[1].Tags[1] != "health" {
		t.Errorf("heading 1: want [work health], got %v", headings[1].Tags)
	}
}

func TestParseFile_IDs(t *testing.T) {
	headings, _ := parseFile([]byte(sampleOrg))
	if headings[0].ID != "e794126b-b510-45af-a233-ee4c9e4879f1" {
		t.Errorf("heading 0: want ID e794126b..., got %q", headings[0].ID)
	}
	if headings[1].ID != "" {
		t.Errorf("heading 1: want no ID, got %q", headings[1].ID)
	}
	if headings[2].ID != "05a231ff-d627-486f-9373-3b98b9f83878" {
		t.Errorf("heading 2: want ID 05a231ff..., got %q", headings[2].ID)
	}
}

func TestNormalizeIDs_AddsMissingIDs(t *testing.T) {
	headings, _ := parseFile([]byte(sampleOrg))

	newContent, changed, err := normalizeIDs([]byte(sampleOrg), headings)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true since heading 1 lacks an ID")
	}

	// Re-parse the new content and verify all 3 headings now have IDs.
	headings2, err := parseFile(newContent)
	if err != nil {
		t.Fatal(err)
	}
	if len(headings2) != 3 {
		t.Fatalf("want 3 headings after normalize, got %d", len(headings2))
	}
	for i, h := range headings2 {
		if h.ID == "" {
			t.Errorf("heading %d: still has no ID after normalize", i)
		}
	}

	// Pre-existing IDs must be preserved exactly.
	if headings2[0].ID != "e794126b-b510-45af-a233-ee4c9e4879f1" {
		t.Errorf("heading 0 ID changed: got %q", headings2[0].ID)
	}
	if headings2[2].ID != "05a231ff-d627-486f-9373-3b98b9f83878" {
		t.Errorf("heading 2 ID changed: got %q", headings2[2].ID)
	}

	// The body content from the original file should still be present.
	for _, expected := range []string{
		"One of my biggest focus inhibitors",
		"explore session state",
		"body content here",
	} {
		if !strings.Contains(string(newContent), expected) {
			t.Errorf("body line %q missing from normalized output", expected)
		}
	}
}

func TestNormalizeIDs_Idempotent(t *testing.T) {
	headings, _ := parseFile([]byte(sampleOrg))
	newContent, _, _ := normalizeIDs([]byte(sampleOrg), headings)

	headings2, _ := parseFile(newContent)
	newContent2, changed, err := normalizeIDs(newContent, headings2)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Errorf("expected normalize to be a no-op on already-normalized content")
	}
	if !equalBytes(newContent, newContent2) {
		t.Errorf("normalized content changed on second pass")
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
