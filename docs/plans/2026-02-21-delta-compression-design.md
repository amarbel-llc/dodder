# Delta Compression for Inventory Archives

## Motivation

The inventory archive store (v0) stores each blob in full with optional
per-entry compression. When a repository contains many versions of similar
content (successive edits of zettels, incremental config changes), the archives
store redundant data. Delta compression encodes similar blobs as differences
against a base blob, reducing both on-disk archive size and transfer size during
remote sync.

Goals (equally weighted):

- Reduce archive size on disk by storing deltas instead of full blobs
- Reduce transfer size for remote sync operations
- Design a format that supports future evolution (delta chains, cross-archive
  references, content-type-aware packing)

## Architecture

### V1 archive format

A new v1 archive format extends v0 with per-entry encoding metadata and a delta
entry type. V0 and v1 are horizontal implementations — separate structs
implementing shared interfaces, not a single implementation with version
branches.

### Packing as a blob store capability

Packing behavior is determined by the blob store's config and implementation.
`PackableArchive` already exists as a capability interface. The v1 inventory
archive blob store implements it with delta-aware packing. The factory selects
the v0 or v1 implementation based on the blob store config version.

### Delta algorithm pluggability

The delta algorithm is an interface with a byte identifier stored per-entry in
the data file. The initial implementation uses xdelta (VCDIFF). The blob store
config declares which algorithm(s) are permitted. Future algorithms can be added
without format changes.

### Base selection pluggability

Base selection (choosing which blobs become deltas of which bases) is a pluggable
interface using a dialog pattern — the strategy queries blob metadata via a
slice-like interface and writes results via a map-like interface, avoiding
all-in-memory requirements. The initial implementation uses size-based grouping.

## On-disk format

### Data file (`<checksum>.inventory_archive-v1`)

```
[file header]
  magic:              4 bytes  "DIAR"
  version:            2 bytes  uint16 (1)
  hash_format_id_len: 1 byte
  hash_format_id:     variable (e.g. "blake2b-256")
  default_encoding:   1 byte   (0=none, 1=gzip, 2=zlib, 3=zstd)
  flags:              2 bytes  (bit 0: has_deltas,
                                bit 1: reserved_cross_archive)

[full entry]  (entry_type = 0x00)
  hash:               N bytes
  entry_type:         1 byte   0x00
  encoding:           1 byte   (compression algorithm for this entry)
  uncompressed_size:  8 bytes  uint64
  compressed_size:    8 bytes  uint64
  compressed_data:    [compressed_size bytes]

[delta entry]  (entry_type = 0x01)
  hash:               N bytes
  entry_type:         1 byte   0x01
  encoding:           1 byte   (compression applied to delta payload)
  delta_algorithm:    1 byte   (0=xdelta)
  base_hash:          N bytes
  uncompressed_size:  8 bytes  uint64 (of reconstructed blob)
  delta_size:         8 bytes  uint64 (compressed delta payload size)
  delta_data:         [delta_size bytes]

[file footer]
  entry_count: 8 bytes uint64
  checksum:    N bytes
```

The `encoding` byte per entry uses the same compression byte values as v0. The
`delta_algorithm` byte identifies the diff algorithm. `base_hash` must reference
a full entry within the same archive (self-contained constraint). The
`reserved_cross_archive` flag bit is reserved for future cross-archive base
references.

### Index file (`<checksum>.inventory_archive_index-v1`)

```
[header]
  magic:              4 bytes  "DIAX"
  version:            2 bytes  uint16 (1)
  hash_format_id_len: 1 byte
  hash_format_id:     variable
  entry_count:        8 bytes  uint64

[fan-out table]
  256 x uint64  (cumulative count by first byte of hash)

[sorted entries]  (sorted by hash bytes for binary search)
  hash:            N bytes
  pack_offset:     8 bytes uint64
  compressed_size: 8 bytes uint64
  entry_type:      1 byte  (0x00=full, 0x01=delta)
  base_offset:     8 bytes uint64 (offset of base in data file; 0 for full)

[footer]
  checksum: N bytes
```

`base_offset` is the pack offset of the base entry within the same data file.
For full entries it is 0, which is never a valid entry offset since the header
occupies byte 0. This lets the reader resolve base lookups without scanning the
data file.

### Index cache file (`index_cache-v1`)

```
[header]
  magic:              4 bytes  "DIAC"
  version:            2 bytes  uint16 (1)
  hash_format_id_len: 1 byte
  hash_format_id:     variable
  entry_count:        8 bytes  uint64

[entries]  (sorted by hash)
  hash:              N bytes
  archive_checksum:  N bytes
  offset:            8 bytes  uint64
  compressed_size:   8 bytes  uint64
  entry_type:        1 byte   (0x00=full, 0x01=delta)
  base_offset:       8 bytes  uint64

[footer]
  checksum: N bytes
```

The store can read both v0 and v1 archives simultaneously. The cache file is
rebuilt from all index files (v0 and v1) and uses the v1 format if any v1
archives exist.

### Reader reconstruction flow

1. Look up blob hash in cache -> get archive checksum, offset, entry_type,
   base_offset
2. If `entry_type == full`: seek to offset, decompress, done
3. If `entry_type == delta`: seek to base_offset, decompress base, then seek to
   offset, decompress delta payload, look up delta algorithm from entry's
   `delta_algorithm` byte, apply delta to reconstruct original blob

## Interfaces

### Delta algorithm

```go
type DeltaAlgorithm interface {
    Id() byte

    Compute(
        base domain_interfaces.BlobReader,
        baseSize int64,
        target io.Reader,
        delta io.Writer,
    ) error

    Apply(
        base domain_interfaces.BlobReader,
        baseSize int64,
        delta io.Reader,
        target io.Writer,
    ) error
}
```

`base` is a `BlobReader` rather than `io.ReaderAt` because the current
`BlobReader` implementations do not support seeking through
compression/encryption. `BlobReader` already includes `ReadAtSeeker` in its
interface definition; when compression/encryption gain seek support, delta
algorithms get random access for free. Until then, the xdelta implementation
reads the full base via `BlobReader`'s `io.Reader` into a buffer.

The initial implementation wraps a Go xdelta/VCDIFF library. A registry maps
algorithm bytes to implementations:

```go
const DeltaAlgorithmByteXdelta byte = 0

var deltaAlgorithms = map[byte]DeltaAlgorithm{...}
var deltaAlgorithmNames = map[string]byte{
    "xdelta": DeltaAlgorithmByteXdelta,
}
```

### Base selection

```go
type BlobMetadata struct {
    Id   domain_interfaces.MarklId
    Size uint64
}

type BlobSet interface {
    Len() int
    At(index int) BlobMetadata
}

type DeltaAssignments interface {
    Assign(blobIndex, baseIndex int)
    AssignError(blobIndex int, err error)
}

type BaseSelector interface {
    SelectBases(blobs BlobSet, assignments DeltaAssignments)
}
```

The packer creates a concrete `DeltaAssignments` implementation and passes it
along with a `BlobSet` wrapper around its data source. The strategy iterates via
`Len()`/`At()` and writes via `Assign()`. Errors on individual blobs are
reported via `AssignError()`.

### Size-based base selection strategy

The initial `BaseSelector` implementation:

1. Build a size-sorted index via `Len()`/`At()` (indices only, no data copying)
2. Walk the sorted list, grouping blobs within a configurable size ratio
3. Within each group, pick the largest blob as the base
4. Call `assignments.Assign(blobIndex, baseIndex)` for each delta candidate
5. Skip blobs below a minimum size threshold (delta overhead exceeds savings)
6. Skip blobs above a maximum size threshold (avoid large in-memory buffers)

Trial-and-discard lives in the packer, not the strategy. After computing a
delta, if the compressed delta is larger than the compressed full blob, the
packer discards the delta and stores the blob as a full entry.

### Delta config

```go
type DeltaConfigImmutable interface {
    GetDeltaEnabled() bool
    GetDeltaAlgorithm() string
    GetDeltaMinBlobSize() uint64
    GetDeltaMaxBlobSize() uint64
    GetDeltaSizeRatio() float64
}
```

## Configuration

### Config type: `TomlInventoryArchiveV1`

```toml
# !toml-blob_store_config-inventory_archive-v1
# ---
hash_type-id = "blake2b-256"
compression-type = "zstd"
loose-blob-store-id = "my-loose-store"

[delta]
enabled = true
algorithm = "xdelta"
min-blob-size = 256
max-blob-size = 10485760
size-ratio = 2.0
```

`TomlInventoryArchiveV1` is a new horizontal struct alongside
`TomlInventoryArchiveV0`. `TomlInventoryArchiveV0` gains an `Upgrade()` method
that produces `TomlInventoryArchiveV1` with `delta.enabled = false` — existing
stores migrate without behavior change.

### Config interface

`ConfigInventoryArchive` is extended (or a new `ConfigInventoryArchiveDelta`
interface is added):

```go
ConfigInventoryArchiveDelta interface {
    ConfigInventoryArchive
    DeltaConfigImmutable
}
```

### Type registration

| Constant | Value |
|----------|-------|
| `TypeTomlBlobStoreConfigInventoryArchiveV1` | `"!toml-blob_store_config-inventory_archive-v1"` |
| `TypeInventoryArchiveV1` | `"!inventory_archive-v1"` |
| `TypeInventoryArchiveIndexV1` | `"!inventory_archive_index-v1"` |
| `TypeDeltaAlgorithmXdelta` | `"!delta_algorithm-xdelta"` |

### Disambiguation of local hash-bucketed configs

The existing `TomlV0`, `TomlV1`, `TomlV2` in `golf/blob_store_configs/` are for
the local hash-bucketed blob store. Add comments to each file clarifying this,
with a TODO to rename them (e.g., `TomlLocalHashBucketedV0`) for disambiguation.

## Blob store implementation

### Horizontal versioning

`inventoryArchiveV0` and `inventoryArchiveV1` are separate structs, both
implementing `domain_interfaces.BlobStore` and `PackableArchive`. The factory
(`MakeBlobStore`) selects the implementation based on the config version.

- `inventoryArchiveV0.Pack()` produces v0 archives (current behavior, extracted
  from existing `inventoryArchive`)
- `inventoryArchiveV1.Pack()` produces v1 archives with delta support
- Both share common helpers for I/O, fan-out tables, checksums
- Each reads only its own archive version; the `multi` store composes them for
  coexistence

### V1 packing phases

1. **Collect** — build `BlobSet` from loose store (metadata loaded lazily)
2. **Select** — call `BaseSelector.SelectBases(blobSet, assignments)`
3. **Order** — topological sort: all full entries first, then delta entries
   (trivial with single-level deltas)
4. **Write** — two-pass write to `DataWriterV1`:
   - Pass 1 (bases + unassigned): write as full entries, record offsets
   - Pass 2 (deltas): for each assigned pair, open base and target via
     `BlobReaderFactory`, compute delta via `DeltaAlgorithm.Compute()`, compare
     compressed delta size vs compressed full size, discard delta if larger
5. **Index + Cache** — build v1 index entries with `entry_type` and
   `base_offset`, write index and cache files

### Error handling

Hard fail on first I/O or delta computation error.

```
// TODO: Collect all blob failures during packing, present summary
// to user with interactive choices (retry individual, skip to full
// entry, abort). For now, hard fail on first error.
```

## Module changes

| Module | File | Change |
|--------|------|--------|
| `echo/inventory_archive` | `data_writer_v1.go` | New: v1 data writer with delta entry support |
| `echo/inventory_archive` | `data_reader_v1.go` | New: v1 data reader with delta reconstruction |
| `echo/inventory_archive` | `index_v1.go` | New: v1 index entry type, writer, reader |
| `echo/inventory_archive` | `cache_v1.go` | New: v1 cache entry type, writer, reader |
| `echo/inventory_archive` | `delta_algorithm.go` | New: `DeltaAlgorithm` interface and registry |
| `echo/inventory_archive` | `delta_xdelta.go` | New: xdelta implementation |
| `echo/inventory_archive` | `base_selector.go` | New: `BaseSelector`, `BlobSet`, `DeltaAssignments` interfaces |
| `echo/inventory_archive` | `base_selector_size.go` | New: size-based base selection strategy |
| `echo/ids` | `types_builtin.go` | Add v1 archive and delta algorithm type constants |
| `golf/blob_store_configs` | `toml_inventory_archive_v1.go` | New: `TomlInventoryArchiveV1` with `DeltaConfig` |
| `golf/blob_store_configs` | `toml_inventory_archive_v0.go` | Add `Upgrade()` method |
| `golf/blob_store_configs` | `main.go` | Add `ConfigInventoryArchiveDelta` and `DeltaConfigImmutable` interfaces |
| `golf/blob_store_configs` | `toml_v0.go` | Add comment: local hash-bucketed store config, TODO rename |
| `golf/blob_store_configs` | `toml_v1.go` | Add comment: local hash-bucketed store config, TODO rename |
| `golf/blob_store_configs` | `toml_v2.go` | Add comment: local hash-bucketed store config, TODO rename |
| `india/blob_stores` | `store_inventory_archive_v0.go` | Extract from current `store_inventory_archive.go` |
| `india/blob_stores` | `store_inventory_archive_v1.go` | New: v1 store with delta-aware packing |
| `india/blob_stores` | `pack_v0.go` | Extract from current `pack.go` |
| `india/blob_stores` | `pack_v1.go` | New: v1 packing with delta pipeline |
| `india/blob_stores` | `main.go` | Update `MakeBlobStore` factory for config version dispatch |

## Deferred (TODOs)

- Content-type base selection strategy: madder queries dodder for blob type info
  (dodder's type system includes a `binary` flag), groups text blobs separately
  from binary for algorithm-appropriate delta encoding
- Object-history base selection strategy: dodder provides related-object hash
  chains, packer deltas successive versions of the same object against each other
- Cross-archive base references (`reserved_cross_archive` flag bit)
- Delta chain depth > 1 (deltas referencing other deltas)
- Interactive error handling during packing (collect failures, present summary
  with choices: retry, skip to full entry, abort)
- `io.ReaderAt`/`io.Seeker` support in `BlobReader` through
  compression/encryption layers
- `repack` command to convert v0 archives to v1 with delta compression
- Automatic packing on threshold (like git auto-gc)
