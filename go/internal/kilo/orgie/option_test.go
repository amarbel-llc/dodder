package orgie

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

type testMutableConfigDryRun struct {
	dryRun bool
}

func (c *testMutableConfigDryRun) IsDryRun() bool {
	return c.dryRun
}

func (c *testMutableConfigDryRun) SetDryRun(v bool) {
	c.dryRun = v
}

// TestMetadataSettingsFieldReadWriteRoundTrip pins the OptionCommentSet.Set
// double-wrap fix: before the fix, a registered settings field consumed via
// ReadFrom (e.g. the legacy `% dry-run:true` comment) was wrapped in
// OptionCommentWithKey twice, which broke both String() (rendering
// "dry-run:dry-run:true") and the OptionCommentSettingsField interface
// assertion WriteTo uses to choose the `- _key=value` spelling (Go's
// embedded-interface promotion does not forward methods beyond what the
// embedded interface itself declares, so the doubly-wrapped value silently
// fails the assertion and falls back to the `%` spelling).
func TestMetadataSettingsFieldReadWriteRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})
	metadata.OptionCommentSet.AddPrototype(
		"dry-run",
		&OptionCommentDryRun{MutableConfigDryRun: &testMutableConfigDryRun{}},
	)

	_, err := metadata.ReadFrom(strings.NewReader("% dry-run:true\n"))
	t.AssertNoError(err)

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	expected := "- _dry-run=true\n"

	if buf.String() != expected {
		t.Errorf("\nexpected: %q\n  actual: %q", expected, buf.String())
	}
}

// TestMetadataReadFromRejectsUnregisteredSettingsField pins the ReadFrom
// regression fix: a `- key=value` line is only treated as a settings field
// when `key` is BOTH `_`-prefixed AND resolves to a registered prototype
// implementing OptionCommentSettingsField. Before the fix, any `_`-prefixed
// `=`-containing line -- including one matching a real but non-settings-
// field prototype like the built-in "hide" -- was silently absorbed as an
// "unknown option comment" with no error and no tag added, instead of
// falling through to tag parsing (which correctly rejects `=` as invalid
// tag content).
func TestMetadataReadFromRejectsUnregisteredSettingsField(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("- _hide=true\n"))

	if err == nil {
		t.Fatalf("expected an error for \"_hide=true\" (registered as \"hide\", but not a settings field), got nil")
	}
}

// TestMetadataReadFromNoOpsOnEntirelyUnknownSettingsFieldKey pins the
// complementary case: a `_`-prefixed key with NO prototype registered at
// all (e.g. "dry-run" read without -dry-run on the CLI, so
// ApplyToOrganizeOptions never registered it) must remain a harmless no-op,
// exactly like an unregistered `%` comment always has been -- this is what
// lets a generated document's settings fields round-trip through contexts
// that don't recognize them, and is the case
// organize_dry_run_reads_settings_field / …_legacy_comment_alias_… pin at
// the bats level.
func TestMetadataReadFromNoOpsOnEntirelyUnknownSettingsFieldKey(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("- _dry-run=true\n"))
	t.AssertNoError(err)

	if metadata.TagSet.Len() != 0 {
		t.Errorf("expected no tags, got: %s", metadata.TagSet)
	}
}
