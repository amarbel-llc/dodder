//go:build test

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// expectedSeedTypeCount pins the Group 3 triage result: 57 types classified
// as the dodder.net seed set (docs/plans/2026-07-04-type-reconciliation-
// groups.md §"Group 3 triage results"). A table edit that changes the count
// must consciously update this.
const expectedSeedTypeCount = 57

// allowedVimFiletypes is the allowlist of vim-syntax-type values the table
// may use: vim's REAL filetype names (":help filetype"), not extensions
// (javascript, not js; sh, not bash), plus "" for binary formats. rego and
// plantuml are plugin-conventional filetypes (not in vim core runtime) —
// kept because they are the de-facto names the respective vim plugins set.
var allowedVimFiletypes = map[string]bool{
	"":           true,
	"awk":        true,
	"cfg":        true,
	"csv":        true,
	"dot":        true,
	"javascript": true,
	"jq":         true,
	"json":       true,
	"lilypond":   true,
	"mail":       true,
	"plantuml":   true,
	"rego":       true,
	"sh":         true,
	"svg":        true,
	"xml":        true,
}

var (
	seedTypeNamePattern      = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	seedTypeExtensionPattern = regexp.MustCompile(`^[a-z0-9]+$`)
)

func TestSeedTypeTableInvariants(t1 *testing.T) {
	t := ui.MakeT(t1)

	if len(seedTypes) != expectedSeedTypeCount {
		t.Fatalf(
			"expected %d seed types, got %d",
			expectedSeedTypeCount,
			len(seedTypes),
		)
	}

	seenNames := make(map[string]bool, len(seedTypes))

	for _, entry := range seedTypes {
		if !seedTypeNamePattern.MatchString(entry.Name) {
			t.Errorf("invalid seed type name: %q", entry.Name)
		}

		if seenNames[entry.Name] {
			t.Errorf("duplicate seed type name: %q", entry.Name)
		}

		seenNames[entry.Name] = true

		if entry.Description == "" {
			t.Errorf("seed type %q has no description", entry.Name)
		}

		if !seedTypeExtensionPattern.MatchString(entry.FileExtension) {
			t.Errorf(
				"seed type %q has invalid file extension: %q",
				entry.Name,
				entry.FileExtension,
			)
		}

		if !allowedVimFiletypes[entry.VimSyntaxType] {
			t.Errorf(
				"seed type %q has vim-syntax-type %q outside the allowlist",
				entry.Name,
				entry.VimSyntaxType,
			)
		}

		if entry.Binary && entry.VimSyntaxType != "" {
			t.Errorf(
				"binary seed type %q must not set a vim-syntax-type (got %q)",
				entry.Name,
				entry.VimSyntaxType,
			)
		}
	}

	if !sort.SliceIsSorted(
		seedTypes,
		func(left, right int) bool {
			return seedTypes[left].Name < seedTypes[right].Name
		},
	) {
		t.Errorf("seed type table is not sorted by name")
	}
}

// TestSeedTypeGenerationDeterministic pins the regeneration contract: two
// generation runs into fresh directories produce byte-identical file sets,
// and rerunning over an existing directory prunes stale .type files from
// removed table entries.
func TestSeedTypeGenerationDeterministic(t1 *testing.T) {
	t := ui.MakeT(t1)

	readAll := func(dir string) map[string][]byte {
		files := make(map[string][]byte)

		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("reading %s: %s", dir, err)
		}

		for _, dirEntry := range dirEntries {
			var content []byte

			content, err = os.ReadFile(filepath.Join(dir, dirEntry.Name()))
			if err != nil {
				t.Fatalf("reading %s: %s", dirEntry.Name(), err)
			}

			files[dirEntry.Name()] = content
		}

		return files
	}

	firstDir := t1.TempDir()
	secondDir := t1.TempDir()

	if err := generateSeedTypes(firstDir); err != nil {
		t.Fatalf("first generation failed: %s", err)
	}

	if err := generateSeedTypes(secondDir); err != nil {
		t.Fatalf("second generation failed: %s", err)
	}

	firstFiles := readAll(firstDir)
	secondFiles := readAll(secondDir)

	if len(firstFiles) != expectedSeedTypeCount {
		t.Fatalf(
			"expected %d generated files, got %d",
			expectedSeedTypeCount,
			len(firstFiles),
		)
	}

	if len(firstFiles) != len(secondFiles) {
		t.Fatalf(
			"generation runs disagree on file count: %d vs %d",
			len(firstFiles),
			len(secondFiles),
		)
	}

	for name, firstContent := range firstFiles {
		if !bytes.Equal(firstContent, secondFiles[name]) {
			t.Errorf("generation of %s is not deterministic", name)
		}
	}

	// Rerunning over an existing directory must prune stale .type files
	// (removed table entries) and leave everything else byte-identical.
	stalePath := filepath.Join(firstDir, "removed-entry.type")

	if err := os.WriteFile(stalePath, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("writing stale file: %s", err)
	}

	if err := generateSeedTypes(firstDir); err != nil {
		t.Fatalf("regeneration failed: %s", err)
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Errorf("stale .type file was not pruned on regeneration")
	}

	regeneratedFiles := readAll(firstDir)

	if len(regeneratedFiles) != expectedSeedTypeCount {
		t.Fatalf(
			"expected %d files after regeneration, got %d",
			expectedSeedTypeCount,
			len(regeneratedFiles),
		)
	}
}

// TestSeedTypeBlobsParseAsTomlV2 confirms every generated blob body (the
// content after the hyphence metadata section, including the
// nixpkgs-formatter-candidates TOML comment) decodes as a TomlV2 type blob
// via the existing coding and round-trips the table's values.
func TestSeedTypeBlobsParseAsTomlV2(t1 *testing.T) {
	t := ui.MakeT(t1)

	boundary := []byte("---\n\n")

	for _, entry := range seedTypes {
		parts := bytes.SplitN(entry.render(), boundary, 2)

		if len(parts) != 2 {
			t.Fatalf(
				"seed type %q: no blank line after the closing hyphence boundary",
				entry.Name,
			)
		}

		doc, err := type_blobs.DecodeTomlV2(parts[1])
		if err != nil {
			t.Fatalf("seed type %q: blob does not decode: %s", entry.Name, err)
		}

		decoded := doc.Data()

		if decoded.Binary != entry.Binary {
			t.Errorf(
				"seed type %q: binary = %t after decode, expected %t",
				entry.Name,
				decoded.Binary,
				entry.Binary,
			)
		}

		if decoded.FileExtension != entry.FileExtension {
			t.Errorf(
				"seed type %q: file-extension = %q after decode, expected %q",
				entry.Name,
				decoded.FileExtension,
				entry.FileExtension,
			)
		}

		if decoded.VimSyntaxType != entry.VimSyntaxType {
			t.Errorf(
				"seed type %q: vim-syntax-type = %q after decode, expected %q",
				entry.Name,
				decoded.VimSyntaxType,
				entry.VimSyntaxType,
			)
		}
	}
}
