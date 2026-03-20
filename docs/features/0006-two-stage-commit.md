---
status: experimental
date: 2026-03-20
promotion-criteria: "store.Store exposes CommitPlan method replacing per-object Commit for plan-based mutations; Checkin migrated; dry-run inspects plan without committing; full test suite passes"
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
own I/O — not for the entire operation. But today, zettel ID allocation happens
mid-commit (`papa/store/mutating.go:139`), interleaved with blob saving, hook
execution, and finalization. There is no point where "all zettel IDs have been
allocated" and "no inventory list writes have started" — the two are interleaved
per-object.

The remote transfer path (`pull`/`clone`) already solves this via `import_plan`:
objects are classified and validated in a plan phase (no lock required), then
committed in a batch under a single lock acquisition. But the local mutation
commands (`add`, `organize`, `new`, `checkin`) bypass `import_plan` entirely —
they call `store.Commit` directly for each object, with the lock held for the
full duration.

## Design

### Overview

Separate store mutations into two phases:

**Phase 1 — Plan (no lock required):**
- Allocate zettel IDs for new objects (in bulk if multiple objects)
- Classify objects (new vs update vs skip)
- Produce a commit plan: a list of objects with pre-allocated IDs, ready to
  persist

**Phase 2 — Commit (under `LockSmith`, same as today):**
- Acquire `LockSmith`
- For each plan entry, call `store.Commit` with the pre-allocated ID already set
  on the object (skipping the `CreateZettelId` call inside `commit`)
- Flush (inventory list, stream index, zettel ID index, etc.)
- Release `LockSmith`

Phase 2 uses the existing `LockSmith` — no `flock(2)` changes yet. The goal of
this FDR is to separate ID allocation from persistence, not to change the
locking mechanism. `flock(2)` migration is a follow-up concern
([ADR-0001](../decisions/0001-use-flock-for-fine-grained-resource-locking.md)).

The plan is the unit of atomicity — either all objects in the plan are committed,
or none are.

### Zettel ID Allocation in the Plan Phase

Today, `CreateZettelId` is called inside `commitFacilitator.commit`
(`papa/store/mutating.go:134-148`) when an object has an empty ID and
`AddToInventoryList` is true:

```go
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

1. The plan builder iterates over incoming objects
2. For each object needing a new ID, it calls `zettelIdIndex.CreateZettelId()`
   and sets the ID on the object
3. The plan entry stores the object with its pre-allocated ID
4. Phase 2 calls `store.Commit` — the empty-ID check at line 134 sees a
   non-empty ID and skips `CreateZettelId` entirely

No changes to `store.Commit` are needed. The existing empty-ID check already
gates `CreateZettelId` — pre-populating the ID is sufficient to bypass it.

For bulk operations (e.g., `new -count 5`), all IDs are allocated in a tight
loop before any commit work begins. Today they are allocated one at a time,
interleaved with commits.

### Starting Point: `new` Command (Zero-Arg Path)

The `new` command's zero-arg path (create empty zettels) is the simplest
candidate for migration. It has a clean, traceable call chain:

```
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
```

The two-stage version would be:

```
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
```

### What About `tryPrecommit`?

`tryPrecommit` (`papa/store/mutating.go:44-98`) runs blob saving, applies
proto, discovers references, and runs pre-commit hooks. It is currently called
inside `commit`, after ID allocation.

For this first migration, **`tryPrecommit` stays where it is** — inside
`store.Commit`, under the lock. Only zettel ID allocation moves to the plan
phase. Moving blob saving and hooks out of the locked section is a future
optimization that can happen after the ID allocation separation is proven.

### Approach: Modify `WriteNewZettels`, Not `import_plan`

The existing `import_plan.Builder` is designed for remote transfers — it handles
deduplication, TAI reassignment, topological sorting, and classification of
already-identified objects. Local mutations have different needs (ID allocation,
proto application) and simpler classification (always "create").

Rather than extending `import_plan` with local-mutation concerns, the
recommended approach is to modify `WriteNewZettels` directly:

1. Move the `CreateZettelId` + `Set` calls into a pre-lock loop
2. Collect the prepared objects into a slice
3. Lock, commit each, unlock

This is a minimal change to `WriteNewZettels.RunMany` (~20 lines moved). No new
types or packages needed. If the pattern proves out, it becomes the template for
migrating other commands, at which point a shared plan builder may be extracted.

### Commands to Migrate (in order)

| Command | Current call chain | Complexity |
|---------|--------------------|------------|
| `new` (zero-arg) | `WriteNewZettels.RunMany` → `CreateOrUpdateDefaultProto` → `Commit` | Lowest — single object type, no blob, no hooks that depend on ID |
| `new` (with paths) | `CreateFromPaths.Run` → `CreateOrUpdateDefaultProto` → `Commit` | Low — already collects objects before locking (`toCreate` map) |
| `new` (with shas) | `CreateFromShas.Run` → similar | Low — similar to paths |
| `add` | `store_fs.SaveBlob` → `Commit` | Medium — blob saving interleaved |
| `checkin` | `store_fs` → `Commit` | Medium — similar to add |
| `organize` | `store_workspace` → `Commit` | Medium — batch from editor |
| `edit` | `CreateOrUpdate` | Low — single object, but may have blob |
| `pull`/`clone` | `import_plan` → `CommitPlan` | Already two-stage |

### What Does NOT Change

- `import_plan` and `CommitPlan` for remote transfers — already correct
- `store.Commit` method — it remains the inner commit loop, just receives
  objects with IDs already set
- `commitFacilitator.tryPrecommit` — stays inside `commit` for now
- Object format, blob format, inventory list format
- Content-addressable blob writes — already idempotent and lockless
- `LockSmith` mechanism — no `flock(2)` changes in this FDR

## Key Files

| File | Role |
|------|------|
| `go/internal/victor/commands_dodder/new.go` | `New.Run` — entry point, dispatches to `WriteNewZettels` / `CreateFromPaths` / `CreateFromShas` |
| `go/internal/tango/user_ops/write_new_zettels.go` | `WriteNewZettels.RunMany` — **primary migration target**. Currently: Lock → loop(Make + CreateOrUpdateDefaultProto) → Unlock |
| `go/internal/tango/user_ops/create_from_paths.go` | `CreateFromPaths.Run` — secondary target. Already separates parsing (pre-lock) from commit (under lock), but ID allocation is inside commit |
| `go/internal/papa/store/create.go` | `CreateOrUpdate` / `CreateOrUpdateDefaultProto` — sets `AddToInventoryList = true`, calls `Commit` |
| `go/internal/papa/store/mutating.go` | `commitFacilitator.commit` — lines 134-148 allocate zettel ID when object ID is empty. Lines 44-98 (`tryPrecommit`) handle blob saving and hooks |
| `go/internal/foxtrot/zettel_id_index/v0/main.go` | `index.CreateZettelId` — picks from available IDs in `map[int]bool`. `AddZettelId` — removes an ID from available pool |
| `go/internal/foxtrot/zettel_id_index/main.go` | `Index` interface — `CreateZettelId`, `AddZettelId`, `Reset`, `Flush`, `PeekZettelIds` |
| `go/internal/sierra/local_working_copy/lock.go` | `Repo.Lock` / `Repo.Unlock` — acquires/releases `LockSmith`. `Unlock` triggers flush |

## Next Phase: Unify Local and Remote Plans via `import_plan.Builder`

With inline ID pre-allocation proven across all local mutation commands, the
next step is to consolidate the ad-hoc plan slices into a shared plan type
backed by `import_plan.Builder`. This unifies local and remote commit paths
under a single plan abstraction.

### Why Now

Each migrated command independently implements the same pattern: collect objects
into a slice, pre-allocate IDs, lock, commit each, unlock. The duplication is
manageable at four call sites but will not scale if new commands or plan-phase
concerns (dedup, conflict detection, dry-run) are added. Builder already solves
classification, deduplication, TAI collision resolution, and topological sorting
for remote transfers — extending it for local mutations avoids reinventing these.

### What Builder Provides

`import_plan.Builder` (`india/import_plan/builder.go`) is a stateful accumulator
with a six-stage pipeline:

1. Apply transforms (optional `ObjectTransform` callbacks)
2. Skip blobless types
3. Deduplicate by content digest (configurable format purpose)
4. Resolve TAI collisions (within batch and against existing index)
5. Check against existing store index (skip-exists vs import)
6. Build type dependency graph → topological sort → height assignment

Output is an immutable `Plan` with sorted `Entry` values, each carrying a
`Classification` enum (`import`, `skip-exists`, `skip-dedup`,
`resolve-tai-reassign`, etc.).

### What Builder Lacks for Local Mutations

- **Zettel ID allocation** — Builder assumes objects already have IDs (they
  arrive from a remote with IDs assigned). Local mutations need
  `CreateZettelId()` for new objects.
- **Blob saving** — handled inside `commitFacilitator.tryPrecommit`, not in the
  plan phase.
- **Hook execution** — same, inside `tryPrecommit`.

### Why `CommitPlan` Cannot Be Reused As-Is

`remote_transfer.CommitPlan()` (`romeo/remote_transfer/commit_plan.go`) is
hardcoded to `remote_transfer.importer` — it casts the `repo.Importer` interface
to the private `importer` struct and calls private methods for blob copying,
signature overwriting, and inventory list import. A local commit executor is
much simpler: iterate committable entries, call `store.Commit()` with
pre-populated IDs.

### Unification Path

1. **Add a post-classification hook to Builder** for zettel ID allocation —
   call `CreateZettelId()` on entries classified as "import" with empty IDs.
   This runs during `Builder.Build()`, before the plan is finalized.
2. **Write a local `CommitPlan` function** that iterates committable entries and
   calls `store.Commit()` — no blob copying, no signature overwriting. This
   replaces the inline `for _, object := range planned` loops in each command.
3. **The shared `Plan` type becomes the unit of atomicity** for both local and
   remote mutations. Commands construct a `Builder`, add objects, call
   `Build()` (which allocates IDs), then pass the resulting `Plan` to the
   appropriate `CommitPlan` (local or remote).

### What This Enables

- **Dry-run / preview** — `Build()` produces a plan that can be inspected
  before committing. Commands could gain a `-dry-run` flag that prints the plan
  without executing phase 2.
- **Deduplication for local mutations** — `checkin` and `add` can benefit from
  Builder's content-digest dedup, skipping objects that already exist in the
  store.
- **Conflict detection** — the plan phase can detect TAI collisions between
  concurrent local mutations before acquiring the lock.
- **Single code path for `tryPrecommit` extraction** — once all mutations flow
  through `Plan`, moving blob saving and hooks out of the locked section becomes
  a change to one commit executor, not four.

## Implementation Status

### Phase 1: Inline ID Pre-Allocation (Complete)

All local mutation commands that create new zettels now pre-allocate zettel IDs
before acquiring `LockSmith`:

| Command | File | Migration |
|---------|------|-----------|
| `new` (zero-arg) | `tango/user_ops/write_new_zettels.go` | `RunMany` collects planned objects with pre-allocated IDs |
| `new` (with paths) | `tango/user_ops/create_from_paths.go` | Pre-lock loop over `toCreate` map |
| `new` (with shas) | `tango/user_ops/create_from_shas.go` | Same pattern as paths |
| `checkin` | `sierra/local_working_copy/op_checkin.go` | Phase 1 loop over untracked Zettel/Blob objects |

Commands that only update existing objects (`organize`, `edit`) were not
migrated — they never call `CreateZettelId`.

Remote transfers (`pull`/`clone`) were already two-stage via `import_plan`.

### Phase 2: Builder Unification (In Progress)

- [x] Write `MakeAllocateZettelIdTransform` for zettel ID allocation
- [x] Write local `CommitPlan` function in `tango/user_ops`
- [x] Migrate `WriteNewZettels` (`new` zero-arg) to Builder + CommitPlan
- [x] Migrate `CreateFromShas` (`new -shas`) to Builder + CommitPlan
- [x] Migrate `CreateFromPaths` (`new` with file args) to Builder + CommitPlan
- [ ] Migrate `Checkin` (deferred — mixed update/create loop needs richer commit executor)

## Rollback Strategy

The two-stage commit is purely internal — no CLI surface changes. Phase 1
(inline pre-allocation) and Phase 2 (Builder unification) are independently
revertable:

- **Revert Phase 1**: Move `CreateZettelId` back inside the lock in each
  command. No data migration needed — the plan phase produces the same objects
  as the old path.
- **Revert Phase 2** (once built): Replace Builder-based plans with the current
  inline slices. The inline approach is correct under `LockSmith`.

Both phases are correct under the existing `LockSmith` because:

- Pre-allocating IDs then committing produces the same result as allocating
  during commit
- The zettel ID index is flushed during `Unlock` regardless of when allocation
  happened
- No other process can interfere because `LockSmith` is held during phase 2

### Phase 3: Store Consumes Plan (Not Started)

Today `CommitPlan` in `tango/user_ops` iterates plan entries and calls
`store.CreateOrUpdateDefaultProto` per-object — it's a loop around the
single-commit interface. The store has no concept of "here's a batch of work."

Moving plan consumption into the store enables:
- **Dry-run as plan inspection** — store receives plan, validates, decides not
  to flush. No wasted commit work.
- **File-persisted plan** — store writes the plan as an inventory list before
  executing, enabling atomic commit and crash recovery
  ([#9](https://github.com/amarbel-llc/dodder/issues/9)).
- **Checkin unification** — store handles per-entry dispatch (create vs update
  checked-out) internally, removing the mixed loop from Checkin
  ([#12](https://github.com/amarbel-llc/dodder/issues/12)).

### Promotion Criteria (experimental → testing)

- `store.Store` exposes a `CommitPlan(*import_plan.Plan, ...)` method that
  replaces per-object `Commit` calls for plan-based mutations
- Checkin migrated to Builder + store-level CommitPlan
- Dry-run mode inspects the plan without executing commits
- Full test suite passes
