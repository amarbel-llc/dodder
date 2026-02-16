# Inventory Archive Blob Store Design

## Motivation

The existing `localHashBucketed` blob store creates one file per blob. At scale
this causes inode pressure, slow directory listings, and wasted disk space from
filesystem overhead. An inventory archive store packs many blobs into a single
file with an index for O(1) lookups, reducing file count and enabling archival
of cold data.

Goals (all equally weighted):

- Reduce file count (inode pressure, directory overhead)
- Reduce disk space (fewer per-file filesystem overheads, store-level compression)
- Enable archival / cold storage (pack infrequently-accessed blobs)

## Architecture

### Tiered write-through model

The inventory archive store is a new `BlobStore` implementation that decorates
an existing loose blob store:

- **Writes** delegate to the loose blob store (hash-bucketed, existing behavior)
- **Reads** check the archive index first, then fall through to the loose store
- **Packing** is a separate `madder pack` command that compacts loose blobs into archives

This means no changes to the write path for existing code. The archive store is
purely additive.

### Module placement

The inventory archive store lives in `india/blob_stores` alongside
`localHashBucketed` and the SFTP stores. It implements `domain_interfaces.BlobStore`.

## On-disk format

### Data file (`<checksum>.inventory_archive-v0`)

```
[file header]
  magic:              4 bytes  "DIAR"
  version:            2 bytes  uint16 (0)
  hash_format_id_len: 1 byte
  hash_format_id:     variable (e.g. "blake2b-256")
  compression:        1 byte   (0=none, 1=gzip, 2=zlib, 3=zstd)
  flags:              2 bytes  reserved

[entry 0]
  hash:               N bytes  (determined by archive's hash format)
  uncompressed_size:  8 bytes  uint64
  compressed_size:    8 bytes  uint64
  data:               [compressed_size bytes]

[entry 1]
  ...

[file footer]
  entry_count: 8 bytes uint64
  checksum:    N bytes (same hash format, over everything before footer)
```

Hash format and compression type are declared once in the header and apply to
all entries and all checksums in the file. Multi-hash stores produce separate
archives per hash type.

The filename is `<full_checksum>.inventory_archive-v0` where the checksum is the
data file's footer checksum.

### Index file (`<checksum>.inventory_archive_index-v0`)

```
[header]
  magic:              4 bytes  "DIAX"
  version:            2 bytes  uint16 (0)
  hash_format_id_len: 1 byte
  hash_format_id:     variable
  entry_count:        8 bytes  uint64

[fan-out table]
  256 x uint64  (cumulative count by first byte of hash)

[sorted entries]  (sorted by hash bytes for binary search)
  hash:            N bytes
  pack_offset:     8 bytes uint64
  compressed_size: 8 bytes uint64

[footer]
  checksum: N bytes
```

The fan-out table enables binary search by first byte of hash. The index file
shares the same base name as its data file.

### Index cache file (`index_cache-v0`)

A merged view of all per-archive index files, stored in the XDG cache directory
(not alongside durable archive files).

```
[header]
  magic:              4 bytes  "DIAC"
  version:            2 bytes  uint16 (0)
  hash_format_id_len: 1 byte
  hash_format_id:     variable
  entry_count:        8 bytes  uint64

[entries]  (sorted by hash)
  hash:              N bytes
  archive_checksum:  N bytes  (identifies which .inventory_archive-v0 file)
  offset:            8 bytes  uint64
  compressed_size:   8 bytes  uint64

[footer]
  checksum: N bytes
```

Cache lifecycle:

- **Startup:** Load `index_cache-v0` into memory. If missing or corrupt, rebuild
  from individual index files.
- **After archive write:** Append new entries to `index_cache-v0` and update the
  in-memory map.
- **Rebuild:** Read all `.inventory_archive_index-v0` files, merge, rewrite
  `index_cache-v0`. The cache is derivable and safe to delete.

## Directory layout

```
# XDG_DATA_HOME/dodder/blob_stores/
blob_stores/
  my-loose-store/
    dodder-blob_store-config       # TomlV2
    ab/cdef01...                   # loose blobs
  my-archive-store/
    dodder-blob_store-config       # TomlInventoryArchiveV0
    <checksum>.inventory_archive-v0
    <checksum>.inventory_archive_index-v0

# XDG_CACHE_HOME/dodder/blob_stores/
blob_stores/
  my-archive-store/
    index_cache-v0                 # merged index (derivable, safe to delete)
```

The cache path is derived from `envDir.Env`'s XDG access during construction,
not from `BlobStorePath`. This avoids changing the `BlobStorePath` interface.

## Configuration

### Config type: `TomlInventoryArchiveV0`

```toml
# !toml-blob_store_config-inventory_archive-v0
# ---
hash_type-id = "blake2b-256"
compression-type = "zstd"
loose-blob-store-id = "my-loose-store"
```

### Config interface

```go
ConfigInventoryArchive interface {
    Config
    ConfigHashType
    domain_interfaces.BlobIOWrapper
    GetLooseBlobStoreId() blob_store_id.Id
}
```

### Registration touchpoints

| Layer | File | Change |
|-------|------|--------|
| `echo/ids` | `types_builtin.go` | Add `TypeTomlBlobStoreConfigInventoryArchiveV0` constant and register |
| `golf/blob_store_configs` | `main.go` | Add `ConfigInventoryArchive` interface |
| `golf/blob_store_configs` | `toml_inventory_archive_v0.go` | New file: struct and methods |
| `golf/blob_store_configs` | `coding.go` | Register in `Coder` map |
| `india/blob_stores` | `store_inventory_archive.go` | New file: `inventoryArchive` struct |
| `india/blob_stores` | `main.go` | Add case in `MakeBlobStore` switch, add `BlobStoreMap` param |

## BlobStore implementation

### Struct

```go
type inventoryArchive struct {
    config         blob_store_configs.ConfigInventoryArchive
    defaultHash    markl.FormatHash
    basePath       string
    cachePath      string
    looseBlobStore domain_interfaces.BlobStore

    // merged index: hash -> archiveEntry (archive checksum, offset, size)
    // loaded from cache on startup, rebuilt from individual index files on miss
    index          map[string]archiveEntry
}
```

### Read path

1. `HasBlob(id)` — check in-memory merged index first, then delegate to loose
   blob store.
2. `MakeBlobReader(id)` — if found in index: open archive data file, seek to
   offset, decompress, return reader. Otherwise delegate to loose blob store.
3. `AllBlobs()` — union of index entries and loose blob store's `AllBlobs()`,
   deduplicated.

### Write path

`MakeBlobWriter(hashFormat)` — delegates directly to the loose blob store.

### Loose blob store wiring

`MakeBlobStore` gains an optional `BlobStoreMap` parameter so the inventory
archive store can look up its loose blob store by ID during construction.
Existing callers pass `nil`.

```go
// NOTE: blobStores parameter added to support inventory archive's
// loose-blob-store-id resolution. This couples MakeBlobStore to the
// store map, which may not scale well if more store types need
// cross-references. If this becomes a problem, switch to two-pass
// initialization: first pass creates all stores without cross-refs,
// second pass wires them up.
func MakeBlobStore(
    envDir env_dir.Env,
    configNamed blob_store_configs.ConfigNamed,
    blobStores BlobStoreMap,  // may be nil
) (store domain_interfaces.BlobStore, err error)
```

## `madder pack` command

### Usage

```
madder pack [-store <archive-store-id>] [-delete-loose]
```

- `-store`: ID of the inventory archive store to pack into. If omitted, uses the
  first inventory archive store found.
- `-delete-loose`: After packing, validate the archive and delete loose blobs
  that are now archived. Off by default.

### Algorithm

1. Resolve the inventory archive store and its loose blob store.
2. Iterate `looseBlobStore.AllBlobs()`, collecting hashes not already in the
   archive's in-memory index.
3. If no new blobs, exit early.
4. Create a temporary `.inventory_archive-v0` data file:
   - Write header (magic, version, hash format, compression from config).
   - For each new blob: read from loose store, compress, write entry
     (hash, uncompressed size, compressed size, data), track offset.
   - Write footer (entry count, checksum).
5. Compute the data file's checksum for the filename.
6. Create the `.inventory_archive_index-v0` file:
   - Sort entries by hash.
   - Build fan-out table.
   - Write header, fan-out, sorted entries, footer with checksum.
7. Atomically move both files from temp to the archive store's data directory.
8. Update `index_cache-v0` in the cache directory.
9. Update in-memory index.
10. If `-delete-loose`:
    - **Validate the archive:** Reopen the newly written archive, read every
      entry, decompress and rehash each blob. If any entry fails validation,
      abort deletion entirely and report the error.
    - **Check deletion preconditions:** Call `DeletionPrecondition` (see below).
      If it returns an error, abort deletion.
    - **Delete:** Remove each validated blob from the loose store.

### Deletion preconditions

```go
// DeletionPrecondition checks whether blobs are safe to delete from the
// loose store. The default implementation always returns nil (safe).
// Future implementations can verify off-host replication before allowing
// deletion.
type DeletionPrecondition interface {
    CheckBlobsSafeToDelete(blobs interfaces.SeqError[domain_interfaces.MarklId]) error
}
```

This interface lives in `india/blob_stores`. The `madder pack` command receives
it as a dependency. For V0 the implementation is a no-op. When off-host
verification is needed, an implementation can check remote stores' `HasBlob()`
for every blob before permitting deletion.

The interface rather than a flag/config because deletion safety is a policy
concern that varies per deployment. Some users will want N-of-M remote replica
checks, others none. The interface keeps the pack command clean and makes the
verification strategy pluggable.

### Codebase placement

| Layer | File | Purpose |
|-------|------|---------|
| `kilo/command_components_madder` | `pack.go` | Flag parsing, store resolution, pack logic |
| `lima/commands_madder` | `pack.go` | Command registration, wiring |

### Temp file handling

The pack operation writes to `envDir.GetTempLocal()` first, then moves
atomically. Crashed mid-pack leaves orphan temp files that get cleaned up
normally. No partial archives reach the store directory.

## Scope and non-goals

### In scope (V0)

- Inventory archive data + index file format
- Index cache file with rebuild-on-miss
- `inventoryArchive` BlobStore implementation (read-through to loose store)
- `TomlInventoryArchiveV0` config type with full registration
- `madder pack` command with `-delete-loose` and archive validation
- `DeletionPrecondition` interface (no-op implementation)

### Deferred

- Delta compression between similar blobs
- Automatic packing on threshold (like git auto-gc)
- Multi-store composite (1-to-many loose store references)
- Archive size limits / splitting
- Off-host replication checks in `DeletionPrecondition`
- Direct-write mode (append new blobs to archive without loose store)
