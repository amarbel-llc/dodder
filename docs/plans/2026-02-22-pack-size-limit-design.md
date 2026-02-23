# Pack Size Limit Design

## Problem

The `madder pack` command loads all loose blob data into memory at once, then
buffers the entire serialized archive before writing to disk. Peak memory is
~2-3x the total loose blob size. There is no upper bound.

## Design

### Goal

Bound peak memory by splitting pack operations when the sum of loose blob data
exceeds a configurable limit. Multiple smaller archive files are produced instead
of one large one.

### Config

Add `MaxPackSize uint64` (TOML: `max-pack-size`) as a top-level field on
`TomlInventoryArchiveV1` and `TomlInventoryArchiveV2`. Add `GetMaxPackSize()`
to the `ConfigInventoryArchiveDelta` interface. Also add `GetMaxPackSize()` to
`ConfigInventoryArchive` so V0 packing respects the limit.

- Default: 536870912 (512 MiB)
- `0` means unlimited (no splitting), matching git's `pack.packSizeLimit`
  convention
- The limit applies to the sum of raw loose blob data, not compressed archive
  size on disk

The V0 -> V1 upgrade path sets the default value.

### Pack Splitting

Both V0 and V1 `Pack()` methods currently:

1. Collect all loose blobs into `[]packedBlob`
2. Sort by hash
3. Write one archive (data file + index + cache)

The change:

1. After collecting and sorting (existing Phase 1), split into chunks by
   walking the sorted slice and accumulating data sizes. When the running total
   would exceed `MaxPackSize`, start a new chunk.
2. Loop over each chunk, running the existing archive-writing logic per chunk.

Hash sort order is preserved within each chunk. Each archive gets its own
checksum, index file, and cache entries. The in-memory index and cache are
already append-friendly (maps and slices that accumulate), so multi-pack works
without changes to index/cache update logic.

### Semantics

- When `BlobFilter` is set (from `pack-objects`), the limit applies to the
  filtered set
- A single blob larger than `MaxPackSize` still gets packed (the limit is
  per-archive, not per-blob) -- the chunk just contains that one blob

### Git Reference

Git's `pack.packSizeLimit` defaults to 0 (unlimited), minimum 1 MiB. It was
designed for storage media capacity, not memory bounding. Git notes that using
it "may result in a larger and slower repository" and prevents bitmap index
creation. Our use case differs (memory bounding), but the 0-means-unlimited
convention is reused.

Sources:
- https://git-scm.com/docs/git-pack-objects
- https://git-scm.com/docs/git-repack
