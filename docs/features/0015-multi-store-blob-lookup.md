---
date: 2026-05-13
promotion-criteria: every dodder read site that loads a
  content-addressed blob from `env_repo` goes through
  `env.GetReadBlobStore()` (backed by madder's Multi blob store in
  write-through mode) — no remaining
  `env_repo.GetDefaultBlobStore().MakeBlobReader` calls; writes
  still pin to `GetDefaultBlobStore()` by convention; the 5
  `clone_history_*` bats tests stay green after the stop-gap in
  `loadMutableConfigBlob` is rolled back in favor of the shared
  accessor (bats coverage is the integration gate; dedicated Go
  unit tests are deferred because building stub BlobStores would
  recreate what bats already exercises and madder owns Multi's
  correctness via its own test suite); FDR linked from dodder
  issue #196 and references madder issues #195 (Multi manpage)
  and #196 (two-pass optimization)
status: accepted
accepted: 2026-05-18
---

# Multi-Store Blob Lookup

> **Status (2026-05-18):** Accepted. Phases 1–4 shipped. All 23 reader
> call sites identified in the Phase 3 list, plus 5 additional sites
> surfaced during the Phase 3 caller audit (golf/sku_json_fmt/bookmark,
> romeo/local_working_copy/blob_tree_materializer, quebec/remote_transfer's
> inventory-list HasBlob check, lima/store_fs/main, lima/store_fs/file_encoder),
> migrated to `env.GetReadBlobStore()`. Writers untouched (25 sites).
> Go unit tests + 484 bats tests green. See FDR-0007's pattern: this
> remains a living document; future per-store probe optimizations
> ([madder#196](https://github.com/amarbel-llc/madder/issues/196))
> will land when madder picks an approach.

## Problem Statement

Dodder was originally built against a single madder blob store.
`env_repo.GetDefaultBlobStore()` returned *the* store; reads and writes
both went through it; there was no possibility of more than one madder
store being relevant to a given dodder operation.

Madder has since evolved. A madder `BlobStoreEnv` now enumerates every
`.madder/` directory it can reach during a CWD walk-up, registering
each as a separately-addressable store with a prefix encoding its
distance from CWD: `.default` is CWD's store, `..default` the parent's,
and so on. The ceiling
(`MADDER_CEILING_DIRECTORIES` /
[git's GIT_CEILING_DIRECTORIES][gitceiling]) bounds the walk.
`madder.cat` already takes advantage of the multi-store world: when the
default store misses, it iterates the remaining stores and reads from
the first one whose `HasBlob(blobId)` returns true.

Dodder has not followed suit. Every read site still calls
`env_repo.GetDefaultBlobStore().MakeBlobReader(blobId)`, treating
`.default` as if it were the singleton it used to be. That assumption
breaks the moment a single dodder process holds two `env_repo.Env`
instances rooted at different basePaths, which is exactly what
`clone`/`pull`/`push` do:

1. `Clone.Run`'s `OnTheFirstDay` initializes a **local** `env_repo`
   at process CWD. Local `.default` resolves to `CWD/.madder/`.
2. `MakeRemoteFromBlob` builds a **remote** `env_repo` with
   `basePath = realpath(./them)`. The remote's `.default` *should*
   resolve to `them/.madder/`.
3. But both env_repos share the same process: `MADDER_CEILING_DIRECTORIES`,
   `os.Getwd()`-derived state, etc. When the remote env_repo's
   `MakeBlobReader` runs, madder's walk-up enumerates `them/.madder/`
   as `.default` *and* `CWD/.madder/` as `..default`. Dodder's pin to
   `.default` is no longer unambiguous — the local clone's empty
   `.madder/` shadows the remote's populated one.

The concrete symptom (issue
[#196](https://github.com/amarbel-llc/dodder/issues/196)): clone's
`loadMutableConfigBlob` looks up the konfig blob in `.default`, which
resolves to `CWD/.madder/` (the fresh local clone, empty), even though
the actual remote `.madder/` has the blob. Five bats tests in the
`clone_history_*` cluster failed because of this.

The fix that landed in
[commit d0ceb5ac3](https://github.com/amarbel-llc/dodder/commit/d0ceb5ac3)
loosens **only** `loadMutableConfigBlob`: it tries `.default` first,
then iterates `GetDefaultBlobStoreAndRemaining()` gated by
`HasBlob(blobId)`, returning the first reader that opens. That restored
all 5 clone tests. It is a stop-gap — it fixes the specific call site
that the regression hit, but the same hazard exists at every other
`GetDefaultBlobStore().MakeBlobReader` site, and the next multi-repo
flow that touches one of those sites will reintroduce the bug.

### Why the test suite mostly hides this

Most existing bats tests run dodder against a single repo. There's no
ancestor `.madder/` to enumerate as `..default`, so `.default` is
unambiguous and reads succeed. The clone tests are unusual: they
deliberately construct two repos in the same `$BATS_TEST_TMPDIR`
subtree. They are the canary for the class of bug, not a special case.

## Design

### Read sites are content-addressed; writes are not

Every dodder lookup of a blob by `mad_domain_interfaces.MarklId` is a
read of *content-addressed* data: any blob store holding the blob is a
valid source. There is no semantic reason to prefer `.default` over
`..default` over an XDG store — the digest disambiguates. The only
reason `.default` was canonical is historical (the
single-store world).

Writes are different. Writing a blob is a decision about *where new
content lives*. Today every dodder writer routes to `.default`, which
is correct: writes should go to the repo's own primary store, not to
an inherited ancestor. The FDR does **not** change write semantics.
`GetDefaultBlobStore().MakeBlobWriter(...)` stays exactly as-is.

This asymmetry — multi-store reads, single-store writes — mirrors
what madder.cat already does for its read path.

### The shared helper

Back the read-fallback with madder's `Multi` blob store
([madder `go/internal/foxtrot/blob_stores/multi.go`][madder-multi];
documentation in progress at
[amarbel-llc/madder#195][madder-issue-195]). `Multi` already
implements ordered read fallback across a list of child stores;
dodder consumes it rather than reimplementing the loop.

Use **write-through mode**:

```go
multi, err := blob_stores.NewMulti(ctx).
    WriteTo(defaultStore).
    Read(remaining...).
    Build() // readFill defaults are fine; we never tee
```

Write-through (not mirror) because dodder pins writes to `.default`
by design. A caller that obtains the Multi and accidentally calls
`MakeBlobWriter` still lands in the default store — the type
enforces the FDR's "writes are placement decisions" rule rather
than relying on code review. (Mirror mode broadcasts writes to
every child; wrong for dodder.)

Single-store envs degrade cleanly:
`NewMulti(ctx).WriteTo(default).Read().Build()` succeeds — Multi's
`Build()` only requires a non-nil write store. Empty `Read(...)`
is permitted.

Expose the Multi as an accessor on `env_repo.Env`:

```go
// GetReadBlobStore returns a madder Multi blob store in
// write-through mode that reads from every enumerated store
// (default first, then walk-up ancestors and the XDG system
// store) and pins writes to the default store.
//
// Prefer this over GetDefaultBlobStore for any content-addressed
// read. Writes that intentionally pin to .default may still call
// GetDefaultBlobStore directly; using GetReadBlobStore is equally
// safe because Multi's write-through mode routes writes to the
// same default store.
func (env Env) GetReadBlobStore() mad_domain_interfaces.BlobStore
```

The accessor builds the Multi on every call. Construction is
cheap (a few struct fields, no I/O) and avoids cache-invalidation
bugs — `blobStoreEnv` is re-made during genesis, so caching the
Multi at `Env` construction time would hand callers stale stores.
Order: `defaultStore` first, then `remaining` in whatever order
`GetDefaultBlobStoreAndRemaining` returns it (madder owns that
ordering).

The return type is `mad_domain_interfaces.BlobStore` (Multi's
interface) rather than `blob_stores.BlobStoreInitialized` (a
struct wrapping `BlobStore` + `Path`). Multi doesn't have a
`Path`; the wrapping struct is for individual filesystem stores
that need provenance metadata. Reader call sites only need
`.MakeBlobReader(blobId)`, which is on `BlobStore`.

The call site reduces to standard `BlobStoreInitialized` use:

```go
blobReader, err := env.GetReadBlobStore().MakeBlobReader(blobId)
```

[madder-multi]: https://github.com/amarbel-llc/madder/blob/master/go/internal/foxtrot/blob_stores/multi.go
[madder-issue-195]: https://github.com/amarbel-llc/madder/issues/195

### Migration

Replace every existing
`env.GetDefaultBlobStore().MakeBlobReader(blobId)` with
`env.GetReadBlobStore().MakeBlobReader(blobId)` (matching for
accessor-chain variants like
`op.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(...)`).
Writes stay on `GetDefaultBlobStore()` by convention — the type
makes either choice safe, but explicit `GetDefaultBlobStore` on
writes documents intent.

Approximate scope: ~30 reader call sites across `internal/` (see
Key Files). Most are mechanical single-line changes; helpers that
take a `BlobStore` parameter migrate at the *caller* (caller
passes `env.GetReadBlobStore()` instead of
`env.GetDefaultBlobStore()`), not by changing the helper's
signature. The helper continues to accept a `BlobStore` and the
read-fallback travels with the value.

### Error surface

Multi returns `blob_io.ErrBlobMissing` on all-miss. The sentinel
is a struct, not a bare value:

```go
type ErrBlobMissing struct {
    BlobId domain_interfaces.MarklId
    Path   string
}
```

It implements `Is()` for `errors.Is` matching. Callers that
branch on missing-vs-IO-error get a typed sentinel they can
match cleanly. Callers that render to the user get the digest and
one path (the default store's path) — same useful diagnostic the
single-store world provided, in a typed envelope.

`Path` carries only the default store's path today. Surfacing
which fallback stores were tried is a madder concern; an upstream
TODO on `ErrBlobMissing` notes "add blob store" as a future field.
If diagnostic depth matters before that lands, dodder call sites
can log fallback paths through `ui.Debug` without changing the
error shape.

### Performance footnote

Multi's `MakeBlobReader` currently does two sequential calls per
child store on the miss path: `HasBlob(id)` then
`MakeBlobReader(id)`. For local filesystem stores the cost is two
stat-equivalents — invisible in practice. For remote backends it
becomes two round trips per probed store, which gets visible once
a fallback chain has multiple remotes.

Tracked upstream as
[amarbel-llc/madder#196][madder-issue-196] — the planned
direction is either a `PrefersProbe` opt-in flag or a unified
`TryOpen` primitive that returns `ErrBlobMissing` on miss
without a separate presence check. Dodder ships against today's
two-pass code; the optimization lands when madder picks an
approach.

[madder-issue-196]: https://github.com/amarbel-llc/madder/issues/196

### What does NOT change

- Write semantics. `GetDefaultBlobStore().MakeBlobWriter(...)` stays.
- The blob store enumeration order (`.default`, then walk-up). That's
  owned by madder.
- Madder's `BlobStoreEnv` API surface.
- The `MADDER_CEILING_DIRECTORIES` semantic (git-matching, blocks
  walk-up *above* the listed dir, not *at* it). The
  bats infrastructure's `DODDER_TEST_CEILING` /
  `MADDER_TEST_CEILING` per-call override hook stays.
- The blob_store_id prefix encoding (`.`, `..`, `/`, unprefixed).

### Why not also loosen writes

Routing writes through multiple stores is a different question — it
introduces "where does new content live in a multi-store world" as a
design problem (does a clone write to its own local, or back to the
source it cloned from?). That's not the bug
[#196](https://github.com/amarbel-llc/dodder/issues/196) reported,
and it has nontrivial consequences (push/pull semantics, garbage
collection, fsck across stores). Out of scope for this FDR. If a
future feature requires multi-store writes, it gets its own FDR.

## Implementation Phases

### Phase 1 — Stop-gap (Shipped)

`loadMutableConfigBlob` in `internal/november/store_config/persist.go`
has a private `openBlobReaderAcrossStores` helper that walks every
store. Closed [#196](https://github.com/amarbel-llc/dodder/issues/196).
All 480 bats tests pass.

This is intentionally narrow. It demonstrates the pattern but does
not generalize, and the same bug class lurks at ~30 other read sites.

### Phase 2 — Add the Multi-backed accessor on env_repo (Shipped)

1. `GetReadBlobStore()` added to `env_repo.Env`, returning a
   Multi built in write-through mode from
   `GetDefaultBlobStoreAndRemaining`. Built on every call rather
   than cached — genesis re-makes `blobStoreEnv` so a cached
   Multi would hand callers stale stores.
2. `loadMutableConfigBlob` now calls
   `env.GetReadBlobStore().MakeBlobReader(blobId)` directly. The
   private `openBlobReaderAcrossStores` helper is deleted.
3. Dedicated Go unit tests deferred: bats `clone.bats` (the
   `clone_history_*` cluster) exercises the multi-store fallback
   path end-to-end. Building stub BlobStores for a Go unit test
   would recreate what bats already covers, and madder owns
   Multi's correctness via its own test suite.

### Phase 3 — Migrate all reader call sites

Sweep the ~30 sites enumerated below, replacing
`GetDefaultBlobStore().MakeBlobReader` with
`GetReadBlobStore().MakeBlobReader`. Each migration is a one-line
change. Run the full bats suite (including the `clone_history_*`
cluster) after each batch.

Migration target list (read sites that pass `MakeBlobReader(blobId)`
on `GetDefaultBlobStore()`):

- `internal/hotel/inventory_list_coders/closet.go:296`
- `internal/hotel/env_lua/main.go:115`
- `internal/india/typed_blob_store/repo.go:30`
- `internal/india/typed_blob_store/tag.go:142,173`
- `internal/golf/blob_library/main.go:45`
- `internal/golf/sku_json_fmt/bookmark_url.go:25`
- `internal/golf/type_blobs/coder.go:82`
- `internal/lima/store_fs/file_encoder.go:76`
- `internal/lima/haustoria_orgmode/main.go:303`
- `internal/lima/haustoria_caldav/main.go:254`
- `internal/oscar/store/reference_discovery.go:119`
- `internal/oscar/store/field_writer.go:115`
- `internal/oscar/store/field_reader.go:79`
- `internal/oscar/store/lua.go:29`
- `internal/sierra/store_browser/main.go:134`
- `internal/sierra/remote_http/server_mcp.go:492`
- `internal/romeo/local_working_copy/format.go:746,782,822,1139,1175`
- `internal/uniform/commands_dodder/exec.go:97`
- `internal/uniform/commands_dodder/dormant_edit.go:133`
- `internal/november/store_config/persist.go:483` (in stop-gap form;
  reduces to `env.GetReadBlobStore().MakeBlobReader(blobId)` in Phase 2)

Sites that take a `BlobStore` parameter (e.g. `typed_blob_store/config.go`,
`typed_blob_store/tag.go` constructor params, `sku_fmt`) currently
get the default store and pass it down. Phase 3 migrates the
*caller*, not the typed helper. The helper continues to accept a
`BlobStore`; the caller passes `env.GetReadBlobStore()` instead of
`env.GetDefaultBlobStore()` and the read-fallback travels with the
value.

### Phase 4 — Writes stay pinned (no migration)

`MakeBlobWriter` sites stay on `GetDefaultBlobStore()` unchanged.
The Multi's write-through mode would also route writes to the
default store, but keeping `GetDefaultBlobStore` on writes
documents the placement decision at the call site.

### Phase 5 — Promotion

When every reader migrates, the FDR moves from `proposed` to
`accepted`. The stop-gap in `persist.go` collapses into a single
`env.GetReadBlobStore().MakeBlobReader` call (Phase 2 already did
this).

## Key Files

  ----------------------------------------------------------------------
  File                                                Role
  --------------------------------------------------- ------------------
  `go/internal/foxtrot/env_repo/main.go`              Add
                                                      `GetReadBlobStore`
                                                      method; build the
                                                      Multi from
                                                      `GetDefaultBlobStoreAndRemaining`.

  `go/internal/november/store_config/persist.go`      Stop-gap site;
                                                      collapsed in
                                                      Phase 2.

  (~30 reader call sites)                             Phase 3 migration
                                                      target list above.
  ----------------------------------------------------------------------

## Rollback Strategy

Phase 1 (stop-gap, shipped) is already revertable on its own by
restoring the single-store `GetDefaultBlobStore().MakeBlobReader`
call. The 5 `clone_history_*` tests would fail again.

Phase 2's `GetReadBlobStore` is additive; reverting it means
deleting the new accessor and restoring `loadMutableConfigBlob`
to its stop-gap form.

Phase 3 migrations are each one-line and individually revertable.

No on-disk format changes. No flake / dep changes. Suite passes
between every phase.

[gitceiling]: https://git-scm.com/docs/git#Documentation/git.txt-GITCEILINGDIRECTORIES
