# Parallel Packfile Generation

## Motivation

Madder's `pack` and `pack-blobs` commands run entirely single-threaded. Two
phases dominate wall-clock time:

1. **Phase 1 (metadata collection):** `GetBlobSize` reads every loose blob
   through the full decompression/decryption pipeline and discards the data,
   just to measure size. N blobs = N sequential full-reads.
2. **Phase 2 (delta computation, V1 only):** `bsdiff.Compute()` is CPU-bound.
   Each (target, base) pair is processed serially even though the computations
   are independent once base assignments are known.

Git parallelizes its analogous delta search phase using a work-stealing
fork-join model with per-thread sliding windows. Madder's architecture is
simpler — base assignments are pre-computed by `SizeBasedSelector`, so delta
computations are embarrassingly parallel with no sliding window needed.

Goals:

- Parallelize blob size collection across both V0 and V1
- Parallelize delta computation in V1
- No additional memory beyond what the serial implementation already uses
- Preserve deterministic archive output (identical bytes for identical input)
- Scaffold concurrency in a way that supports future pipeline evolution

## Architecture

### Parallel blob size collection (V0 + V1)

A shared `collectBlobMetasParallel` function replaces the serial collection loop
in both `inventoryArchiveV0.Pack()` and `inventoryArchiveV1.Pack()`.

**Flow:**

1. Iterate `AllBlobs()` serially (the iterator is not concurrent-safe) to build
   a list of candidate blob IDs, filtering archived, null, and BlobFilter blobs
   as today.
2. Allocate `[]packedBlobMeta` with length equal to the candidate count.
3. Launch `min(runtime.NumCPU(), len(candidates))` goroutines via a buffered
   channel semaphore.
4. Each goroutine calls a `sizeFn func(MarklId) (uint64, error)` callback and
   writes the result into the pre-allocated slice at its assigned index. No
   mutex on the hot path.
5. `sync.WaitGroup` gates completion. First error cancels remaining work via a
   shared `context.Context`.
6. Sort by digest (unchanged from today).

The `sizeFn` callback is `store.GetBlobSize` today. When a fast `os.Stat`-based
path lands (see Deferred section), it plugs into the same callback signature
without changing the parallel machinery.

**Memory:** The candidate ID list is temporary and released after sizing. The
`packedBlobMeta` slice is identical in size to today's serial collection.

### Parallel delta computation (V1 only)

Within `packChunkArchiveV1`, delta computation is parallelized using
`sync.WaitGroup` + indexed result slice.

**Types:**

```go
type deltaResult struct {
    blobIdx   int
    deltaData []byte // nil = store as full (failed or delta >= original)
    err       error
}
```

**Flow:**

1. **First pass (bases + unassigned):** Unchanged — write full entries serially,
   recording offsets.
2. **Parallel delta computation:** For all `(blobIdx, baseIdx)` pairs in
   `assignments`:
   - Allocate `results := make([]deltaResult, len(assignments))`
   - Launch goroutines bounded by `runtime.NumCPU()` semaphore channel
   - Each goroutine calls `alg.Compute()`, performs trial-and-discard locally
     (if `len(delta) >= len(target)`, sets `deltaData = nil`), stores result at
     its indexed position
   - `sync.WaitGroup.Wait()`
3. **Sequential write pass:** Iterate results in deterministic order, writing
   `WriteDeltaEntry` or `WriteFullEntry` based on whether `deltaData` is nil.

**Memory:** The delta buffers contain the same data that would have been
produced serially. The blobs slice is already fully loaded for the chunk. The
`deltaResult` metadata adds only `len(assignments) * 32` bytes.

**Scaffolding:** The `deltaResult` struct and the separation of computation from
writing creates a natural seam for future evolution. A pipeline could replace
the WaitGroup with a channel-based producer/consumer, or a ring buffer could
bound concurrent delta memory.

### Concurrency control

Both parallel phases use the same pattern:

- **Worker count:** `runtime.NumCPU()` by default. No config flag initially.
- **Error propagation:** First error captured via `sync.Once`. Remaining workers
  check context cancellation before starting new work.
- **No shared mutable state on hot path:** Workers write to pre-allocated
  indexed slots.
- **Semaphore:** Buffered `chan struct{}` of capacity N limits concurrency.

### Determinism

Archive output is byte-identical for identical input:

- Size collection order does not affect archive content (results are sorted by
  digest before chunking).
- Delta results are written in the same deterministic order as the serial
  implementation (indexed by assignment position).
- The `DataWriter` remains single-threaded.

## What this design does NOT change

- Archive format, write order, checksums — identical output for identical input
- Index generation, cache writing, validation, deletion — all remain serial
- V0 delta behavior — V0 has no deltas, gains only parallel size collection
- `DataWriter` / `DataWriterV1` — no concurrency added to writers
- Chunk-level parallelism (packing multiple chunks concurrently) — deferred
  because it would require concurrent in-memory index updates

## Deferred

### Fast blob size via `os.Stat` (near-future)

`GetBlobSize` currently reads every byte of every loose blob. For a local
filesystem loose store without compression or encryption, `os.Stat` gives the
size in one syscall.

Add a `BlobSizer` capability interface:

```go
type BlobSizer interface {
    GetBlobSize(id MarklId) (uint64, error)
}
```

The local hash-bucketed store implements `BlobSizer` using `os.Stat` when no
compression/encryption is configured. The read-and-discard implementation
becomes the fallback for stores that transform data on disk.

The parallel collection function accepts a `sizeFn` callback, so this
optimization plugs in without changing the parallel machinery.

### Additional parallelization opportunities

Drawn from git's packfile architecture, applicable to madder in the future:

- **Smarter base selection with name/path hash:** Group blobs by content
  locality (filename, content prefix) before size, producing better deltas.
  Git's `type_size_sort` uses `(type, name_hash, size)` ordering.
- **Sliding window delta search:** Replace pre-computed base assignments with
  git's sliding window approach for discovering better delta candidates.
  Parallelizable using git's partition + work-stealing model.
- **Geometric repacking:** Maintain a geometric progression of pack sizes.
  Small packs merge into medium packs only when the size ratio is violated.
  Amortizes cost — each pack invocation only touches the smallest tier.
- **Chunk-level parallelism:** Pack multiple chunks concurrently, each producing
  an independent archive. Requires concurrent-safe in-memory index updates.
- **Write-phase compression pipelining:** Pre-compress blobs in worker
  goroutines and hand finished payloads to a single writer goroutine.

## References

- [Git pack-objects threading model](https://github.com/git/git/blob/master/builtin/pack-objects.c)
  — work-stealing fork-join with per-thread sliding windows
- [Git pack-heuristics](https://git-scm.com/docs/pack-heuristics) — dual-sort
  strategy for delta locality vs read locality
- [Git name-hash v2 (2.49)](https://github.blog/open-source/git/highlights-from-git-2-49/)
  — path-aware hashing for better delta grouping
- [Git geometric repacking](https://git-scm.com/docs/git-repack) — amortized
  pack maintenance via `--geometric=<factor>`
- [Git multi-pack indexes](https://git.github.io/htmldocs/technical/multi-pack-index.html)
  — unified index over multiple packfiles
