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

// TestMetadataSettingsFieldReadWriteRoundTrip pins the SettingSet.Set
// double-wrap fix: before the fix, a registered settings field consumed via
// ReadFrom (e.g. the legacy `% dry-run:true` comment) was wrapped in
// SettingWithKey twice, which broke both String() (rendering
// "dry-run:dry-run:true") and the SettingAsField interface
// assertion WriteTo uses to choose the `- _key=value` spelling (Go's
// embedded-interface promotion does not forward methods beyond what the
// embedded interface itself declares, so the doubly-wrapped value silently
// fails the assertion and falls back to the `%` spelling).
func TestMetadataSettingsFieldReadWriteRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})
	metadata.SettingSet.AddPrototype(
		"dry-run",
		&SettingDryRun{MutableConfigDryRun: &testMutableConfigDryRun{}},
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
// implementing SettingAsField. Before the fix, any `_`-prefixed
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

// TestMetadataBaseDigestSettingsFieldRoundTrip pins the dodder#374(b)
// `_base=@<digest>` settings field: it's registered unconditionally
// (unlike `_dry-run`, which is only registered when -dry-run is active),
// so it must round-trip through ReadFrom/WriteTo with no special setup.
func TestMetadataBaseDigestSettingsFieldRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	const digest = "@blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz"

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("- _base=" + digest + "\n"))
	t.AssertNoError(err)

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	expected := "- _base=" + digest + "\n"

	if buf.String() != expected {
		t.Errorf("\nexpected: %q\n  actual: %q", expected, buf.String())
	}
}

// TestMetadataBaseDigestRejectsMalformedValue pins that `_base` validates
// its value is actually digest-shaped rather than accepting an arbitrary
// string -- `_base` is required on every organize document (dodder#374(b)
// §8), so a malformed value must fail loudly at parse time, not surface
// later as a confusing "undereferenceable" error.
func TestMetadataBaseDigestRejectsMalformedValue(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("- _base=not-a-digest\n"))

	if err == nil {
		t.Fatalf("expected an error for a malformed _base value, got nil")
	}
}

// TestMetadataAllowDeletionSettingsFieldRoundTrip pins the
// dodder#374(b) `_allow-deletion=true` settings field, structurally
// identical to `_dry-run` but also registered unconditionally.
func TestMetadataAllowDeletionSettingsFieldRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("- _allow-deletion=true\n"))
	t.AssertNoError(err)

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	expected := "- _allow-deletion=true\n"

	if buf.String() != expected {
		t.Errorf("\nexpected: %q\n  actual: %q", expected, buf.String())
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
