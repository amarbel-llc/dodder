---
status: exploring
date: 2026-03-19
promotion-criteria: at least one command (add or new) refactored to use two-stage commit with zettel ID pre-allocation in the plan phase, BATS tests pass unchanged
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
mid-commit (line 139 of `papa/store/mutating.go`), interleaved with blob saving,
hook execution, and finalization. There is no point where "all zettel IDs have
been allocated" and "no inventory list writes have started" — the two are
interleaved per-object.

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
- Save blobs to content-addressable store (idempotent, no lock needed)
- Run pre-commit hooks and validation
- Allocate zettel IDs for new objects (acquires zettel ID flock briefly, then
  releases)
- Classify objects (new vs update vs skip)
- Produce a commit plan: a list of objects ready to persist

**Phase 2 — Commit (lock per resource):**
- Acquire inventory list flock
- Write all objects to the inventory list in a single batch
- Atomic swap of the inventory list file
- Release inventory list flock
- Update stream index, abbreviation index (can use their own flocks or be
  rebuilt from the inventory list)

The plan is the unit of atomicity — either all objects in the plan are committed,
or none are.

### Zettel ID Allocation in the Plan Phase

Today, `CreateZettelId` is called inside `commitFacilitator.commit` when an
object has an empty ID and `AddToInventoryList` is true. In the two-stage model:

1. The plan builder iterates over incoming objects
2. For each object needing a new ID, it acquires the zettel ID flock, reads the
   bitset, allocates an ID, writes the bitset, releases the flock
3. The allocated ID is stored in the plan entry
4. Phase 2 commits objects using their pre-allocated IDs — no zettel ID index
   access needed

This means the zettel ID flock is held only during allocation (phase 1), not
during the entire commit (phase 2). For bulk operations like `add` with 50
files, the IDs can be batch-allocated in a single flock acquisition.

### Existing Two-Stage Pattern: import_plan

`import_plan.Builder` + `remote_transfer.CommitPlan` already implement this
pattern for remote transfers:

- `Builder.AddObject` classifies each incoming object (import, skip-exists,
  skip-dedup, resolve-tai-reassign, error-missing-blob) without holding any lock
- `Builder.Build` produces a `Plan` with topologically sorted entries
- `CommitPlan` acquires the lock, iterates committable entries, and commits each

The gap: `import_plan` does not handle zettel ID allocation (remote objects
arrive with IDs already assigned). Local commands need a plan builder that also
allocates IDs.

### Commands to Migrate

| Command | Current path | Notes |
|---------|-------------|-------|
| `new` | `store.CreateOrUpdate` | Allocates one zettel ID per call |
| `add` | `store_fs.SaveBlob` → `store.Commit` | Batch of files, each committed individually |
| `checkin` | `store_fs` → `store.Commit` | Similar to add |
| `organize` | `store_workspace` → `store.Commit` | Batch of objects from editor |
| `edit` | `store.CreateOrUpdate` | Single object |
| `pull`/`clone` | `import_plan` → `CommitPlan` | Already two-stage |

### What Does NOT Change

- `import_plan` and `CommitPlan` for remote transfers — already correct
- Object format, blob format, inventory list format
- The `store.Commit` method itself — it becomes the inner loop of phase 2,
  called with pre-allocated IDs
- Content-addressable blob writes — already idempotent and lockless

## Implementation Status

### What's Built

- `import_plan.Builder` and `import_plan.Plan` — the plan data structure and
  classification logic
- `remote_transfer.CommitPlan` — batch commit execution under lock
- `commitFacilitator.tryPrecommit` — already separates pre-processing from
  commit (blob saving, hooks, validation), but is called per-object inside the
  locked section

### What's NOT Built

- Local command plan builder (extends `import_plan` or new builder that handles
  zettel ID allocation)
- Zettel ID batch allocation (acquire flock once, allocate N IDs, release)
- Migration of `add`, `new`, `organize`, `checkin`, `edit` to two-stage
- Atomic file swap for the zettel ID index gob file

## Rollback Strategy

### Dual-Architecture Period

The two-stage commit is purely internal — no CLI surface changes. Commands can
be migrated one at a time. During migration, some commands use the old
single-stage path while others use the new two-stage path. Both are correct
under the existing `LockSmith`.

### Promotion Criteria (exploring -> proposed)

- At least one local command (`new` or `add`) prototyped with two-stage commit
- Zettel ID allocation moved to plan phase for that command
- Existing BATS tests pass unchanged

### Rollback Procedure

Revert the command's `Run` method to the old single-stage path. No data
migration needed — the plan phase produces the same objects as the old path.
