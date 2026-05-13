---
date: 2026-05-13
promotion-criteria: every dodder read site that loads a
  content-addressed blob from `env_repo` goes through a single shared
  multi-store helper (no remaining `env_repo.GetDefaultBlobStore().MakeBlobReader`
  calls); writes still pin to `GetDefaultBlobStore()` by design; the
  helper has unit coverage exercising default-hit, fallback-hit, and
  all-miss; the 5 clone_history_* bats tests stay green after the
  stop-gap in `loadMutableConfigBlob` is rolled back in favor of the
  shared helper; FDR linked from issue #196
status: proposed
---

# Multi-Store Blob Lookup

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

Add one read-side helper to `env_repo.Env`:

```go
// OpenBlobReader returns a reader for the content-addressed blob,
// searching the default store first and then every remaining
// enumerated store. Returns the default store's error if no store
// holds the blob, so diagnostics keep pointing at the canonical
// local path.
func (env Env) OpenBlobReader(
    blobId mad_domain_interfaces.MarklId,
) (mad_domain_interfaces.BlobReader, error)
```

Implementation mirrors the
[`Cat.blobFromRemainingStores`](https://github.com/amarbel-llc/madder/blob/main/internal/india/commands/cat.go)
pattern: try `.default`, then iterate `GetDefaultBlobStoreAndRemaining`'s
`remaining` map gated by `HasBlob`. The stop-gap's
`openBlobReaderAcrossStores` in
`internal/november/store_config/persist.go` becomes
`env.OpenBlobReader` and the call site reduces to:

```go
blobReader, err := store.envRepo.OpenBlobReader(blobId)
```

### Migration

Replace every existing
`env.GetDefaultBlobStore().MakeBlobReader(blobId)` with
`env.OpenBlobReader(blobId)` (matching for accessor-chain variants
like `op.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(...)`).
Writes stay as-is.

Approximate scope: ~30 reader call sites across `internal/` (see Key
Files). Most are mechanical single-line changes; a few thread a
`BlobStore` value through a helper and would migrate by switching the
helper's parameter from `BlobStoreInitialized` to `env_repo.Env` (or
by giving the helper a `BlobReader`-returning accessor).

### Error surface

`OpenBlobReader` returns the default store's error on all-miss. The
existing single-store error message ("Blob with id ... does not exist
locally: <path>") is the most useful diagnostic — it points at the
file the user would look at first. Surfacing fallback-store paths in
the error would clutter the message for the common case where the
blob is genuinely missing.

If diagnostic depth proves valuable later, the helper can grow an
options bag (`OpenBlobReaderOptions{IncludeFallbackPaths: true}`)
without breaking call sites.

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
store. Closed `#196`. All 480 bats tests pass.

This is intentionally narrow. It demonstrates the pattern but does
not generalize, and the same bug class lurks at ~30 other read sites.

### Phase 2 — Lift the helper onto env_repo

1. Move `openBlobReaderAcrossStores` from
   `store_config/persist.go` onto `env_repo.Env` as
   `OpenBlobReader`.
2. Update `loadMutableConfigBlob` to call `env.OpenBlobReader`
   directly. The private helper is deleted.
3. Add a unit test for `OpenBlobReader` covering default-hit,
   fallback-hit, and all-miss (error path preserves the default
   store's error message).

### Phase 3 — Migrate all reader call sites

Sweep the ~30 sites enumerated below, replacing
`GetDefaultBlobStore().MakeBlobReader` with `OpenBlobReader`.
Each migration is a one-line change. Run the full bats suite
(including the `clone_history_*` cluster) after each batch.

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
  reduces to `env.OpenBlobReader(blobId)` in Phase 2)

Sites that take a `BlobStore` parameter (e.g. `typed_blob_store/config.go`,
`typed_blob_store/tag.go` constructor params, `sku_fmt`) currently
get the default store and pass it down. Phase 3 migrates the
*caller*, not the typed helper. The helper continues to accept a
`BlobStore`; the caller resolves it through `OpenBlobReader`.

### Phase 4 — Writes stay pinned (no migration)

`MakeBlobWriter` sites stay on `GetDefaultBlobStore()` unchanged.

### Phase 5 — Promotion

When every reader migrates, the FDR moves from `proposed` to
`accepted`. The stop-gap in `persist.go` collapses into a single
`env.OpenBlobReader` call (Phase 2 already did this).

## Key Files

  ----------------------------------------------------------------------
  File                                                Role
  --------------------------------------------------- ------------------
  `go/internal/foxtrot/env_repo/main.go`              Add
                                                      `OpenBlobReader`
                                                      method.

  `go/internal/november/store_config/persist.go`      Stop-gap site;
                                                      collapses in
                                                      Phase 2.

  `go/internal/foxtrot/env_repo/main_test.go`         New: unit test
                                                      for
                                                      `OpenBlobReader`.

  (~30 reader call sites)                             Phase 3 migration
                                                      target list above.
  ----------------------------------------------------------------------

## Rollback Strategy

Phase 1 (stop-gap, shipped) is already revertable on its own by
restoring the single-store `GetDefaultBlobStore().MakeBlobReader`
call. The 5 `clone_history_*` tests would fail again.

Phase 2's `OpenBlobReader` is additive; reverting it means deleting
the new method and restoring `loadMutableConfigBlob` to its stop-gap
form.

Phase 3 migrations are each one-line and individually revertable.

No on-disk format changes. No flake / dep changes. Suite passes
between every phase.

[gitceiling]: https://git-scm.com/docs/git#Documentation/git.txt-GITCEILINGDIRECTORIES
