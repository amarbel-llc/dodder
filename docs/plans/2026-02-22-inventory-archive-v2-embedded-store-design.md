# Inventory Archive V2: Embedded Hash-Bucketed Store

## Problem

Inventory archive V0/V1 stores reference an external loose blob store by ID.
This creates a coupling: the archive depends on a separately-configured store
for writes and for the loose blobs that get packed. There is no self-contained
archive store.

## Design

### Config: `TomlInventoryArchiveV2`

Same fields as V1 minus `LooseBlobStoreId`. Implements `ConfigInventoryArchiveDelta`
with `GetLooseBlobStoreId()` returning zero value.

```toml
# type: !toml-blob_store_config-inventory_archive-v2
hash_type-id = "blake2b-256"
compression-type = "zstd"

[delta]
enabled = false
algorithm = "bsdiff"
min-blob-size = 256
max-blob-size = 10485760
size-ratio = 2.0
```

No new interfaces. The embedded store inherits hash type, compression, and
encryption from the archive config.

### Factory: embedded store construction

In `MakeBlobStore`, the `ConfigInventoryArchiveDelta` case checks
`config.GetLooseBlobStoreId().IsEmpty()`. If empty, constructs an embedded
`localHashBucketed` at `<basePath>/loose/` using the archive's settings and
default bucket depth `[2]`. Otherwise, looks up the external store as before.

### Store struct

Reuses `inventoryArchiveV1`. The embedded vs external loose store is transparent
once constructed.

### Init command

`madder init-inventory-archive-embedded` registered with
`TypeTomlBlobStoreConfigInventoryArchiveV2`.

### Unchanged

Pack, delete-loose, validation, delta selection, read path, index, and cache
are all unchanged. The only difference is where the loose store comes from.

## Type registration

- New constant: `TypeTomlBlobStoreConfigInventoryArchiveV2` in `echo/ids/types_builtin.go`
- Register in `init()` of the same file
- `TypeTomlBlobStoreConfigInventoryArchiveVCurrent` updated to point to V2
- Coder registration in `golf/blob_store_configs/`

## Upgrade path

`TomlInventoryArchiveV1.Upgrade()` returns `TomlInventoryArchiveV2` with delta
config preserved. Since V2 has no `LooseBlobStoreId`, the upgrade drops it and
the store becomes self-contained. Existing V1 configs continue to work as-is.
