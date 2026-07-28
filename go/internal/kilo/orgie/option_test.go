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

	// RFC 0015 piece 4: dry-run is operational-plane, so it renders as
	// "%:dry-run = true" -- not the pre-RFC-0015 data-plane
	// "- _dry-run = true" spelling.
	expected := "%:dry-run = true\n"

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

	_, err := metadata.ReadFrom(strings.NewReader("- _hide = true\n"))

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

	_, err := metadata.ReadFrom(strings.NewReader("- _base = " + digest + "\n"))
	t.AssertNoError(err)

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	// Spaced "=" per RFC 0015's merged two-plane revision (ruled
	// 2026-07-28: normative for all metadata lines, no exemption for
	// already-shipped _base/_group-by).
	expected := "- _base = " + digest + "\n"

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

	_, err := metadata.ReadFrom(strings.NewReader("- _base = not-a-digest\n"))

	if err == nil {
		t.Fatalf("expected an error for a malformed _base value, got nil")
	}
}

// TestMetadataAllowDeletionSettingsFieldRoundTrip pins the
// dodder#374(b) `allow-deletion` settings field, structurally similar
// to `dry-run` and also registered unconditionally. RFC 0015 piece 4
// reclassifies it to the operational plane: `%:allow-deletion = true`,
// not the pre-RFC-0015 data-plane `- _allow-deletion = true` spelling.
func TestMetadataAllowDeletionSettingsFieldRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("%:allow-deletion = true\n"))
	t.AssertNoError(err)

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	expected := "%:allow-deletion = true\n"

	if buf.String() != expected {
		t.Errorf("\nexpected: %q\n  actual: %q", expected, buf.String())
	}
}

// TestMetadataReadFromRejectsLegacyDryRunDataPlaneSyntax pins RFC 0015
// piece 4's reclassification from the read side: once "dry-run" is
// registered (mirroring ApplyToOrganizeOptions's real -dry-run-active
// case), the pre-RFC-0015 data-plane spelling `- _dry-run = true` must
// error -- SettingDryRun no longer implements SettingAsField, so this
// is the same "registered but not a settings field" case
// TestMetadataReadFromRejectsUnregisteredSettingsField pins for "hide".
func TestMetadataReadFromRejectsLegacyDryRunDataPlaneSyntax(t1 *testing.T) {
	metadata := NewMetadata(ids.RepoId{})
	metadata.SettingSet.AddPrototype(
		"dry-run",
		&SettingDryRun{MutableConfigDryRun: &testMutableConfigDryRun{}},
	)

	_, err := metadata.ReadFrom(strings.NewReader("- _dry-run = true\n"))

	if err == nil {
		t1.Fatalf("expected an error for legacy data-plane \"- _dry-run = true\", got nil")
	}
}

// TestMetadataReadFromRejectsLegacyAllowDeletionDataPlaneSyntax is the
// same regression pin as above for "allow-deletion", which (unlike
// "dry-run") is registered unconditionally by MakeSettingSet, so no
// manual registration is needed here.
func TestMetadataReadFromRejectsLegacyAllowDeletionDataPlaneSyntax(t1 *testing.T) {
	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("- _allow-deletion = true\n"))

	if err == nil {
		t1.Fatalf("expected an error for legacy data-plane \"- _allow-deletion = true\", got nil")
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

	_, err := metadata.ReadFrom(strings.NewReader("- _dry-run = true\n"))
	t.AssertNoError(err)

	if metadata.TagSet.Len() != 0 {
		t.Errorf("expected no tags, got: %s", metadata.TagSet)
	}
}

// TestMetadataSettingsFieldReadFromAcceptsUnspacedLegacyForm pins that
// ReadFrom still accepts the pre-RFC-0015 unspaced form (`- _key=value`,
// no whitespace around "=") -- TrimSpace on both the key and value halves
// (metadata.go's addTagOrSettingsField) accepts the new spaced form
// unconditionally and the old unspaced form for free (a no-op trim when
// there's no whitespace to remove), so this is read-side back-compat
// verified explicitly rather than assumed.
func TestMetadataSettingsFieldReadFromAcceptsUnspacedLegacyForm(t1 *testing.T) {
	t := ui.MakeT(t1)

	const digest = "@blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz"

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("- _base=" + digest + "\n"))
	t.AssertNoError(err)

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	// Written form is always the new spaced canonical spelling,
	// regardless of which spelling was read.
	expected := "- _base = " + digest + "\n"

	if buf.String() != expected {
		t.Errorf("\nexpected: %q\n  actual: %q", expected, buf.String())
	}
}
