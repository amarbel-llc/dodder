package orgie

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// TestMetadataWriteDataPlaneToSkipsOperationalPlane pins the dodder
// RFC-0015 followup: base-blob generation (WriteDataPlaneTo) must render
// ONLY the data plane (`-`/`!` lines) -- an operational-plane Setting
// (rendered as a `%` comment by ordinary WriteTo) must never appear in a
// data-plane-only render, since the organize-base-v1 blob's digest must
// not depend on the operational plane (cutting-garden RFC 0015, merged).
// Deliberately verified standalone with a synthetic operational-plane
// Setting rather than deferred to piece 4's end-to-end test: this
// mechanism's own payoff ("must not appear in the base") is otherwise
// unobservable until _dry-run/_allow-deletion are reclassified.
func TestMetadataWriteDataPlaneToSkipsOperationalPlane(t1 *testing.T) {
	t := ui.MakeT(t1)

	metadata := NewMetadata(ids.RepoId{})

	// Data-plane item: SettingBaseDigest implements SettingAsField
	// (IsSettingsField() == true), so it must survive both renders.
	baseDigest := &SettingBaseDigest{}
	errors.PanicIfError(baseDigest.Id.Set(
		"blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz",
	))
	metadata.SettingSet.AddPrototypeAndOption("base", baseDigest)

	// Operational-plane item: AddInertProse stores an unwrapped
	// SettingUnknown (no SettingAsField, no SettingWithKey), so
	// isSettingsField(o) is false for it -- exactly the "not a settings
	// field" case the data-plane filter must skip.
	metadata.SettingSet.AddInertProse("prose comment")

	var fullBuf, dataPlaneBuf strings.Builder

	_, err := metadata.WriteTo(&fullBuf)
	t.AssertNoError(err)

	_, err = metadata.WriteDataPlaneTo(&dataPlaneBuf)
	t.AssertNoError(err)

	full := fullBuf.String()
	dataPlane := dataPlaneBuf.String()

	if !strings.Contains(full, "_base =") {
		t1.Errorf("expected full render to contain the data-plane _base field, got %q", full)
	}

	if !strings.Contains(full, "% prose comment") {
		t1.Errorf("expected full render to contain the operational-plane comment, got %q", full)
	}

	if !strings.Contains(dataPlane, "_base =") {
		t1.Errorf("expected data-plane render to contain the _base field, got %q", dataPlane)
	}

	if strings.Contains(dataPlane, "prose comment") {
		t1.Errorf("expected data-plane render to OMIT the operational-plane comment, got %q", dataPlane)
	}
}
