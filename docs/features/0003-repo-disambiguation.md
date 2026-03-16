---
status: accepted
date: 2026-03-15
promotion-criteria: all BATS tests pass with -override-xdg-with-cwd replaced by -repo_id .; -override-xdg-with-cwd fully removed from codebase; new tests cover explicit . selection, DODDER_REPO_ID env var, and / panic
---

# Repo Disambiguation

## Problem Statement

Dodder currently has one implicit repo location: the XDG user directory. To
operate on a CWD repo (`.dodder/` in `$PWD`), users must pass
`-override-xdg-with-cwd` at init time, which bakes the choice into the XDG
setup. There's no way for a single user session to address multiple repo
locations, and no consistent identifier scheme for selecting between them.

Madder blob stores solved this for blob storage with prefix-based IDs
(`.archive`, `/system`, `default`). Repos need the same — but with a key
constraint: there is at most one repo per location prefix (CWD, user, system),
so the prefix alone is the identifier. Remotes (unlimited cardinality) are
defined within repos and require separate infrastructure.

## Interface

### Repo Location Identifiers

A repo location is selected by a single-character prefix with no name portion:

| Identifier | Location | Resolution |
|---|---|---|
| *(empty)* | Auto | CWD if `.dodder/` exists in `$PWD`, otherwise `$XDG_DATA_HOME/dodder/` |
| `.` | CWD | `$PWD/.dodder/` — errors if not present (except `init`) |
| `/` | XDG system | Not yet implemented — panics with `Err501NotImplemented` |

Remotes (`/<repo-id>`) are deferred. They require user-config-wide repo
definitions that exist outside of individual repos.

### CLI Surface

**New flag**: `-repo_id <value>` on `repo_config_cli.Config` (global, available
to all dodder commands).

**New env var**: `DODDER_REPO_ID` — persistent default, overridden by the flag.
Useful for scoping a shell session to a specific repo location.

**Conflict rules**:
- `-repo_id` and `-override-xdg-with-cwd` are mutually exclusive — error if
  both set
- `-dir-dodder` remains as a raw path escape hatch, independent of `-repo_id`

### Future: Positional Arg Inline Switching

Following the madder blob store pattern, repo location could be accepted as a
positional argument interspersed with other args. For example, `dodder show .
one/two` would select the CWD repo and show zettel `one/two`. This adds parsing
complexity and ambiguity with existing positional args, so it's deferred.

## Implementation

### New Packages

**`xdg_location_type`** — Extracted from `blob_store_id.location`. Exports a
`Type` enum with values `Unknown`, `Cwd`, `XDGUser`, `XDGSystem`. Both
`blob_store_id` and the new `repo_id` package import this.

**`repo_id`** — New type with `Set(string)` accepting `.`, `/`, or empty string.
Stores an `xdg_location_type.Type`. No name portion — the prefix is the full
identifier. `String()` returns the prefix character (or empty for default/auto).

### Changes to Existing Code

**`blob_store_id`** — Replace internal `location` type with
`xdg_location_type.Type`. Public API unchanged, just the backing type.

**`repo_config_cli.Config`** — Add `RepoId repo_id.Id` field. Register
`-repo_id` flag. Add validation: error if both `RepoId` is set and
`OverrideXDGWithCwd` is true (on commands where both flags exist).

**`env_dir`** — Consume `RepoId` during XDG setup. When `.`, force CWD mode.
When `/`, panic not implemented. When empty, check for `.dodder/` in CWD first,
fall back to XDG user.

**`env_repo.Options`** — Add `RepoId` field alongside existing `BasePath`.

### Environment Variable

`DODDER_REPO_ID` read in `repo_config_cli` or `env_dir` as the default value for
the flag. The `-repo_id` flag overrides it.

### Removal of `-override-xdg-with-cwd`

Once `-repo_id .` is functional, `-override-xdg-with-cwd` is removed entirely:
the flag registration, `BigBang.OverrideXDGWithCwd` field, and all test usage
replaced with `-repo_id .`.

### TODOs

- `init` command should be split into `init` (XDG) and `init-cwd` —
  `-repo_id` should not be present on init at all
- `/<repo-id>` remote selection requires user-config-wide repo definitions
  outside of repos
- `/` (system repo) XDG path resolution

## Rollback Strategy

### Dual-Architecture Period

The new `-repo_id` flag and `DODDER_REPO_ID` env var are purely additive.
Existing flags (`-dir-dodder`, `-override-xdg-with-cwd`) continue to work
unchanged during development. The empty-default behavior (CWD if present, else
XDG user) matches current behavior — no existing workflows break.
`-override-xdg-with-cwd` is removed only after `-repo_id .` is verified.

### Promotion Criteria

- All existing BATS tests pass with `-override-xdg-with-cwd` replaced by
  `-repo_id .`
- `-override-xdg-with-cwd` fully removed from codebase (flag registration,
  `BigBang` field, all test usage)
- New tests cover: explicit `.` selection, `DODDER_REPO_ID` env var, `/` panic

### Rollback Procedure

Revert the commits. `-override-xdg-with-cwd` is restored since its removal is
part of the same change set. No persistent state affected — repo_id is a runtime
selection mechanism, not stored in configs or on disk.
