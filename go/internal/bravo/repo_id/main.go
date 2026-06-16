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
//     cannot yet resolve, so callers fail with a clear error instead of
//     madder's 501 panic for an unwired scope. Relaxed as madder wires
//     them (system scope: madder#230; multi-dot open path: madder#153).
//   - IsAuto distinguishes "no selector given" from scoped_id.IsEmpty()
//     (which is name-empty), so the init flow can default the bare
//     invocation to the cwd repo without clobbering an explicit scope.
package repo_id

import (
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// DefaultName is the name of the implicit repo selected when no -repo_id
// is given. Since the nameless `.`/`/` pins are dropped (scoped_id is
// name-required), every repo — including the default — lives under
// repos/<name>/; the default just uses this fixed name so its on-disk
// layout is deterministic.
const DefaultName = "default"

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
// never empty, because nameless repos no longer exist.
func EffectiveName(id scoped_id.Id) string {
	if name := id.GetName(); name != "" {
		return name
	}

	return DefaultName
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
// skip the repos/<name>/ nesting. cwdDepth/digest are dropped, which is
// safe because CheckSupported rejects multi-dot/system before resolution.
func EffectiveId(id scoped_id.Id) scoped_id.Id {
	return scoped_id.MakeWithLocation(EffectiveName(id), id.GetLocationType())
}

// CheckSupported rejects the FDR grammar this dodder prototype parses
// but cannot yet resolve, so a user gets a clear error rather than the
// madder 501 panic for an unwired scope:
//
//   - multi-dot cwd depth (`..name`): madder parses the depth and tags
//     discovered stores with it, but the open path ignores depth
//     (MakeDefaultAndInitialize's cwd branch roots at os.Getwd(), never
//     walks up cwdDepth parents) — tracked as madder#153. Single-dot
//     (depth 0, nearest) resolves today.
//   - system scope (`/name`, `//name`): never wired in madder
//     (madder#230) — no XDGSystem layout exists to resolve into, so
//     MakeDefaultAndInitialize would panic 501.
//
// FDR-0019 P2 pickup: when madder#153 lands, drop the cwd-depth>0 reject;
// when madder#230 lands, drop the XDGSystem reject. This is the single
// place that gates both, and every repo-opening path funnels through it
// (see CheckSupported callers), so each relaxation enables the scope
// uniformly.
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
