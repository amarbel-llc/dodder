---
status: proposed
date: 2026-03-12
promotion-criteria: import of a production inventory list with known blobless types, TAI collisions, and overwritten signatures completes phase 1 (plan) without errors, and phase 2 (commit) produces the same store state as the current single-pass import minus the dropped objects. Multi-list import merges plans correctly.
---

# Two-Phase Import with Topographic Processing

## Problem Statement

`der import` currently processes inventory list objects sequentially in arrival
order, committing each object immediately. This design has three consequences:

1. **No upfront validation.** Blobless type definitions
   (`ErrBloblessTypeSkipped`), ObjectId+TAI collisions
   (`ErrObjectIdTaiCollision`), dedup decisions (`ErrDeduped`), and missing blobs
   are discovered mid-stream. The user sees errors interleaved with successful
   imports and has no way to review the full picture before objects are committed.

2. **No dependency ordering.** Objects arrive in inventory list order, which is
   not guaranteed to be topographic. A zettel may be imported before the type it
   references, causing the type lock to fail or resolve against stale local
   state. The existing TODO at `import.go:17` (`// TODO create an open list and
   resolve the graph as necessary`) acknowledges this gap.

3. **No review step.** Once `der import` starts committing, there is no way to
   inspect the import plan, skip specific objects, or abort cleanly. The
   `-continue-on-error` flag helps surface problems but doesn't prevent
   partially-imported state.

A two-phase design — plan+validate, then review+commit — would let the user see
all issues upfront, ensure types are imported before their dependents, and
provide a natural point to inspect or abort before any state changes.

## Interface

### Phase 1: Plan

Given an inventory list path, the importer reads all objects, builds a dependency
graph, and produces an import plan. The plan includes:

- **Topographic ordering.** Objects sorted so that types are imported before
  objects that reference them. Within a dependency level, objects are ordered by
  TAI (oldest first).

- **Validation summary.** All issues detected upfront:
  - Blobless type definitions (currently `ErrBloblessTypeSkipped`)
  - ObjectId+TAI collisions (currently `ErrObjectIdTaiCollision`)
  - Dedup candidates (currently `ErrDeduped`)
  - Missing blobs (currently collected in `missingBlobs` heap)
  - Objects already present in the local store (`ErrExists`)

- **Object classification.** Each object in the plan is marked as one of:
  - `import` — new object, will be committed
  - `skip:exists` — already in local store
  - `skip:dedup` — duplicate content within the batch
  - `skip:blobless-type` — type definition without blob data
  - `resolve:tai-reassign` — ObjectId+TAI collision, will be committed with adjusted TAI
  - `error:missing-blob` — blob not available in source or local store

No state changes occur during phase 1. The repo is not locked.

### Phase 2: Commit

After the user reviews the plan, the importer commits objects in topographic
order. Only objects classified as `import` are committed. The repo lock is
acquired at the start of phase 2 and released at the end.

Objects that depend on a skipped or errored type are themselves reclassified
during the plan phase — they are not silently committed with broken type locks.

### TAI collision resolution

When two objects share the same ObjectId+TAI but have different blob digests
(bug 4 — 10,731 occurrences in production), the plan detects the collision and
assigns a resolution:

- **Within a single list:** the first occurrence (by stream order) is classified
  `import`; subsequent collisions are classified `resolve:tai-reassign` and
  receive a new TAI with the next available sub-second increment. This preserves
  all distinct content.

- **Against the local store:** if the local store already has an object at that
  ObjectId+TAI with a different digest, the incoming object is classified
  `resolve:tai-reassign` with the same TAI adjustment.

TAI reassignment is visible in the plan output so the user can verify no
semantically important timestamps are shifted.

### Signature rewriting in topographic order

When `-overwrite-signatures` is set, phase 2 re-signs objects in topographic
order. Type objects are signed first; dependent objects are then signed using the
new type signatures. This replaces the current per-object callback that signs in
stream order (which can produce stale type references in dependent signatures).

The plan phase records which objects need re-signing. Phase 2 performs:

1. Sign type objects (topographic leaves first).
2. Propagate new type signatures to dependent objects' metadata.
3. Sign dependent objects.

Objects whose types were skipped or errored are reclassified before signing.

### Multiple inventory lists

`der import` accepts one or more inventory list paths:

    der import <list-1> [<list-2> ...]

Phase 1 reads all lists and merges them into a single plan. Cross-list
deduplication and TAI collision detection operate across the merged set. The plan
summary groups counts per list and shows a merged total.

When lists contain the same object (same ObjectId+TAI+digest), the duplicate is
classified `skip:dedup` regardless of which list it appears in.

### CLI flags

    der import <list-path> [<list-path> ...]

Default behavior: run phase 1, print the plan summary to stderr, then proceed to
phase 2. This preserves backward compatibility — existing scripts continue to
work.

New flags:

- `-dry-run` — run phase 1 only, print the plan, exit without committing.
  Exit code 0 if no errors, non-zero if errors were detected.

- `-plan-format <format>` — control plan output format. Values: `summary`
  (default, human-readable counts), `objects` (one line per object with
  classification).

Existing flags are unchanged. `-continue-on-error` applies to phase 2 commit
errors (not phase 1 validation, which always reports all issues).

### Topographic ordering

The dependency graph has two edge types:

1. **Type edges.** Every object with a non-builtin type depends on its type
   object. Type objects may themselves have types (meta-types).

2. **Referenced object edges.** Objects with referenced-object locks
   (FDR-0001) depend on their referenced objects.

Cycles are not expected (types form a DAG) but are detected and reported as
plan errors if encountered.

## Examples

Plan-only run showing a summary:

    $ der import -dry-run export.inventory
    import plan for export.inventory:
      1,204 objects to import
         38 types (imported first)
        112 skip (already exist)
          7 skip (dedup)
          2 skip: blobless type definitions
         12 resolve: TAI reassignment (24 objects)
          1 error: missing blob

Multi-list plan:

    $ der import -dry-run repo-a.inventory repo-b.inventory
    import plan for 2 inventory lists:
      repo-a.inventory:  1,204 objects
      repo-b.inventory:    387 objects
      merged total:      1,544 objects to import
                            47 skip (dedup, cross-list)
                            12 resolve: TAI reassignment

Per-object plan output:

    $ der import -dry-run -plan-format objects export.inventory
    import  type     one/uno          1709234567.0
    import  type     two/dos          1709234568.0
    skip:blobless-type  type  three/tres  1709234569.0
    import  zettel   ceroplastes/midtown  1709234570.0
    skip:exists  zettel  papilio/uptown  1709234571.0
    resolve:tai-reassign  zettel  bombyx/downtown  1709234572.0 -> 1709234572.1
    ...

Normal import with plan summary before commit:

    $ der import export.inventory
    import plan for export.inventory:
      1,204 objects to import
         38 types (imported first)
        112 skip (already exist)
    [   1/1204] one/uno
    [   2/1204] two/dos
    ...

## Limitations

- Phase 2 does not support selective import (e.g. "import only these 5
  objects"). The plan classifies objects automatically; user intervention is
  limited to reviewing the plan and deciding whether to proceed. Selective
  import may be added later as a `-filter` flag or interactive mode.

## Future exploration: streaming plan construction

The current design loads all objects into memory during phase 1 to build the
dependency graph. This is acceptable for production inventory lists today (the
index is already memory-resident), but may not scale to very large imports or
memory-constrained environments.

Approaches worth exploring:

- **Two-pass streaming.** First pass: stream objects to extract only the
  dependency edges (ObjectId, type reference, TAI) into a compact graph. Second
  pass: stream again in topographic order, classifying each object against the
  pre-built graph and the local index. Only the graph structure lives in memory,
  not the full object metadata.

- **External sort.** Write decoded objects to a temporary file keyed by
  topographic level + TAI during the first pass, then read back in order during
  phase 2. Trades disk I/O for memory.

- **Chunked planning.** Process objects in dependency-level chunks — all types
  first (one streaming pass filtered to type genre), then all dependents (second
  pass). Works if the dependency graph is shallow (few levels), which is typical.

## More Information

- `import.go:17` TODO: `// TODO create an open list and resolve the graph as necessary`
- `import.go:88` TODO: `// TODO traverse object graph and rewrite all signature in topological order`
- FDR-0001: Object Locks (referenced object edges in the dependency graph)
