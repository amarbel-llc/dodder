---
status: exploring
date: 2026-03-19
promotion-criteria: zettel ID index migrated to flock(2) with atomic file swap, BATS tests pass, workspace repos can allocate IDs from parent's index without corruption under concurrent access
---

# Use flock(2) for Fine-Grained Resource Locking

## Context and Problem Statement

Dodder uses a single repo-wide filesystem lock (`LockSmith` via `file_lock.Lock`)
to serialize all store mutations. This lock uses `O_EXCL|O_CREATE` — file
existence means locked, file removal means unlocked. It is too broad (blocks all
mutations when only one resource is being touched), too eager (acquired before
any I/O, held through the entire operation), and not crash-safe (a crash leaves a
stale lockfile that requires manual `rm` to recover).

This becomes a blocking problem for workspace-repos
([FDR-0005](../features/0005-workspace-as-repo.md)), which need to allocate
zettel IDs from the parent repo's index. The workspace and parent are independent
processes with independent `LockSmith` instances — neither's repo-wide lock
protects the other from concurrent access to the shared zettel ID index file.

The broader question: should dodder move from a single repo-wide lock to
per-resource locks using `flock(2)`, starting with the zettel ID index as a
proving ground?

## Decision Drivers

* Workspace-repos must allocate zettel IDs from the parent's index without
  copying the index or risking ID collisions
* Crash safety — stale locks from killed processes should not block subsequent
  operations
* Concurrency — fine-grained locks enable parallel operations on independent
  resources (e.g., zettel ID allocation does not block inventory list commits)
* `flock(2)` is supported on Linux and macOS (Darwin), the two target platforms.
  NFS is not a concern — workspace-repos are constrained to the same filesystem
  as their parent
* Incremental migration — the repo-wide `LockSmith` cannot be removed in one
  step; per-resource locks must coexist during transition

## Considered Options

* **Option 1: Per-resource flock(2) locks, incremental migration starting with
  zettel ID index**
* **Option 2: Dedicated flock for zettel ID index only, keep LockSmith for
  everything else permanently**
* **Option 3: Migrate LockSmith itself to flock(2) under the hood, keep
  repo-wide granularity**

## Pros and Cons of the Options

### Option 1: Per-resource flock(2), incremental migration

Each resource that needs cross-process safety gets its own `flock(2)` lock.
Resources are migrated one at a time. Index rebuilds use atomic file swap
(write to temp file, `link(2)`/`rename(2)` into place). The repo-wide
`LockSmith` is eventually retired once all resources have their own locks.

Migration order:
1. Zettel ID index (proving ground — simplest resource, enables workspace
   remote index)
2. Inventory list log
3. Stream index, dormant index, other caches

Each resource's lock protects only its own I/O. The atomic swap pattern
(temp file + hardlink) ensures readers never see partial writes — either they
see the old file or the new one.

* Good, because crash-safe — kernel releases flock on process exit
* Good, because fine-grained — independent resources can be mutated concurrently
* Good, because the zettel ID index is the ideal proving ground (single gob
  file, short hold time, clear read/write boundary)
* Good, because atomic swap eliminates partial-write corruption without
  requiring the reader to hold a lock for the entire read
* Bad, because two locking mechanisms coexist during the migration period
* Bad, because each resource migration requires auditing all code paths that
  touch that resource's files
* Neutral, because lock ordering must be documented and enforced (always acquire
  resource-specific flock after any broader lock, never in reverse)

### Option 2: Dedicated flock for zettel ID index only

Add `flock(2)` solely for the zettel ID bitset file. The parent's local index
and the workspace's remote index both acquire this flock before reading/writing
the gob file. Everything else continues using `LockSmith` as-is.

* Good, because minimal blast radius — only touches zettel ID index code
* Good, because solves the immediate workspace problem
* Bad, because the stale-lock-on-crash problem remains for all other resources
* Bad, because it's a dead-end — if other resources later need cross-process
  safety, the pattern must be reinvented each time
* Bad, because the zettel ID index must still coordinate with `LockSmith`
  (parent process holds LockSmith + flock; workspace holds only flock)

### Option 3: Migrate LockSmith to flock(2), keep repo-wide

Replace the `O_EXCL|O_CREATE` mechanism inside `file_lock.Lock` with
`flock(2)`. All existing code continues acquiring a single repo-wide lock,
but it becomes crash-safe.

* Good, because crash-safe with no code changes outside `file_lock`
* Good, because no coexisting locking mechanisms
* Bad, because does not solve the workspace problem — the workspace still
  cannot acquire the parent's repo-wide lock without opening a full parent env
* Bad, because granularity remains repo-wide — no concurrency improvement
* Neutral, because it could be a stepping stone to Option 1 (migrate LockSmith
  first, then decompose into per-resource locks)

## Prerequisites

* [FDR-0006: Two-Stage Commit](../features/0006-two-stage-commit.md) —
  separating pre-processing (zettel ID allocation, validation) from persistence
  (inventory list writes) is required before per-resource flocks can be held
  narrowly. Without two-stage commit, the flock would be held for the entire
  operation — equivalent to the current repo-wide lock.

## More Information

* [FDR-0005: Workspace-as-Repo](../features/0005-workspace-as-repo.md) —
  motivating use case for cross-repo zettel ID allocation
* `go/internal/echo/file_lock/lock.go` — current `LockSmith` implementation
* `go/internal/foxtrot/zettel_id_index/v0/main.go` — current zettel ID index
  (gob-encoded `map[int]bool`)
* `go/lib/_/interfaces/lock.go` — `LockSmith` interface
* `flock(2)` — POSIX advisory file locking, available on Linux and macOS.
  Go exposes it via `syscall.Flock`. Not safe on NFS (irrelevant here —
  workspaces are same-filesystem)
