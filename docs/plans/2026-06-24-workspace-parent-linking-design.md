# Workspace Parent Linking Design

## Status

**Path B landed.** When this design was first written, `init-workspace`
recorded no parent link at all. Since then a **path-based implicit parent**
shipped (see Landed Baseline), and then **path B of this design — pubkey pinning
+ verification — landed** (#287b): `init-workspace` pins the parent's pubkey
into the workspace config; `push`/`pull` verify the resolved parent against the
pin (hard-fail on mismatch); a legacy pin-less workspace confirm-pins on a TTY
or hard-fails non-interactively; `dodder set-parent` is the explicit
(re-)pin / migration command. **Path A remains future work** (post-soak): the
`%` sentinel, `der sync`, and the full scope-picker inference. The sections
below specify A; B is the pinning+verification subset that shipped.

## Problem

A workspace repo needs a first-class, ergonomic, *safe* link to its parent. Two
original motivations stand:

1. **`remote-add` is the wrong tool for a workspace parent.** It writes the
   remote as a **content object** (genre `Repo`, e.g.
   `[/them @blake2b256-... !toml-repo-local_override_path-v0]`) committed into
   the store. A later `push :z` (or any query catching it) would ship that
   pointer object **back to the parent**, which is nonsensical. A workspace's
   parent relationship is **infrastructure, not content**.
2. **The parent link should not be silently wrong.** The landed path-based
   parent (below) stores only a filesystem path with **no identity check** --
   if that path is later occupied by a *different* repo, push/pull operate
   against the wrong target with no warning. The remaining design adds
   identity pinning (by pubkey) and verification to close that gap.

## Landed Baseline (what exists on master today)

`init-workspace -experimental-repo` (the default) already creates a repo-backed
workspace linked to a parent, **path-based**:

- **`-parent <path>`** records a CWD-scoped parent; **omitting `-parent`**
  links the **home (XDG-user) repo** as parent. (`init_workspace.go`
  `resolveParentPath` / `runExperimentalRepo`.)
- The link is stored as a plain **`parent-path` string** in workspace config
  **V1** (`workspace_config_blobs.V1.ParentPath`). Empty `ParentPath` on a V1
  config means "home repo is the parent."
- **`push` / `pull` with no remote arg** resolve that stored path implicitly
  (`command_components_dodder.Remote.ResolveImplicitDirectPath` ->
  `GetParentPath()` -> `DirectPath`; empty path + V1 config -> home-repo
  remote). Covered by `workspace_repo_implicit_parent_push_pull`.
- The workspace's blob store is a **`TomlPointerV1`** pointing at the parent's
  default blob store (#200), and the parent's yin/yang zettel-id providers are
  linked so workspace-authored ids draw from the parent's space.
- A **sync checkpoint** (`sync-tai` / `sync-digest`, V1) is written by
  `UpdateSyncBaseline` after init/push/pull and read only by
  `check-workspace dirty`. #286 added a guard that refuses to advance it past a
  list whose blob is absent from the read store.

What the baseline does **not** have, and this design adds: **identity pinning +
verification**, the **`%` sentinel**, **`der sync`**, **scope-picker
inference**, and **`set-parent` / `-no-parent` / `info-workspace`** surfacing.

## Dependency

This design builds on **FDR-0019 (Scoped Repo Resolution)**, which is **landed
on master** (`docs/features/0019-scoped-repo-resolution.md`). The repo-id
selector is madder's `scoped_id.Id`, parsing the full grammar:

| Form | Scope |
|---|---|
| `name` | XDG user |
| `.name` | nearest-ancestor cwd repo |
| `..name` | Nth-ancestor cwd repo (dot-depth = N, **not** `../`) |
| `//name` | XDG system |
| `/name` | remote-first (system fallback) -- **`CheckSupported`-gated, see below** |

Concrete seams this feature reuses:

- **Scope enumeration:** `commands_dodder.listScopedRepos`
  (`go/internal/uniform/commands_dodder/repos_list.go`) returns
  `[]scopedRepo{Name, IsCwd}` with `Spelling()` (`.name` / `name`), across the
  cwd walk-up and the XDG-user scope. This is the picker's data source and the
  backing of `info-repo repos`.
- **Operate-path resolution:** `command_components_dodder.MakeOperateEnvDir`
  + `resolveCwdRepoAncestor` ->
  `directory_layout.ResolveNthAncestorMatch` (deepest-first, ceiling-bounded,
  skips non-matching ancestors). This is how a resolved id (including `..name`)
  becomes a rooted repo env -- the same machinery `%` resolution will reuse.
- **Scope gate:** `repo_id.CheckSupported` (`go/internal/bravo/repo_id/main.go`).

**The crucial overlap:** FDR-0019 explicitly **defers remote transport** --
`CheckSupported` rejects remote-first `/name` with "no remote transport yet."
Actually *operating against* another repo over a link is the gap this feature
fills. The workspace-parent push/pull **is** that deferred remote-transport
work, scoped to the single pinned parent. The two should cross-reference.

**Gaps in the landed enumeration this feature must extend:** `listScopedRepos`
today surfaces only `Name` + `IsCwd` (cwd vs user). The picker (below) needs
three things it does **not** yet provide -- each repo's **pubkey**, its
**directory path**, and the **system scope** -- plus **dot-depth** for the
multi-dot ordering. Surfacing those (a richer `scopedRepo`, or a sibling
enumerator) is in-scope implementation work for this feature, building on
`listScopedRepos` rather than replacing it.

## Design

A workspace's parent is recorded as **pinned-by-pubkey infrastructure** in
`.dodder-workspace` (never a content object), reached through a position-scoped
`%` sentinel in `push`/`pull`, with a new `der sync` for the everyday
bidirectional flow.

### Data Model -- pinning the parent's identity

The landed V1 config already records the parent *location* as a flat
`parent-path` string (plus `sync-tai`/`sync-digest`). This design adds the
parent's **identity** so the location can be verified, not just dereferenced.
The added fields (names/shape a design sketch, not yet built):

```toml
---
! toml-workspace_config-v2     # or a new version per the coder requirements
---

parent-path        = "/abs/path/to/parent"   # LANDED (empty => home repo)
sync-tai           = "..."                    # LANDED (checkpoint)
sync-digest        = "..."                    # LANDED (checkpoint)

parent-pubkey      = "ed25519_pub-ly04ujq4...gstrme4y"  # NEW: pinned identity
parent-original-id = "default"                # NEW: scoped_id spelling, guidance

[defaults]
```

- **`parent-pubkey`** -- authoritative identity. The pin is *truly pinned*:
  resolution opens the parent (via the existing `parent-path` / home-repo
  resolution) and verifies the live repo's pubkey against this field. A
  mismatch is a hard error, never a silent wrong-target. **This is the core
  safety delta over the landed baseline.**
- **`parent-original-id`** -- the human-meaningful `scoped_id` spelling at pin
  time (`name` user, `.name` cwd, `//name` system; dot-depth per FDR-0019).
  Guidance/debugging only: lets error messages and `der info-workspace` say
  `parent: default (user)` rather than only a path or pubkey. Not used for
  resolution. FDR-0019 collapses persisted dot-depth to a single dot, so a
  multi-dot `..name` pin is stored as `.name`.

The pin lives in **workspace metadata** (already the case for `parent-path`),
so it is **not a content object** -- no query can catch it and no push can ship
it back. The `remote-add`-as-content footgun stays closed.

**Versioning / migration.** The landed parent is V1. Adding pin fields is
additive but interacts with the V1/V2/V3 chain (V2 = haustoria, V3 = ignore),
so the exact carrier (extend V1's readers, or a new version) is an
implementation decision deferred to the a/b choice. **Existing V1 workspaces
have a `parent-path` but no `parent-pubkey`** -- the design must treat a
pin-less parent as "unverified, lazily pin on next resolve" (or leave it
path-only), NOT as an error; otherwise every already-created workspace breaks.

**`%` vs. bare push/pull.** The landed baseline already makes **bare**
`push`/`pull` target the parent path. The `%` sentinel (below) is therefore an
*explicit spelling* of that same target, valuable mainly once there are
**multiple** addressable parents/remotes or for disambiguation in the query
DSL -- it is not strictly required to reach the parent today. Whether `%` is
worth adding given bare push/pull already works is part of the a/b decision.

**Relationship to `sync-tai` / `sync-digest`:** those are the landed sync
*checkpoint* (where sync last reached, read by `check-workspace`), distinct
from the parent *target/identity* this section adds. They stay as-is.

### `%` Resolution and Verification

`%` is an **existing** doddish sentinel: in query-position it marks an object-id
virtual (`objectId.Virtual`, `internal/juliett/queries/object_id.go`). This design
adds a **second, position-scoped meaning**: a bare `%` in **remote-position**
(the `repo-id` argument of `push`/`pull`) resolves to the workspace's pinned
parent. The virtual-marker meaning in query-position is **unchanged** -- no
collision, because remote-position is resolved by repo-id lookup, not query
parsing.

When `push`/`pull` sees `%` in remote-position (or, equivalently, when bare
push/pull resolves the implicit parent today):

1. **Require a workspace.** Not in a workspace -> hard error
   (`% as a parent target is only valid inside a workspace`).
2. **Require a parent.** No `parent-path` and no home-repo fallback:
   - **TTY:** run the inference picker (below), record the parent, proceed.
   - **non-TTY:** hard error with guidance (`no parent configured; run der
     set-parent`).
3. **Resolve + open the parent** via the **landed** path resolution
   (`ResolveImplicitDirectPath`: `parent-path` -> `DirectPath`, or the home
   repo). Read its live pubkey.
4. **Verify (the safety delta).** If a `parent-pubkey` is pinned, compare it to
   the live pubkey:
   - match -> use as the remote for the transfer;
   - mismatch / unreachable -> hard error citing `parent-original-id` and
     tridex'd pubkeys; re-pin via `der set-parent`.
   - **no pin (legacy V1 workspace):** proceed unverified (back-compat), or
     lazily pin the observed pubkey -- a/b decision.

Implementation seam: the transfer machinery is already wired through
`ResolveImplicitDirectPath` + `MakeDirectRemoteFromPath`. The new work is
**(a)** an explicit `%` token that maps to that same resolution (vs. relying on
the bare-arg implicit path), and **(b)** the pubkey verification step after the
remote is opened. The transfer itself is unchanged.

### `der sync`

`der sync [query]` = **pull `%`, then push `%`**, with local conflict
resolution between. Merge the parent's changes **down** first; resolve any merge
conflicts in the local working copy; then push the merged, conflict-free result
**up**. If the pull leaves unresolved conflicts, `sync` stops before the push
(needs-merge) so a half-merged state is never pushed.

### Parent Inference (init-workspace / set-parent)

**Landed today:** `init-workspace -parent <path>` takes an explicit path, and
omitting `-parent` defaults to the home repo. There is **no scope enumeration
and no picker** -- the parent is either the given path or the home repo.

**This design adds** scope-aware inference. When a parent must be chosen (no
`-parent`, or `der set-parent` with no arg), query the scope-enumeration API
(`listScopedRepos`, extended -- see the enumeration-gap note above):

- **0 repos in scope** -> no pin written (workspace without a parent is valid;
  `%` later hard-fails).
- **exactly 1** -> pin it silently, no picker.
- **2+** -> interactive picker (TTY); **hard-fail non-TTY**
  (`multiple repos in scope; pass -parent`).

**Picker** (styled like the clown `resume` picker, which is `bubbles/list`, not
`huh.Select`): alt-screen, title, two-line item delegate, filtering, `enter`
selects / `esc`/`q`/`ctrl+c` aborts.

- **Item (one repo):** title = identifier, prefix-stripped; description =
  `<pubkey-tridex>  <location>  <directory>`.
- **Grouping by scope** (deliberate divergence from clown's flat list):
  section headers via a custom delegate. Groups ordered
  **internal/private -> cwd -> user -> system** (matching the `scoped_id`
  scopes; note `listScopedRepos` does not yet surface system -- adding it is
  part of this feature's enumeration extension). Within cwd, sort by dot-depth
  (`.name` before `..name` before `...name`), then by name. Other scopes sort
  by name. Names compared **without the scope prefix**.
- **Tridex scope = dialog-local:** abbreviate pubkeys to the shortest unique
  prefix *across the repos shown* (most-significant parts only).
- **Filtering** matches identifier + directory.

Exact visual styling (headers, spacing, colors) is refined in the build loop
under direct visual inspection rather than fully specified here.

### CLI Surface

Marked L (landed) / N (new in this design):

- **`der init-workspace [dir]`** -- **L:** `-parent <path>` and home-repo
  default. **N:** scope-id `-parent <scoped-id>` resolution + inference picker
  when ambiguous; `-no-parent` to opt out; write `parent-pubkey`.
- **`der set-parent [id]`** (**N**) -- set/change/clear the parent on an
  existing workspace. No arg -> picker; `<id>` -> resolve + pin; `-unset` ->
  remove the parent.
- **`der push [%] <query>` / `der pull [%] <query>`** -- **L:** bare push/pull
  already resolves the implicit parent. **N:** the explicit `%` spelling + the
  pubkey verification step.
- **`der sync [query]`** (**N**) -- pull then push against the parent, local
  conflict resolution between.
- **`der info-workspace`** -- **N:** surface the parent (`parent-original-id` +
  tridex'd pubkey + path) so the user sees the target without a transfer.

**Completion:** any new subcommands (`set-parent`, `sync`) -> add to
`zz-tests_bats/complete.bats` `complete_subcmd` (per `go/CLAUDE.md`).

## Error Handling

All failure modes are **hard errors with guidance**, never silent fallback:

- `%` outside a workspace.
- `%` with no pin, non-TTY.
- Pin pubkey mismatch / locator unreachable (cite `original-id` + tridex'd
  pubkeys; suggest `der set-parent`).
- 2+ in-scope repos, non-TTY, no `-parent`.

`der sync` halts before push if the pull leaves unresolved conflicts.

## Testing

BATS integration tests (the canonical sync surface, per `push.bats` /
`pull.bats`), covering:

- init-workspace inference: 0 / 1 / 2+ repos in scope (non-TTY paths;
  TTY-picker paths are exercised manually / via the build-loop inspection).
- `der push %` / `der pull %`: success, no-workspace error, no-pin non-TTY
  error, pubkey-mismatch error, unreachable-locator error.
- `der sync`: clean pull+push; pull-with-conflict halts before push.
- `der set-parent` / `-unset`; `der info-workspace` shows the pin.
- `%` virtual-marker in query-position still works unchanged alongside `%`
  remote-position (no regression).

## Open Decision: a vs. b (scope)

Reconciliation surfaced that the path-based parent already works, so #287 is no
longer greenfield. The remaining scope splits into:

- **(a) Full design:** identity pinning + `%` sentinel + `der sync` +
  scope-picker inference + `set-parent`/`-no-parent`/`info-workspace`. Largest;
  touches working push/pull/init code; needs the V1-workspace back-compat story
  for the new pin fields.
- **(b) Incremental, highest-value-first:** add **only** pubkey pinning +
  verification to the *existing* path-based parent (the safety win), and defer
  `%` / `der sync` / picker as separate follow-up issues. Smallest diff over
  landed code; closes the "silently wrong parent" gap without redesigning the
  working flow.

Recommendation pending user decision. The sections above specify (a); (b) is
the pinning+verification subset of the Data Model and `%` Resolution sections.

## Rollback / Dual-Architecture

The feature is **additive over the landed path-based parent**. `remote-add` and
existing remote objects are unchanged. Back-compat hinge: **existing V1
workspaces have `parent-path` but no `parent-pubkey`** -- they must keep working
(treated as unverified or lazily pinned), never error. Rollback = "don't pin /
don't use `%` / `der sync` / inference"; an unpinned path-based parent remains
fully functional. (FDR-0019's remote-first `/name` is a still-gated *selection*
syntax, not a path this feature relies on.)

## More Information

- **FDR-0019 (Scoped Repo Resolution)**, landed on master
  (`docs/features/0019-scoped-repo-resolution.md`) -- the grammar, scope
  enumeration, and `CheckSupported` gate this feature builds on. Its deferred
  remote-transport limitation is the gap this feature fills for the pinned
  parent.
- **Issue #287** -- this feature.
- **Issue #286** -- the sync-boundary durability bug that motivated the
  investigation leading here.
- The `%` virtual-object/virtual-tag marker
  (`internal/juliett/queries/object_id.go`, `objectId.Virtual`) -- the existing
  query-position meaning of `%` this feature must not disturb (remote-position
  only).

## Tuning Levers

- **Verification cost** -- always-verify the parent pubkey on every `%` use.
  Current: always-verify (safest). Signal to revisit: a remote/URI parent makes
  per-use verification a measurable round-trip; then consider verify-once-cache
  or a `-no-verify` escape hatch.
- **Tridex scope** -- dialog-local uniqueness. Current: unique across the
  repos shown. Signal to revisit: if pins/diagnostics outside the picker need
  stable abbreviations across invocations, consider a global tridex.
- **`der sync` order** -- pull-then-push with local conflict resolution.
  Current value chosen so conflicts surface locally before anything ships up.
  Signal to revisit: if a workflow needs push-first semantics (ship local work
  before merging down), revisit.
