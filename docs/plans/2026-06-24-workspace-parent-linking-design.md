# Workspace Parent Linking Design

## Problem

`der init-workspace` creates a workspace, but does **not** record any link to a
parent repo -- it only writes query defaults to `.dodder-workspace`. Syncing a
workspace repo back to its parent therefore has no first-class path. The only
existing mechanism, `remote-add`, has two problems for this use case:

1. It writes the remote as a **content object** (genre `Repo`, e.g.
   `[/them @blake2b256-... !toml-repo-local_override_path-v0]`) committed into
   the store. A later `push :z` (or any query catching it) would ship that
   pointer object **back to the parent**, which is nonsensical.
2. It requires a manual, explicit step that users do not expect after
   `init-workspace` (the common assumption is that init-workspace already links
   the parent).

A workspace's parent relationship is **infrastructure, not content**. Modeling
it as a content object is the root footgun.

## Dependency

This design builds on the **repo-id resolution / pinning** layer (in-flight on
another branch/host; not present in this worktree at design time). It assumes a
scope-enumeration API that, given the current invocation, returns the set of
in-scope repos tagged by `LocationType` (cwd / user / system, per
`internal/_/blob_store_id/location.go`), with cwd repos carrying their `../`
directory distance. The inference rule (below) is specified against that assumed
API. **This feature cannot land until that layer exists.**

## Design

A workspace's parent is recorded as **pinned-by-pubkey infrastructure** in
`.dodder-workspace` (never a content object), reached through a position-scoped
`%` sentinel in `push`/`pull`, with a new `der sync` for the everyday
bidirectional flow.

### Data Model -- the `[parent]` pin

`.dodder-workspace` (workspace config) gains a `[parent]` section:

```toml
---
! toml-workspace_config-v1
---

[parent]
pubkey        = "ed25519_pub-ly04ujq4tz747jpk5059rxvx4g6lz5a22fpg8zgm7hgz0af7d4gstrme4y"
locator       = "/Users/sfriedenberg/.local/share/dodder/repos/default"
location-type = "user"      # cwd | user | system
original-id   = "default"   # scope-prefixed id as resolved at pin time

[defaults]
```

- **`pubkey`** -- authoritative identity. The pin is *truly pinned*: `%`
  resolution opens the locator and verifies the live repo's pubkey against this
  field. A mismatch is a hard error, never a silent wrong-target.
- **`locator`** -- concrete path/URI used to reach the parent.
- **`location-type`** -- which scope bucket the parent was resolved in.
  Advisory (for re-resolution / diagnostics); `pubkey` is authoritative.
- **`original-id`** -- the human-meaningful, **scope-prefixed** id exactly as
  the resolution layer reported it at pin time (carrying the `.` / `..` / `/` /
  none prefix from the `blob_store_id` location model). Pure guidance/debugging:
  it lets error messages and `der info-workspace` say `parent: default (user)`
  rather than only a raw path or pubkey. Not used for resolution.

This is **not a content object**: it lives in workspace metadata, so no query
can catch it and no push can ship it back. The footgun is closed by
construction.

**Versioning:** adding `[parent]` is additive. Modeled as a new field on the
workspace-config blob, with a version bump if the coder requires it (per the
repo's horizontal-versioning `*_blobs` pattern).

**Relationship to the old `sync-tai` / `sync-digest`:** those were a sync
*checkpoint* (where sync last reached), distinct from the *target*. This design
records only the target (`[parent]`). Whether `der sync` later reintroduces a
checkpoint field is deferred (YAGNI until the sync engine needs it).

### `%` Resolution and Verification

`%` is an **existing** doddish sentinel: in query-position it marks an object-id
virtual (`objectId.Virtual`, `internal/kilo/queries/object_id.go`). This design
adds a **second, position-scoped meaning**: a bare `%` in **remote-position**
(the `repo-id` argument of `push`/`pull`) resolves to the workspace's pinned
parent. The virtual-marker meaning in query-position is **unchanged** -- no
collision, because remote-position is resolved by repo-id lookup, not query
parsing.

When `push`/`pull` sees `%` in remote-position:

1. **Require a workspace.** Not in a workspace -> hard error
   (`% as a parent target is only valid inside a workspace`).
2. **Require a pin.** No `[parent]`:
   - **TTY:** run the inference picker (below), write the pin, proceed.
   - **non-TTY:** hard error with guidance (`no parent configured; run der
     set-parent`).
3. **Open `locator`, read the live repo pubkey.**
4. **Always verify.** Compare live pubkey vs pinned `pubkey`:
   - match -> use as the remote for the transfer;
   - mismatch / locator unreachable -> hard error citing `original-id` and
     tridex'd pubkeys; re-pin via `der set-parent`.

Implementation seam: `%` only changes how the `remote` is constructed in
`push.go` / `pull.go` -- intercept `%` in remote-position **before** the
stored-object lookup (`local.GetObjectFromObjectId`) and synthesize the remote
from the verified pin. The entire transfer machinery is otherwise unchanged.

### `der sync`

`der sync [query]` = **pull `%`, then push `%`**, with local conflict
resolution between. Merge the parent's changes **down** first; resolve any merge
conflicts in the local working copy; then push the merged, conflict-free result
**up**. If the pull leaves unresolved conflicts, `sync` stops before the push
(needs-merge) so a half-merged state is never pushed.

### Parent Inference (init-workspace / set-parent)

When a parent must be chosen (no `-parent`, or `der set-parent` with no arg),
query the scope-enumeration API:

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
  **internal/private -> cwd -> user -> system**. Within cwd, sort by directory
  distance (`.` < `..` < `...` ...), then by name. Other scopes sort by name.
  Names compared **prefix-stripped**.
- **Tridex scope = dialog-local:** abbreviate pubkeys to the shortest unique
  prefix *across the repos shown* (most-significant parts only).
- **Filtering** matches identifier + directory.

Exact visual styling (headers, spacing, colors) is refined in the build loop
under direct visual inspection rather than fully specified here.

### CLI Surface

- **`der init-workspace [dir]`** -- gains inference. No `-parent` -> run the
  rule above. `-parent <id>` -> resolve that scope-prefixed id and pin it.
  `-no-parent` -> explicitly opt out.
- **`der set-parent [id]`** (new) -- set/change/clear the pin on an existing
  workspace. No arg -> picker; `<id>` -> resolve+pin; `-unset` -> remove the
  `[parent]` pin.
- **`der push % <query>` / `der pull % <query>`** -- `%` in remote-position
  resolves to the verified pin; everything else unchanged.
- **`der sync [query]`** (new) -- pull `%` then push `%` (above).
- **`der info-workspace`** -- surfaces the pin (`original-id` + tridex'd pubkey
  + locator) so the user can see what `%` targets without a transfer.

**Completion:** `set-parent` and `sync` are new subcommands -> add to
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

## Rollback / Dual-Architecture

The feature is **purely additive**. `remote-add` + explicit `der push /name`
are unchanged; workspaces without a `[parent]` behave exactly as today. There is
no migration of existing repos. Rollback = "don't use `%` / `der sync` /
inference"; the dual architecture (explicit remotes vs. pinned parent) coexists
indefinitely.

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
