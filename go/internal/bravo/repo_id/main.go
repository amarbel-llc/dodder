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
//     cannot yet resolve, so callers fail with a clear error instead of a
//     silent mis-resolution. Relaxed as each scope's resolution lands; only
//     remote-first `/name` remains gated (no remote transport). System
//     `//name` (#280) and multi-dot cwd `..name` (#281) now resolve.
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
// auto/zero id becomes the named "default") while preserving its location
// AND cwd dot-depth, for sites that hand a scoped_id to madder's
// MakeDefaultAndInitialize — which derives Config.RepoName from the id's
// name and roots a cwd id at its literal Nth parent (echo/env_dir's
// resolveCwdAncestorOrError). Without the name force, the auto id (name "")
// would set RepoName="" and skip the repos/<name>/ nesting; without the
// depth restore, scoped_id.EffectiveId's flattening of cwdDepth to 0 would
// collapse `..name` back to the literal cwd, so the literal-init paths
// (genesis, MakeEnvRepo) would mis-root a multi-dot repo at depth 0 (#281).
func EffectiveId(id scoped_id.Id) scoped_id.Id {
	return scoped_id.EffectiveId(id).WithCwdDepth(id.GetCwdDepth())
}

// CheckSupported rejects the pieces of the FDR grammar this dodder
// prototype parses but cannot yet resolve, so a user gets a clear error
// rather than a silent mis-resolution. Only one spelling remains gated:
//
//   - remote-first system spelling (`/name`): scoped_id parses both `/name`
//     and `//name` as XDGSystem, distinguished by IsRemoteFirst. `/name`
//     means "consult the repo's remotes first, fall back to the
//     system-scoped name" — but dodder has no remote transport yet, and we
//     can't know whether `name` is a defined remote before opening the
//     repo, so we reject it rather than silently treat it as system (the
//     FDR-0019 remote-transport limitation). Use `//name` for the system repo.
//
// Both cwd scopes now resolve everywhere. Single-dot `.name` is the
// nearest-ancestor walk; multi-dot `..name` (cwdDepth > 0) is the Nth
// same-named ancestor — the literal-init paths (genesis, MakeEnvRepo) root
// it at the literal Nth parent via EffectiveId's preserved depth, and the
// nearest-operate paths (MakeLocalWorkingCopy, serve, info) resolve it
// store-aware via directory_layout.ResolveNthAncestorMatch (#281). Forced
// system scope `//name` roots at the system root (#280). Every repo-opening
// path funnels through CheckSupported, so a relaxation here enables its
// scope uniformly once resolution lands behind it.
func CheckSupported(id scoped_id.Id) (err error) {
	if id.GetLocationType() == scoped_id.LocationTypeXDGSystem &&
		id.IsRemoteFirst() {
		err = errors.Errorf(
			"repo_id %q: remote-first `/name` is not yet resolvable "+
				"(no remote transport); use `//name` for the system-scoped repo",
			id.String(),
		)
		return err
	}

	return err
}
