---
date: 2026-03-15
promotion-criteria: all local mutation callers use ExecutePlan (including
  organize, remote-add); Commit removed from sku.RepoStore interface; full test
  suite passes
status: testing
---

# Two-Stage Commit

## Problem Statement

Store mutations in dodder currently interleave pre-processing (blob saving,
validation, hook execution, zettel ID allocation) with persistence (inventory
list updates, index writes) inside `store.Commit`. The entire operation runs
under a repo-wide `LockSmith` lock acquired before the first mutation and
released after the final flush.

This tight coupling prevents fine-grained locking
([ADR-0001](../decisions/0001-use-flock-for-fine-grained-resource-locking.md)).
To use per-resource `flock(2)` locks (e.g., one for the zettel ID index, another
for the inventory list log), each resource's lock must be held only during its
own I/O --- not for the entire operation. But today, zettel ID allocation
happens mid-commit (`papa/store/mutating.go:139`), interleaved with blob saving,
hook execution, and finalization. There is no point where "all zettel IDs have
been allocated" and "no inventory list writes have started" --- the two are
interleaved per-object.

The remote transfer path (`pull`/`clone`) already solves this via `import_plan`:
objects are classified and validated in a plan phase (no lock required), then
committed in a batch under a single lock acquisition. But the local mutation
commands (`add`, `organize`, `new`, `checkin`) bypass `import_plan` entirely ---
they call `store.Commit` directly for each object, with the lock held for the
full duration.

## Design

### Overview

Separate store mutations into two phases:

**Phase 1 --- Plan (no lock required):** - Allocate zettel IDs for new objects
(in bulk if multiple objects) - Classify objects (new vs update vs skip) -
Produce a commit plan: a list of objects with pre-allocated IDs, ready to
persist

**Phase 2 --- Commit (under `LockSmith`, same as today):** - Acquire
`LockSmith` - For each plan entry, call `store.Commit` with the pre-allocated ID
already set on the object (skipping the `CreateZettelId` call inside `commit`) -
Flush (inventory list, stream index, zettel ID index, etc.) - Release
`LockSmith`

Phase 2 uses the existing `LockSmith` --- no `flock(2)` changes yet. The goal of
this FDR is to separate ID allocation from persistence, not to change the
locking mechanism. `flock(2)` migration is a follow-up concern
([ADR-0001](../decisions/0001-use-flock-for-fine-grained-resource-locking.md)).

The plan is the unit of atomicity --- either all objects in the plan are
committed, or none are.

### Zettel ID Allocation in the Plan Phase

Today, `CreateZettelId` is called inside `commitFacilitator.commit`
(`papa/store/mutating.go:134-148`) when an object has an empty ID and
`AddToInventoryList` is true:

``` go
if options.AddToInventoryList && (daughter.GetObjectId().IsEmpty() ||
    daughter.GetGenre() == genres.Unknown ||
    daughter.GetGenre() == genres.Blob) {
    var zettelId *ids.ZettelId
    if zettelId, err = commitFacilitator.zettelIdIndex.CreateZettelId(); err != nil {
        // ...
    }
    // sets the ID on the object
}
```

In the two-stage model, this moves to the plan phase:

1.  The plan builder iterates over incoming objects
2.  For each object needing a new ID, it calls `zettelIdIndex.CreateZettelId()`
    and sets the ID on the object
3.  The plan entry stores the object with its pre-allocated ID
4.  Phase 2 calls `store.Commit` --- the empty-ID check at line 134 sees a
    non-empty ID and skips `CreateZettelId` entirely

No changes to `store.Commit` are needed. The existing empty-ID check already
gates `CreateZettelId` --- pre-populating the ID is sufficient to bypass it.

For bulk operations (e.g., `new -count 5`), all IDs are allocated in a tight
loop before any commit work begins. Today they are allocated one at a time,
interleaved with commits.

### Starting Point: `new` Command (Zero-Arg Path)

The `new` command's zero-arg path (create empty zettels) is the simplest
candidate for migration. It has a clean, traceable call chain:

    new.Run
      → user_ops.WriteNewZettels.RunMany(proto, count)
        → op.Lock()                              # acquires LockSmith
        → for range count:
            → op.runOneAlreadyLocked(proto)
              → proto.Make()                     # creates empty Transacted
              → store.CreateOrUpdateDefaultProto(object, options)
                → store.CreateOrUpdate(object, options)
                  → store.Commit(object, options)
                    → commitFacilitator.commit(object, options)
                      → zettelIdIndex.CreateZettelId()  # <-- ID allocated here
                      → tryPrecommit (blob save, hooks, validation)
                      → commitTransacted (add to working list)
        → op.Unlock()                            # flushes + releases LockSmith

The two-stage version would be:

    new.Run
      → user_ops.WriteNewZettels.RunMany(proto, count)
        → Phase 1 (no lock):
            → for range count:
                → proto.Make()
                → zettelIdIndex.CreateZettelId()
                → set ID on object
                → append to plan
        → Phase 2 (under lock):
            → op.Lock()
            → for each plan entry:
                → store.CreateOrUpdateDefaultProto(entry.object, options)
                  → store.Commit sees non-empty ID, skips CreateZettelId
            → op.Unlock()

### What About `tryPrecommit`?

`tryPrecommit` (`papa/store/mutating.go:44-98`) runs blob saving, applies proto,
discovers references, and runs pre-commit hooks. It is currently called inside
`commit`, after ID allocation.

For this first migration, **`tryPrecommit` stays where it is** --- inside
`store.Commit`, under the lock. Only zettel ID allocation moves to the plan
phase. Moving blob saving and hooks out of the locked section is a future
optimization that can happen after the ID allocation separation is proven.

### Approach: Modify `WriteNewZettels`, Not `import_plan`

The existing `import_plan.Builder` is designed for remote transfers --- it
handles deduplication, TAI reassignment, topological sorting, and classification
of already-identified objects. Local mutations have different needs (ID
allocation, proto application) and simpler classification (always "create").

Rather than extending `import_plan` with local-mutation concerns, the
recommended approach is to modify `WriteNewZettels` directly:

1.  Move the `CreateZettelId` + `Set` calls into a pre-lock loop
2.  Collect the prepared objects into a slice
3.  Lock, commit each, unlock

This is a minimal change to `WriteNewZettels.RunMany` (\~20 lines moved). No new
types or packages needed. If the pattern proves out, it becomes the template for
migrating other commands, at which point a shared plan builder may be extracted.

### Commands to Migrate (in order)

  ------------------------------------------------------------------------------
  Command           Current call chain                    Complexity
  ----------------- ------------------------------------- ----------------------
  `new` (zero-arg)  `WriteNewZettels.RunMany` →           Lowest --- single
                    `CreateOrUpdateDefaultProto` →        object type, no blob,
                    `Commit`                              no hooks that depend
                                                          on ID

  `new` (with       `CreateFromPaths.Run` →               Low --- already
  paths)            `CreateOrUpdateDefaultProto` →        collects objects
                    `Commit`                              before locking
                                                          (`toCreate` map)

  `new` (with shas) `CreateFromShas.Run` → similar        Low --- similar to
                                                          paths

  `add`             `store_fs.SaveBlob` → `Commit`        Medium --- blob saving
                                                          interleaved

  `checkin`         `store_fs` → `Commit`                 Medium --- similar to
                                                          add

  `organize`        `store_workspace` → `Commit`          Medium --- batch from
                                                          editor

  `edit`            `CreateOrUpdate`                      Low --- single object,
                                                          but may have blob

  `pull`/`clone`    `import_plan` → `CommitPlan`          Already two-stage
  ------------------------------------------------------------------------------

### What Does NOT Change

- `import_plan` and `CommitPlan` for remote transfers --- already correct
- `store.Commit` method --- it remains the inner commit loop, just receives
  objects with IDs already set
- `commitFacilitator.tryPrecommit` --- stays inside `commit` for now
- Object format, blob format, inventory list format
- Content-addressable blob writes --- already idempotent and lockless
- `LockSmith` mechanism --- no `flock(2)` changes in this FDR

## Key Files

  ----------------------------------------------------------------------------------
  File                                                Role
  --------------------------------------------------- ------------------------------
  `go/internal/victor/commands_dodder/new.go`         `New.Run` --- entry point,
                                                      dispatches to
                                                      `WriteNewZettels` /
                                                      `CreateFromPaths` /
                                                      `CreateFromShas`

  `go/internal/tango/user_ops/write_new_zettels.go`   `WriteNewZettels.RunMany` ---
                                                      **primary migration target**.
                                                      Currently: Lock → loop(Make +
                                                      CreateOrUpdateDefaultProto) →
                                                      Unlock

  `go/internal/tango/user_ops/create_from_paths.go`   `CreateFromPaths.Run` ---
                                                      secondary target. Already
                                                      separates parsing (pre-lock)
                                                      from commit (under lock), but
                                                      ID allocation is inside commit

  `go/internal/papa/store/create.go`                  `CreateOrUpdate` /
                                                      `CreateOrUpdateDefaultProto`
                                                      --- sets
                                                      `AddToInventoryList = true`,
                                                      calls `Commit`

  `go/internal/papa/store/mutating.go`                `commitFacilitator.commit` ---
                                                      lines 134-148 allocate zettel
                                                      ID when object ID is empty.
                                                      Lines 44-98 (`tryPrecommit`)
                                                      handle blob saving and hooks

  `go/internal/foxtrot/zettel_id_index/v0/main.go`    `index.CreateZettelId` ---
                                                      picks from available IDs in
                                                      `map[int]bool`. `AddZettelId`
                                                      --- removes an ID from
                                                      available pool

  `go/internal/foxtrot/zettel_id_index/main.go`       `Index` interface ---
                                                      `CreateZettelId`,
                                                      `AddZettelId`, `Reset`,
                                                      `Flush`, `PeekZettelIds`

  `go/internal/sierra/local_working_copy/lock.go`     `Repo.Lock` / `Repo.Unlock`
                                                      --- acquires/releases
                                                      `LockSmith`. `Unlock` triggers
                                                      flush
  ----------------------------------------------------------------------------------

## Builder Unification Design

> **Status:** Largely implemented --- see [Implementation
> Status](#implementation-status) for current state. Only Checkin remains
> unmigrated ([#12](https://github.com/amarbel-llc/dodder/issues/12)).

This section describes the design for consolidating the ad-hoc inline plan
slices (from Phase 1) into a shared plan type backed by `import_plan.Builder`,
unifying local and remote commit paths under a single plan abstraction.

### Why Now

Each migrated command independently implements the same pattern: collect objects
into a slice, pre-allocate IDs, lock, commit each, unlock. The duplication is
manageable at four call sites but will not scale if new commands or plan-phase
concerns (dedup, conflict detection, dry-run) are added. Builder already solves
classification, deduplication, TAI collision resolution, and topological sorting
for remote transfers --- extending it for local mutations avoids reinventing
these.

### What Builder Provides

`import_plan.Builder` (`india/import_plan/builder.go`) is a stateful accumulator
with a six-stage pipeline:

1.  Apply transforms (optional `ObjectTransform` callbacks)
2.  Skip blobless types
3.  Deduplicate by content digest (configurable format purpose)
4.  Resolve TAI collisions (within batch and against existing index)
5.  Check against existing store index (skip-exists vs import)
6.  Build type dependency graph → topological sort → height assignment

Output is an immutable `Plan` with sorted `Entry` values, each carrying a
`Classification` enum (`import`, `skip-exists`, `skip-dedup`,
`resolve-tai-reassign`, etc.).

### What Builder Lacks for Local Mutations

- **Zettel ID allocation** --- Builder assumes objects already have IDs (they
  arrive from a remote with IDs assigned). Local mutations need
  `CreateZettelId()` for new objects.
- **Blob saving** --- handled inside `commitFacilitator.tryPrecommit`, not in
  the plan phase.
- **Hook execution** --- same, inside `tryPrecommit`.

### Why `CommitPlan` Cannot Be Reused As-Is

`remote_transfer.CommitPlan()` (`romeo/remote_transfer/commit_plan.go`) is
hardcoded to `remote_transfer.importer` --- it casts the `repo.Importer`
interface to the private `importer` struct and calls private methods for blob
copying, signature overwriting, and inventory list import. A local commit
executor is much simpler: iterate committable entries, call `store.Commit()`
with pre-populated IDs.

### Unification Path

1.  **Add a post-classification hook to Builder** for zettel ID allocation ---
    call `CreateZettelId()` on entries classified as "import" with empty IDs.
    This runs during `Builder.Build()`, before the plan is finalized.
2.  **Write a local `CommitPlan` function** that iterates committable entries
    and calls `store.Commit()` --- no blob copying, no signature overwriting.
    This replaces the inline `for _, object := range planned` loops in each
    command.
3.  **The shared `Plan` type becomes the unit of atomicity** for both local and
    remote mutations. Commands construct a `Builder`, add objects, call
    `Build()` (which allocates IDs), then pass the resulting `Plan` to the
    appropriate `CommitPlan` (local or remote).

### What This Enables

- **Dry-run / preview** --- `Build()` produces a plan that can be inspected
  before committing. Commands could gain a `-dry-run` flag that prints the plan
  without executing phase 2.
- **Deduplication for local mutations** --- `checkin` and `add` can benefit from
  Builder's content-digest dedup, skipping objects that already exist in the
  store.
- **Conflict detection** --- the plan phase can detect TAI collisions between
  concurrent local mutations before acquiring the lock.
- **Single code path for `tryPrecommit` extraction** --- once all mutations flow
  through `Plan`, moving blob saving and hooks out of the locked section becomes
  a change to one commit executor, not four.

## Implementation Status

### Phase 1: Inline ID Pre-Allocation (Complete)

All local mutation commands that create new zettels now pre-allocate zettel IDs
before acquiring `LockSmith`:

  -------------------------------------------------------------------------------
  Command          File                                        Migration
  ---------------- ------------------------------------------- ------------------
  `new` (zero-arg) `tango/user_ops/write_new_zettels.go`       `RunMany` collects
                                                               planned objects
                                                               with pre-allocated
                                                               IDs

  `new` (with      `tango/user_ops/create_from_paths.go`       Pre-lock loop over
  paths)                                                       `toCreate` map

  `new` (with      `tango/user_ops/create_from_shas.go`        Same pattern as
  shas)                                                        paths

  `checkin`        `sierra/local_working_copy/op_checkin.go`   Phase 1 loop over
                                                               untracked
                                                               Zettel/Blob
                                                               objects
  -------------------------------------------------------------------------------

Commands that only update existing objects (`organize`, `edit`) were not
migrated --- they never call `CreateZettelId`.

Remote transfers (`pull`/`clone`) were already two-stage via `import_plan`.

### Phase 2: Builder Unification (In Progress)

- [x] Write `MakeAllocateZettelIdTransform` for zettel ID allocation
- [x] Write local `CommitPlan` function in `tango/user_ops`
- [x] Migrate `WriteNewZettels` (`new` zero-arg) to Builder + CommitPlan
- [x] Migrate `CreateFromShas` (`new -shas`) to Builder + CommitPlan
- [x] Migrate `CreateFromPaths` (`new` with file args) to Builder + CommitPlan
- [x] Migrate `Checkin` (deferred to Phase 3 --- see Checkin decomposition)

## Rollback Strategy

The two-stage commit is purely internal --- no CLI surface changes. Phase 1
(inline pre-allocation) and Phase 2 (Builder unification) are independently
revertable:

- **Revert Phase 1**: Move `CreateZettelId` back inside the lock in each
  command. No data migration needed --- the plan phase produces the same objects
  as the old path.

- **Revert Phase 2** (once built): Replace Builder-based plans with the current
  inline slices. The inline approach is correct under `LockSmith`.

- **Revert Phase 3**: Restore `tango/user_ops.CommitPlan`, revert callers to use
  it instead of `local.ExecutePlan`. Remove `ExecutePlan` from `LocalRepo`
  interface. Revert Checkin to its inline mixed loop. Remove
  `DefaultCommitOptions` from `Plan` and `Options` from `Entry`.

All phases are correct under the existing `LockSmith` because:

- Pre-allocating IDs then committing produces the same result as allocating
  during commit
- The zettel ID index is flushed during `Unlock` regardless of when allocation
  happened
- No other process can interfere because `LockSmith` is held during phase 2

### Phase 3: Repo Executes Plan (Complete)

Today `CommitPlan` in `tango/user_ops` iterates plan entries and calls
`store.CreateOrUpdateDefaultProto` per-object --- it's a loop around the
single-commit interface. The repo has no concept of "here's a batch of work."

Moving plan execution into the repo enables: - **Dry-run as plan inspection**
--- the plan is built and inspectable before `ExecutePlan` is called. Dry-run is
handled by the store's existing `IsDryRun()` config flag inside its `Commit`
method tree. - **Checkin unification** --- per-entry `CommitOptions` on the plan
allow the mixed create/update dispatch to be expressed as plan data rather than
branching logic in the checkin loop
([#12](https://github.com/amarbel-llc/dodder/issues/12)). - **Single commit
path** --- all local mutations flow through `ExecutePlan`, paving the way to
remove `Commit` from `sku.RepoStore`.

File-persisted plans and crash recovery
([#9](https://github.com/amarbel-llc/dodder/issues/9)) are deferred to a
separate FDR.

#### Interface

`repo.LocalRepo` gains one method:

``` go
ExecutePlan(plan *import_plan.Plan) (sku.TransactedMutableSet, error)
```

`import_plan.Plan` gains a default commit options field:

``` go
type Plan struct {
    // ... existing fields ...
    DefaultCommitOptions sku.CommitOptions
}
```

`import_plan.Entry` gains an optional per-entry override:

``` go
type Entry struct {
    // ... existing fields ...
    Options *sku.CommitOptions // nil = use Plan.DefaultCommitOptions
}
```

#### Execution semantics

`ExecutePlan` on `local_working_copy.Repo`:

1.  Acquire lock (`local.Lock()`)
2.  Iterate committable entries (`entry.Classification.IsCommittable()`)
3.  Resolve options: `entry.Options` if non-nil, else
    `plan.DefaultCommitOptions`
4.  Call `store.Commit(entry.object, options)` directly (not the convenience
    wrappers)
5.  Collect committed objects into the result set
6.  Release lock (`local.Unlock()`)

The convenience wrappers (`CreateOrUpdate`, `CreateOrUpdateDefaultProto`,
`CreateOrUpdateCheckedOut`) are bypassed --- each plan entry carries the
complete `CommitOptions` needed for its commit.

`ExecutePlan` does not populate default values for `CommitOptions` fields like
`Clock` or `RepoId`. Zero values behave identically to the current code paths:
`commitFacilitator` uses its own sunrise clock when `Clock` is zero, and
`RepoId` defaults to the local repo. Callers set only the fields they need
(typically `StoreOptions` flags and `Proto`).

`Entry.Options` is a pointer for nil-means-use-default semantics only. The value
is copied before use --- callers may share a `CommitOptions` across entries
without mutation concerns.

#### Caller migrations

  -----------------------------------------------------------------------------
  Caller                                 Change
  -------------------------------------- --------------------------------------
  `WriteNewZettels`                      Set `plan.DefaultCommitOptions` with
                                         `ApplyProto` + proto. Call
                                         `local.ExecutePlan(plan)`

  `CreateFromPaths`                      Same pattern as WriteNewZettels

  `CreateFromShas`                       Same pattern as WriteNewZettels

  `Checkin`                              Pre-plan: `RefreshCheckedOut`,
                                         `UpdateTransactedFromBlobs`, zettel ID
                                         allocation. Per-entry options:
                                         `StoreOptionsCreate` + proto for
                                         untracked, `StoreOptionsCreate` for
                                         tracked. Post-plan:
                                         `UpdateCheckoutFromCheckedOut` and
                                         `DeleteCheckedOut` as cleanup

  `tango/user_ops.CommitPlan`            Deleted --- replaced by
                                         `LocalRepo.ExecutePlan`
  -----------------------------------------------------------------------------

`LockAndCommitOrganizeResults` (organize) and `remote_add` are not migrated in
this phase. They continue to call `store.CreateOrUpdate` directly under their
own Lock/Unlock. Migration is required before `Commit` can be removed from
`sku.RepoStore`.

#### Checkin decomposition

Checkin's current mixed loop decomposes into three phases:

**Pre-plan (no lock):**

1.  `RefreshCheckedOut` on each object --- re-reads filesystem state. Currently
    runs under the lock; moving it pre-lock introduces a race if files change
    between refresh and lock acquisition. This is a known concern
    ([#25](https://github.com/amarbel-llc/dodder/issues/25)) accepted for now.
2.  For untracked Zettel/Blob with non-empty metadata: allocate zettel ID,
    `UpdateTransactedFromBlobs`, `proto.Apply`
3.  Add all objects to builder with appropriate per-entry `CommitOptions`:
    `StoreOptionsCreate` + proto for untracked, `StoreOptionsCreate` for tracked
4.  Build plan
5.  Checkin maintains a side map of object ID → `CheckedOut` for post-plan
    correlation

**Execute:**

6.  `local.ExecutePlan(plan)` --- returns `TransactedMutableSet` of committed
    objects

**Post-plan (no lock):**

7.  For tracked objects with `updateCheckout`: correlate committed objects back
    to their `CheckedOut` via the side map, call
    `store.UpdateCheckoutFromCheckedOut`
8.  For objects with `delete`: call `store.DeleteCheckedOut`

Post-plan operations are safe without the lock: they operate on workspace
checkout files (per-process working copy), not on shared store state protected
by `LockSmith`. `UpdateCheckoutFromCheckedOut` writes to `store_fs`;
`DeleteCheckedOut` removes working copy files. Neither modifies inventory lists
or the stream index.

Checkout-store operations (refresh, update, delete) are not commit concerns and
stay outside `ExecutePlan`.

#### Read-then-update commands

`UpdateObject` (`tango/user_ops/update_object.go`) and `Update`
(`victor/commands_dodder/update.go`) follow a read-then-modify-then-commit
pattern under the lock: read the current object state, apply changes (tags,
description, type lock), then commit. Moving the read pre-lock introduces the
same race as `RefreshCheckedOut` --- the object could change between read and
lock acquisition. This is accepted for the same reason: single-user CLI tool,
low contention. Both commands are candidates for `ExecutePlan` migration once
the pre-lock read race is accepted.

#### Lock/Unlock stays on Repo

`ExecutePlan` acquires and releases the lock on `local_working_copy.Repo`.
`Unlock` continues to flush all subsystems (inventory list, store, config,
dormant index). The lock/unlock architecture is not changed in this phase.

The current lock/unlock system conflates mutex release with multi-subsystem
flush. Untangling this is future work that enables store-owned locking and
per-resource `flock(2)` locks (ADR-0001). Context for that refactor is captured
in project memory.

### Phase 3 Scope

- Add `DefaultCommitOptions` to `import_plan.Plan` and optional `Options` to
  `import_plan.Entry`
- Add `ExecutePlan` to `repo.LocalRepo` interface, implement on
  `local_working_copy.Repo`
- Migrate `WriteNewZettels`, `CreateFromPaths`, `CreateFromShas` to
  `ExecutePlan`
- Migrate `Checkin` to plan-based (pre-plan / execute / post-plan)
- Delete `tango/user_ops.CommitPlan`
- Full test suite passes

### Phase 3 Out of Scope

- File-persisted plans / crash recovery (separate FDR)

### Phase 4: Remaining Caller Migration (Complete)

Migrated four additional command-level callers to `ExecutePlan`:

- `remote_add` (`victor/commands_dodder/remote_add.go`)
- `edit_config` (`victor/commands_dodder/edit_config.go`)
- `checkin_blob` (`victor/commands_dodder/checkin_blob.go`)
- `LockAndCommitOrganizeResults` (`sierra/local_working_copy/organize.go`)

Split `MakeBuilder` into `MakeImportBuilder` (stream index, dedup, Config genre
skip) and `MakeLocalBuilder` (no index, no dedup, allows Config genre).

Extracted `Commit` from `sku.RepoStore` into a new `StoreCommitter` interface.
`RepoStore` is now read-only. Internal callers (`store_fs` via `Supplies`,
`remote_transfer` via `committer`) use `StoreCommitter` explicitly.

### Promotion Criteria (experimental → testing) --- Met

- `LocalRepo` exposes `ExecutePlan` method
- All local mutation callers use `ExecutePlan` (including organize, remote-add,
  edit-config, checkin-blob)
- `Commit` removed from `sku.RepoStore` interface (extracted to
  `StoreCommitter`)
- Full test suite passes
