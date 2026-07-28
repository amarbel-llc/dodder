package orgie

import (
	"errors"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// TestMetadataReadFromParsesNewDirectiveSyntaxRoundTrip pins RFC 0015's
// (merged) core new syntax: `%:<name> = value`, colon adjacent to `%`,
// resolved against the prototype registry and round-tripping back out
// in the same canonical spelling. Uses SettingBooleanFlag (no
// SettingAsField method, matching checkin's real "delete" flag) so this
// test stays valid regardless of piece 4's later dry-run/allow-deletion
// reclassification.
func TestMetadataReadFromParsesNewDirectiveSyntaxRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	var flagValue bool

	metadata := NewMetadata(ids.RepoId{})
	metadata.SettingSet.AddPrototype("flag", SettingBooleanFlag{Value: &flagValue})

	_, err := metadata.ReadFrom(strings.NewReader("%:flag = true\n"))
	t.AssertNoError(err)

	if !flagValue {
		t1.Errorf("expected flagValue to be true after parsing %%:flag = true")
	}

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	expected := "%:flag = true\n"

	if buf.String() != expected {
		t.Errorf("\nexpected: %q\n  actual: %q", expected, buf.String())
	}
}

// TestMetadataReadFromDirectivePresenceOnlyDefaultsTrue pins RFC 0015's
// presence-only boolean spelling: a bare "%:name" (no "=" at all) sets
// the directive true, same as an explicit "%:name = true".
func TestMetadataReadFromDirectivePresenceOnlyDefaultsTrue(t1 *testing.T) {
	t := ui.MakeT(t1)

	var flagValue bool

	metadata := NewMetadata(ids.RepoId{})
	metadata.SettingSet.AddPrototype("flag", SettingBooleanFlag{Value: &flagValue})

	_, err := metadata.ReadFrom(strings.NewReader("%:flag\n"))
	t.AssertNoError(err)

	if !flagValue {
		t1.Errorf("expected flagValue to be true after parsing presence-only %%:flag")
	}
}

// TestMetadataReadFromParsesNamespacedDirectiveRoundTrip pins RFC 0015's
// namespace-optional routing (piece 3): a `%:<command>/<name>`
// directive resolves against a prototype the DRIVING COMMAND registers
// via RegisterNamespaced (checkin.go's real future call site, e.g.
// `%:checkin/delete = true`), not a bare harness-level name -- reusing
// the SAME flat SettingSet.prototype map SetDirective already resolves
// bare directives against, keyed here under "namespace/name".
func TestMetadataReadFromParsesNamespacedDirectiveRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	var flagValue bool

	metadata := NewMetadata(ids.RepoId{})
	metadata.SettingSet.RegisterNamespaced(
		"checkin", "delete",
		SettingBooleanFlag{Value: &flagValue},
	)

	_, err := metadata.ReadFrom(strings.NewReader("%:checkin/delete = true\n"))
	t.AssertNoError(err)

	if !flagValue {
		t1.Errorf("expected flagValue to be true after parsing %%:checkin/delete = true")
	}

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	expected := "%:checkin/delete = true\n"

	if buf.String() != expected {
		t.Errorf("\nexpected: %q\n  actual: %q", expected, buf.String())
	}
}

// TestSettingSetSetDirectiveRejectsUnrecognizedName pins RFC 0015's
// unrecognized-bare-directive rule at the SettingSet level directly
// (bypassing ReadFrom's ohio-driven line dispatch, which re-wraps
// whatever error a line handler returns into ohio's own
// ErrExhaustedFuncSetStringersLine -- a generic "a line handler failed"
// signal with no typed-error passthrough by design, so the specific
// ErrUnrecognizedDirective is only inspectable pre-wrap, at this level):
// unlike prose (always fine) or an unrecognized `- _key = value`
// data-plane field (a legitimate no-op,
// TestMetadataReadFromNoOpsOnEntirelyUnknownSettingsFieldKey), an
// unrecognized `%:` directive is a loud error -- directives are
// behavior-bearing by construction, so silently ignoring one would
// silently skip behavior the user asked for.
func TestSettingSetSetDirectiveRejectsUnrecognizedName(t1 *testing.T) {
	ocs := MakeSettingSet(nil)

	err := ocs.SetDirective("bogus = true")

	if err == nil {
		t1.Fatalf("expected an error for an unrecognized bogus directive, got nil")
	}

	if !errors.Is(err, ErrUnrecognizedDirective{}) {
		t1.Errorf("expected ErrUnrecognizedDirective, got: %v", err)
	}
}

// TestMetadataReadFromRejectsUnrecognizedDirective is the ReadFrom-level
// integration check for the same rule: an unrecognized %:bogus directive
// must surface SOME error (the specific type is verified directly above,
// against ohio's line-dispatch wrapping -- see that test's comment).
func TestMetadataReadFromRejectsUnrecognizedDirective(t1 *testing.T) {
	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("%:bogus = true\n"))

	if err == nil {
		t1.Fatalf("expected an error for an unrecognized %%:bogus directive, got nil")
	}
}

// TestMetadataReadFromLegacyDryRunAliasNoOpsWhenUnrecognized pins the
// dodder-local back-compat exception to the rule above: `% dry-run:true`
// predates the error-on-unrecognized rule (dry-run had no settings-field
// spelling before dodder#374(c) existed), and reading it in a context
// where "-dry-run" wasn't passed on the CLI (so "dry-run" was never
// registered) must remain a silent no-op, exactly as it always has --
// pinned at the bats level by
// organize_dry_run_legacy_comment_alias_still_accepted. Regression test:
// an earlier draft of this change routed the legacy alias through the
// SAME strict path as a genuine %: directive, which would have broken
// that bats test by turning the no-op into a hard parse error.
func TestMetadataReadFromLegacyDryRunAliasNoOpsWhenUnrecognized(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	_, err := metadata.ReadFrom(strings.NewReader("% dry-run:true\n"))
	t.AssertNoError(err)

	if metadata.TagSet.Len() != 0 {
		t1.Errorf("expected no tags, got: %s", metadata.TagSet)
	}
}

// TestMetadataInertProsePreservesColonContent pins the fix for a latent
// bug in the pre-RFC-0015 design: every "%"-line used to be parsed as
// key:value via strings.Cut, so genuine prose containing a colon (e.g.
// "% meeting at 3:00") silently lost everything after the first colon
// (it became an unrecognized "key" with an empty value). Under the
// two-plane model, bare "% <prose>" is NEVER parsed for structure --
// verified here with prose that would have been mangled under the old
// design.
func TestMetadataInertProsePreservesColonContent(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	const line = "% meeting at 3:00\n"

	_, err := metadata.ReadFrom(strings.NewReader(line))
	t.AssertNoError(err)

	var buf strings.Builder
	_, err = metadata.WriteTo(&buf)
	t.AssertNoError(err)

	if buf.String() != line {
		t.Errorf("\nexpected: %q\n  actual: %q", line, buf.String())
	}
}
