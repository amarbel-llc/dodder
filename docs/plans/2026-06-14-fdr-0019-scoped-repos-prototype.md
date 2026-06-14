# FDR-0019 Scoped Repo Resolution --- Dodder-Side Prototype

Status: prototype (dodder-only, ahead of the madder env_dir.RepoId
change FDR-0019 calls for). See
`docs/features/0019-scoped-repo-resolution.md` for the full design.

## Why a dodder-only prototype

The full FDR routes the repo-id grammar (name + scope prefix +
dot-depth) and the `repos/<name>/` layout through madder's
`env_dir.RepoId`, which dodder consumes. Madder's `RepoId` is still the
FDR-0003 location-only selector (`.`, `/`, empty) and lives in a
separate repo. This prototype implements the user-visible slice
entirely inside dodder so the feature can be exercised before the
madder change lands, then folded onto the real type later.

## What it does

- **Grammar.** New `internal/bravo/repo_id.Id` parses the FDR grammar:
  `name` (user), `.name` / `..name` (cwd, dot-depth), `//name`
  (system), `/name` (system; remote-first not implemented), `~name`
  (parse-only user alias), plus the legacy nameless `.` / `/` / empty.
  `String()` renders the canonical single-dot form. Covered by
  `repo_id` unit tests.
- **Layout.** `env_dir.NestUnderRepoName` appends `repos/<name>/` to
  the dodder XDG category dirs (data/config/state/cache/runtime), so a
  named repo's whole metadata tree --- config-seed, object index,
  inventory-list log, lock --- nests automatically. Wired into the
  three env-construction sites: `MakeEnvRepo`, `OnTheFirstDay`
  (genesis), and `MakeLocalWorkingCopy`.
- **CLI / env.** The global `-repo_id` flag and `DODDER_REPO_ID` accept
  the full grammar; `dodder init -repo_id <name> ...` creates a named
  repo; reads address it by the same id.
- **Shared blob store.** Genesis tolerates a pre-existing default blob
  store config (idempotent write), so named repos in one scope share
  madder's content-addressed blob pool while keeping independent
  metadata/index/identity.

## Deliberate limitations (vs. the full FDR)

- **Blobs are shared, not isolated.** Madder re-derives the blob-store
  XDG via `CloneWithUtilityName`, discarding any suffix applied
  dodder-side, so `repos/<name>/` nesting reaches only the dodder
  metadata tree. Full per-repo blob isolation needs the madder change.
- **Empty id = legacy user scope**, not the FDR's "nearest cwd repo on
  the walk-up, else user `default`". This keeps every existing
  single-repo tree resolving unchanged (no fixture regen).
- **Single-dot cwd depth only.** `..name` (depth > 1) parses but is
  rejected at resolution by `Id.CheckPrototypeSupported`.
- **`/name` resolves straight to system scope**; the remote-first
  lookup is out of scope (no remote transport yet).
- No migration command, no `info-repo repos` listing, no MCP
  `repo_id` dimension --- those layer on once the grammar is real.

## Path to the full FDR

1. Land the grammar + `repos/<name>/` layout on madder's
   `env_dir.RepoId`; `just go/update-flake-input madder`.
2. Replace `repo_id.Id` with the madder type (or have it delegate),
   dropping `CheckPrototypeSupported` once the walk-up resolves
   multi-dot depth.
3. Nest the blob-store XDG too (full isolation), add
   `migrate-repo-layout`, `info-repo repos`, MCP `repo_id` + resource
   URI segment, and completions.

## Tests

- `internal/bravo/repo_id/main_test.go` --- grammar parse/canonical
  round-trip, error cases, depth gate.
- `zz-tests_bats/current_version/scoped_repos.bats` --- two
  user-scoped named repos isolated, `DODDER_REPO_ID` addressing,
  multi-dot rejection.
