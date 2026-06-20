// Package repo_id holds dodder-side helpers for the FDR-0019 repo
// location selector. The selector type itself is madder's scoped_id.Id
// (it parses the full FDR grammar — name, cwd dot-depth, system /
// remote-first spellings), so dodder no longer defines its own grammar.
// What remains here is dodder-specific policy that scoped_id, as a
// madder-agnostic type, does not encode:
//
//   - Nameless repos are dropped: scoped_id is name-required (per
//     Sasha's grammar ruling), so every repo — including the implicit
//     default — lives under repos/<name>/. EffectiveName / DefaultName /
//     CwdDefault give the auto invocation a fixed "default" name instead
//     of the retired nameless `.` cwd pin.
//   - CheckSupported gates the grammar this dodder prototype parses but
//     cannot yet resolve everywhere, so callers fail with a clear error
//     instead of a silent mis-resolution. Relaxed as the operate path
//     learns each scope (multi-dot: #281; system scope: #280).
//   - IsAuto distinguishes "no selector given" from scoped_id.IsEmpty()
//     (which is name-empty), so the init flow can default the bare
//     invocation to the cwd repo without clobbering an explicit scope.
package repo_id

import (
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// DefaultName is the name of the implicit repo selected when no -repo_id
// is given. It delegates to madder's scoped_id.DefaultName so the two
// repos share one source of truth (dodder#274). Since the nameless
// `.`/`/` pins are dropped (scoped_id is name-required), every repo —
// including the default — lives under repos/<name>/; the default just uses
// this fixed name so its on-disk layout is deterministic.
const DefaultName = scoped_id.DefaultName

// IsAuto reports that no repo was explicitly selected: the zero-value id
// (Unknown location, empty name) left in place when neither -repo_id nor
// DODDER_REPO_ID is set. This is deliberately narrower than
// scoped_id.IsEmpty(), which is also true for a nameless scope pin — an
// explicit scope is NOT auto.
func IsAuto(id scoped_id.Id) bool {
	return id.GetLocationType() == scoped_id.LocationTypeUnknown &&
		id.GetName() == ""
}

// EffectiveName returns the repos/<name>/ path segment a resolved id
// nests under: the explicit name, or DefaultName for the auto id. It is
// never empty, because nameless repos no longer exist. Delegates to
// madder's scoped_id.EffectiveName — the single source of truth both repos
// share so the resolution cannot diverge (dodder#274).
func EffectiveName(id scoped_id.Id) string {
	return scoped_id.EffectiveName(id)
}

// CwdDefault is the cwd-rooted default repo id (`.default`), used by the
// init flow to create or address the per-directory repo when no -repo_id
// is given. It replaces the dropped nameless `.` cwd pin.
func CwdDefault() scoped_id.Id {
	return scoped_id.MakeWithLocation(DefaultName, scoped_id.LocationTypeCwd)
}

// EffectiveId returns id with its name forced to EffectiveName (so the
// auto/zero id becomes the named "default") while preserving its
// location, for sites that hand a scoped_id to madder's
// MakeDefaultAndInitialize — which derives Config.RepoName from the id's
// name. Without this, the auto id (name "") would set RepoName="" and
// skip the repos/<name>/ nesting. Delegates to madder's
// scoped_id.EffectiveId (dodder#274). cwdDepth/digest are dropped, which
// is safe because CheckSupported rejects multi-dot/system before
// resolution; the depth-preserving variant lands here when the multi-dot
// operate path does (#281).
func EffectiveId(id scoped_id.Id) scoped_id.Id {
	return scoped_id.EffectiveId(id)
}

// CheckSupported rejects the FDR grammar this dodder prototype parses but
// cannot yet resolve everywhere, so a user gets a clear error rather than
// a silent mis-resolution or a madder 501:
//
//   - multi-dot cwd depth (`..name`): madder#153 wired the literal cwdDepth
//     walk-up into MakeDefaultAndInitialize (the init / info-repo / serve
//     paths), but the operate path (MakeLocalWorkingCopy ->
//     env_dir.MakeDefault) is name-only and ignores depth, so a multi-dot
//     id would silently resolve to the nearest same-named ancestor there.
//     Kept until the operate path resolves depth — tracked as #281.
//     Single-dot (depth 0, nearest) resolves today.
//   - system scope (`/name`, `//name`): XDGSystem is not wired into
//     MakeDefaultAndInitialize (it panics 501), and the operate path drops
//     the location too — tracked as #280.
//
// This is the single place that gates both, and every repo-opening path
// funnels through it (see CheckSupported callers), so each relaxation
// enables the scope uniformly once its resolution lands.
func CheckSupported(id scoped_id.Id) (err error) {
	if id.GetCwdDepth() > 0 {
		err = errors.Errorf(
			"repo_id %q: cwd dot-depth > 1 is not yet implemented",
			id.String(),
		)
		return err
	}

	if id.GetLocationType() == scoped_id.LocationTypeXDGSystem {
		err = errors.Errorf(
			"repo_id %q: system scope is not yet resolvable",
			id.String(),
		)
		return err
	}

	return err
}
