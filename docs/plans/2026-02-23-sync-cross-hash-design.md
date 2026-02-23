# Sync Cross-Hash Digest Support

## Problem

The madder `sync` command copies blobs between blob stores. When source and
destination use different hash types (e.g., sha256 vs blake2b-256), the
destination stores the blob under the source's digest. The destination's blob
lookup assumes its own hash type, so it can't find the blob.

`CopyBlobIfNecessary` verifies `writerDigest == expectedDigest` after copy, which
fails when hash types differ (different algorithms produce different digests for
the same content).

The `UseDestinationHashType` flag exists in `BlobImporter` but doesn't work
because of this verification mismatch.

## Approach

Dual-write with symlink for multi-hash stores; rehash-only for single-hash
stores.

### Multi-hash destination

1. Write blob under destination's native hash type (rehash via
   `MakeBlobWriter(dstHashType)`)
2. Create relative symlink from foreign digest path to native digest file
3. `HasBlob(foreignDigest)` returns true (symlink exists at expected path)
4. `MakeBlobReader(foreignDigest)` follows symlink, reads same bytes

### Single-hash destination

Rehash under native type. Source digest is not preserved. Requires explicit
consent (see Sync Command Changes).

## Changes

### 1. New interface: `BlobForeignDigestAdder`

**File:** `alfa/domain_interfaces/blob_store.go`

```go
type BlobForeignDigestAdder interface {
    AddForeignBlobDigestForNativeDigest(foreign, native MarklId) error
}
```

Optional interface. Stores that support foreign digest mapping implement it.
Called by `CopyBlobIfNecessary` after a successful cross-hash copy.

### 2. `CopyBlobIfNecessary` cross-hash verification

**File:** `india/blob_stores/copy.go`

After copy, verification changes:

- `readerDigest == expectedDigest` — source integrity (same hash type, always
  works)
- If cross-hash: skip `writerDigest == expectedDigest` (different algorithms,
  can't compare). Instead, call `AddForeignBlobDigestForNativeDigest` on
  destination if it implements `BlobForeignDigestAdder`.
- If same-hash: `writerDigest == expectedDigest` (unchanged)

### 3. `localHashBucketed` implements `BlobForeignDigestAdder`

**File:** `india/blob_stores/store_local_hash_bucketed.go`

Implementation:

1. If `!multiHash` — return error
2. Construct native path: `MakeHashBucketPathFromMerkleId(native, ...)`
3. Construct foreign path: `MakeHashBucketPathFromMerkleId(foreign, ...)`
4. `os.MkdirAll` for foreign parent directory
5. Compute relative symlink target (`filepath.Rel`)
6. `os.Symlink(relativeTarget, foreignPath)`

No existence check needed — `HasBlob(foreignDigest)` short-circuits the entire
copy flow before `AddForeignBlobDigestForNativeDigest` is reached.

Example on disk:

```
basePath/blake2b256/aa/bbccdd...   (native blob file)
basePath/sha256/xx/yyzzzz...       (foreign symlink → ../../blake2b256/aa/bbccdd...)
```

### 4. `blobReaderFrom` fix for multi-hash correctness

**File:** `india/blob_stores/store_local_hash_bucketed.go`

Existing bug: `blobReaderFrom` passes `makeEnvDirConfig(nil)`, always using the
store's default hash format for the reader's digester. When reading a blob via a
foreign digest symlink (or any non-default hash type), the digester computes the
wrong hash.

Fix: pass the digest's own format to `makeEnvDirConfig` instead of `nil`.

### 5. Sync command changes

**File:** `lima/commands_madder/sync.go`

New flag: `-allow-rehashing` (bool, default false).

Cross-hash behavior matrix:

| Destination | `-allow-rehashing` | Interactive | Result                    |
|-------------|--------------------|-------------|---------------------------|
| Multi-hash  | any                | any         | Copy + symlink. No prompt |
| Single-hash | true               | any         | Rehash. Source digest lost |
| Single-hash | false              | yes         | Prompt user for consent   |
| Single-hash | false              | no          | Error with explanation    |

Detection: compare `source.GetDefaultHashType()` vs
`destination.GetDefaultHashType()` once upfront before the copy loop.

### 6. `BlobImporter` changes

**File:** `kilo/blob_transfers/main.go`

Set `UseDestinationHashType = true` when cross-hash is detected and permitted.

### 7. TAP-14 output for sync command (separate task)

Convert sync command from `ui.Out().Print` / `ui.Err().Printf` to TAP-14 using
`tap.NewWriter`. Separate implementation task.

## Archive stores

Archive stores do NOT implement `BlobForeignDigestAdder` in this iteration.
Packed blobs are only findable by native digest. Loose blobs within an archive
store inherit behavior from the embedded `localHashBucketed` store.

## Future work

Archive store foreign digest support via loose-to-packfile symlinks (see
TODO.md).
