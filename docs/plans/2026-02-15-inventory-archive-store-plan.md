# Inventory Archive Blob Store Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an inventory archive blob store that packs loose blobs into archive files for space efficiency, with a `madder pack` command to trigger packing.

**Architecture:** A new `BlobStore` implementation in `india/blob_stores` that decorates an existing loose blob store. Reads check archive index first, writes delegate to loose store. A new binary format stores packed blobs with per-archive and merged cache indexes. Config, type registration, and coder follow the existing `TomlV2`/`TomlPointerV0` patterns.

**Tech Stack:** Go, binary encoding (`encoding/binary`), existing compression (`charlie/compression_type`), existing hash (`echo/markl`), TOML config via triple-hyphen-io.

**Reference:** See `docs/plans/2026-02-15-inventory-archive-store-design.md` for the full design rationale.

---

### Task 1: Register the type constant

**Files:**
- Modify: `go/src/echo/ids/types_builtin.go`

**Step 1: Add the type constant**

Add after line 31 (`TypeTomlBlobStoreConfigPointerV0`), before `TypeTomlBlobStoreConfigVCurrent`:

```go
TypeTomlBlobStoreConfigInventoryArchiveV0 = "!toml-blob_store_config-inventory_archive-v0"
```

**Step 2: Register in init()**

Add after the `TypeTomlBlobStoreConfigPointerV0` registration block (after line 87):

```go
registerBuiltinTypeString(
    TypeTomlBlobStoreConfigInventoryArchiveV0,
    genres.Unknown,
    false,
)
```

**Step 3: Verify it compiles**

Run: `cd go && go build ./src/echo/ids/`
Expected: success, no errors.

**Step 4: Commit**

```
feat(echo/ids): register inventory archive blob store config type
```

---

### Task 2: Add the config interface and struct

**Files:**
- Modify: `go/src/golf/blob_store_configs/main.go`
- Create: `go/src/golf/blob_store_configs/toml_inventory_archive_v0.go`

**Step 1: Add the `ConfigInventoryArchive` interface**

In `go/src/golf/blob_store_configs/main.go`, add after the `ConfigPointer` interface block (after line 63):

```go
ConfigInventoryArchive interface {
    configLocal
    ConfigHashType
    domain_interfaces.BlobIOWrapper
    GetLooseBlobStoreId() blob_store_id.Id
}
```

This requires `blob_store_id` to be imported — it's already imported transitively via `directory_layout` but add the direct import if needed.

**Step 2: Create `toml_inventory_archive_v0.go`**

Create `go/src/golf/blob_store_configs/toml_inventory_archive_v0.go`. Follow the `TomlV2` pattern for hash/compression and `TomlPointerV0` pattern for referencing another store:

```go
package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

type TomlInventoryArchiveV0 struct {
	HashTypeId        string                           `toml:"hash_type-id"`
	CompressionType   compression_type.CompressionType `toml:"compression-type"`
	LooseBlobStoreId  blob_store_id.Id                 `toml:"loose-blob-store-id"`
	Encryption        markl.Id                         `toml:"encryption"`
}

var (
	_ ConfigInventoryArchive = TomlInventoryArchiveV0{}
	_ ConfigMutable          = &TomlInventoryArchiveV0{}
	_                        = registerToml[TomlInventoryArchiveV0](
		Coder.Blob,
		ids.TypeTomlBlobStoreConfigInventoryArchiveV0,
	)
)

func (TomlInventoryArchiveV0) GetBlobStoreType() string {
	return "local-inventory-archive"
}

func (config *TomlInventoryArchiveV0) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	config.CompressionType.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&config.HashTypeId,
		"hash_type-id",
		markl.FormatIdHashBlake2b256,
		"hash type for archive checksums and blob hashes",
	)

	flagSet.Var(
		&config.LooseBlobStoreId,
		"loose-blob-store-id",
		"id of the loose blob store to read from and write to",
	)
}

func (config TomlInventoryArchiveV0) getBasePath() string {
	return ""
}

func (config TomlInventoryArchiveV0) SupportsMultiHash() bool {
	return false
}

func (config TomlInventoryArchiveV0) GetDefaultHashTypeId() string {
	return config.HashTypeId
}

func (config TomlInventoryArchiveV0) GetBlobCompression() interfaces.IOWrapper {
	return &config.CompressionType
}

func (config TomlInventoryArchiveV0) GetBlobEncryption() domain_interfaces.MarklId {
	return config.Encryption
}

func (config TomlInventoryArchiveV0) GetLooseBlobStoreId() blob_store_id.Id {
	return config.LooseBlobStoreId
}
```

**Step 3: Verify it compiles**

Run: `cd go && go build ./src/golf/blob_store_configs/`
Expected: success.

**Step 4: Commit**

```
feat(golf/blob_store_configs): add TomlInventoryArchiveV0 config type
```

Note: The `registerToml` call in the `var` block auto-registers with the Coder, so no separate edit to `coding.go` is needed — `registerToml` appends to the same `Coder.Blob` map at init time.

---

### Task 3: Add `BlobStoreMap` parameter to `MakeBlobStore`

This is a refactor-only task — no new behavior, just threading the parameter through.

**Files:**
- Modify: `go/src/india/blob_stores/main.go`
- Modify: `go/src/kilo/command_components_madder/blob_store.go`

**Step 1: Change `MakeBlobStore` signature**

In `go/src/india/blob_stores/main.go`, change the signature at line 136:

From:
```go
func MakeBlobStore(
	envDir env_dir.Env,
	configNamed blob_store_configs.ConfigNamed,
) (store domain_interfaces.BlobStore, err error) {
```

To:
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
	blobStores BlobStoreMap,
) (store domain_interfaces.BlobStore, err error) {
```

**Step 2: Update internal callers in `main.go`**

Line 100 (inside `MakeBlobStores`): already has access to `blobStores` local variable. Change:
```go
if blobStore.BlobStore, err = MakeBlobStore(
    envDir,
    blobStore.ConfigNamed,
); err != nil {
```
To:
```go
if blobStore.BlobStore, err = MakeBlobStore(
    envDir,
    blobStore.ConfigNamed,
    blobStores,
); err != nil {
```

Line 123 (inside `MakeRemoteBlobStore`): pass `nil`:
```go
if blobStore.BlobStore, err = MakeBlobStore(
    envDir,
    configNamed,
    nil,
); err != nil {
```

Line 202 (pointer config recursive call): pass `blobStores` through:
```go
return MakeBlobStore(envDir, configNamed, blobStores)
```

**Step 3: Update external callers in `kilo/command_components_madder/blob_store.go`**

Line 47: pass `nil`:
```go
if blobStore.BlobStore, err = blob_stores.MakeBlobStore(
    envBlobStore,
    blobStore.ConfigNamed,
    nil,
); err != nil {
```

Line 100: pass `nil`:
```go
if blobStore.BlobStore, err = blob_stores.MakeBlobStore(
    envBlobStore,
    blobStore.ConfigNamed,
    nil,
); err != nil {
```

**Step 4: Verify it compiles**

Run: `cd go && go build ./...`
Expected: success. No behavior change.

**Step 5: Commit**

```
refactor(india/blob_stores): thread BlobStoreMap through MakeBlobStore

Preparation for inventory archive store which needs to look up its
loose blob store by ID during construction.
```

---

### Task 4: Binary format types — archive data file writer/reader

This task creates a new package for the binary archive format at a layer low enough for both the store and the pack command to use.

**Files:**
- Create: `go/src/charlie/inventory_archive/data_writer.go`
- Create: `go/src/charlie/inventory_archive/data_reader.go`
- Create: `go/src/charlie/inventory_archive/types.go`
- Create: `go/src/charlie/inventory_archive/data_writer_test.go`

`charlie` is appropriate because it depends on `compression_type` (also in `charlie`) and basic types. The `echo/markl` dependency for hash format puts it at `echo` minimum, but the format types themselves can use raw byte slices and string IDs for the hash format, keeping the binary codec in `charlie` and letting higher layers convert to/from `markl` types.

Actually, because we need `markl.FormatHash` and `markl.GetFormatHashOrError`, place this package in `echo/inventory_archive` instead.

**Files (revised):**
- Create: `go/src/echo/inventory_archive/types.go`
- Create: `go/src/echo/inventory_archive/data_writer.go`
- Create: `go/src/echo/inventory_archive/data_reader.go`
- Create: `go/src/echo/inventory_archive/data_writer_test.go`

**Step 1: Write the test for round-tripping a data file**

Create `go/src/echo/inventory_archive/data_writer_test.go`:

```go
package inventory_archive

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

func TestDataFileRoundTrip(t *testing.T) {
	hashFormatId := "sha256"
	hashSize := 32
	comp := compression_type.CompressionTypeNone

	blob1 := []byte("hello world")
	blob2 := []byte("goodbye world")
	hash1 := make([]byte, hashSize)
	hash2 := make([]byte, hashSize)
	// fill with recognizable patterns
	for i := range hash1 {
		hash1[i] = byte(i)
	}
	for i := range hash2 {
		hash2[i] = byte(i + 32)
	}

	// Write
	var buf bytes.Buffer
	w, err := NewDataWriter(&buf, hashFormatId, comp)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.WriteEntry(hash1, blob1); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteEntry(hash2, blob2); err != nil {
		t.Fatal(err)
	}

	checksum, err := w.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(checksum) != hashSize {
		t.Fatalf("expected checksum length %d, got %d", hashSize, len(checksum))
	}

	// Read
	reader := bytes.NewReader(buf.Bytes())
	r, err := NewDataReader(reader)
	if err != nil {
		t.Fatal(err)
	}

	if r.HashFormatId() != hashFormatId {
		t.Fatalf("expected hash format %q, got %q", hashFormatId, r.HashFormatId())
	}

	entries, err := r.ReadAllEntries()
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if !bytes.Equal(entries[0].Hash, hash1) {
		t.Fatal("hash1 mismatch")
	}
	if !bytes.Equal(entries[0].Data, blob1) {
		t.Fatal("blob1 data mismatch")
	}
	if !bytes.Equal(entries[1].Hash, hash2) {
		t.Fatal("hash2 mismatch")
	}
	if !bytes.Equal(entries[1].Data, blob2) {
		t.Fatal("blob2 data mismatch")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/`
Expected: compilation failure — types don't exist yet.

**Step 3: Implement types.go**

Create `go/src/echo/inventory_archive/types.go`:

```go
package inventory_archive

const (
	DataFileMagic  = "DIAR"
	IndexFileMagic = "DIAX"
	CacheFileMagic = "DIAC"

	DataFileVersion  uint16 = 0
	IndexFileVersion uint16 = 0
	CacheFileVersion uint16 = 0

	DataFileExtension  = ".inventory_archive-v0"
	IndexFileExtension = ".inventory_archive_index-v0"
	CacheFileName      = "index_cache-v0"

	CompressionByteNone byte = 0
	CompressionByteGzip byte = 1
	CompressionByteZlib byte = 2
	CompressionByteZstd byte = 3
)

type DataEntry struct {
	Hash             []byte
	UncompressedSize uint64
	CompressedSize   uint64
	Data             []byte
	Offset           uint64 // populated during write/read for index building
}

type IndexEntry struct {
	Hash           []byte
	PackOffset     uint64
	CompressedSize uint64
}

type CacheEntry struct {
	Hash             []byte
	ArchiveChecksum  []byte
	Offset           uint64
	CompressedSize   uint64
}
```

Add compression byte mapping functions:

```go
import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

func CompressionToByte(ct compression_type.CompressionType) byte {
	switch ct {
	case compression_type.CompressionTypeGzip:
		return CompressionByteGzip
	case compression_type.CompressionTypeZlib:
		return CompressionByteZlib
	case compression_type.CompressionTypeZstd:
		return CompressionByteZstd
	default:
		return CompressionByteNone
	}
}

func ByteToCompression(b byte) (compression_type.CompressionType, error) {
	switch b {
	case CompressionByteNone:
		return compression_type.CompressionTypeNone, nil
	case CompressionByteGzip:
		return compression_type.CompressionTypeGzip, nil
	case CompressionByteZlib:
		return compression_type.CompressionTypeZlib, nil
	case CompressionByteZstd:
		return compression_type.CompressionTypeZstd, nil
	default:
		return "", errors.Errorf("unknown compression byte: %d", b)
	}
}
```

**Step 4: Implement data_writer.go**

Create `go/src/echo/inventory_archive/data_writer.go`. The writer:
- Writes header (magic, version, hash_format_id_len, hash_format_id, compression byte, 2 reserved bytes)
- For each entry: writes hash, uncompressed_size, compressed_size, then compressed data
- On Close: writes entry_count (uint64), computes checksum of everything written so far, writes checksum, returns checksum bytes

Use `encoding/binary` with `binary.BigEndian` for all integers. The writer wraps an `io.Writer` and hashes everything through a `hash.Hash` from the standard library (e.g. `sha256.New()` or `blake2b.New256()`). Use `markl.FormatHash` to get the hasher — or since we're in `echo`, we can use the markl hash format ID string and resolve it ourselves.

Implementation outline:
```go
package inventory_archive

import (
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

type DataWriter struct {
	writer       io.Writer
	hashWriter   markl.Hash // hashes everything written
	hashFormatId string
	hashSize     int
	compression  compression_type.CompressionType
	entryCount   uint64
	entries      []DataEntry // track for index building
}
```

The writer hashes every byte written via `io.MultiWriter(underlying, hashWriter)` so the footer checksum covers all prior content.

**Step 5: Implement data_reader.go**

Create `go/src/echo/inventory_archive/data_reader.go`. The reader:
- Reads and validates header (magic, version, hash format, compression)
- Provides `ReadEntry()` iterator and `ReadAllEntries()` convenience
- Validates footer checksum

Use `io.ReaderAt` + `io.Seeker` for random access (needed by the store for seeking to specific offsets).

**Step 6: Run tests**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/`
Expected: PASS.

**Step 7: Commit**

```
feat(echo/inventory_archive): implement data file writer and reader

Binary format for packing blobs into inventory archive files.
Includes round-trip test.
```

---

### Task 5: Binary format — index file writer/reader

**Files:**
- Create: `go/src/echo/inventory_archive/index_writer.go`
- Create: `go/src/echo/inventory_archive/index_reader.go`
- Create: `go/src/echo/inventory_archive/index_test.go`

**Step 1: Write the test**

Create `go/src/echo/inventory_archive/index_test.go`:

Test that:
- Given a set of `IndexEntry` values, write an index file
- Read it back
- Fan-out table correctly partitions by first byte
- Binary search by hash returns correct offset and size
- Footer checksum validates

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug -run TestIndex ./src/echo/inventory_archive/`
Expected: compilation failure.

**Step 3: Implement index_writer.go**

The writer:
- Accepts sorted `[]IndexEntry` (caller sorts before writing)
- Writes header (magic, version, hash_format_id, entry_count)
- Builds and writes fan-out table (256 x uint64 cumulative counts)
- Writes sorted entries (hash, pack_offset, compressed_size)
- Writes footer checksum

**Step 4: Implement index_reader.go**

The reader:
- Reads and validates header
- Loads fan-out table
- Provides `LookupHash(hash []byte) (offset uint64, compressedSize uint64, found bool)` using fan-out + binary search
- Provides `ReadAllEntries() []IndexEntry` for cache rebuilding
- Validates footer checksum

**Step 5: Run tests**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/`
Expected: PASS.

**Step 6: Commit**

```
feat(echo/inventory_archive): implement index file writer and reader

Fan-out table with binary search for O(1) hash lookups.
```

---

### Task 6: Binary format — cache file writer/reader

**Files:**
- Create: `go/src/echo/inventory_archive/cache_writer.go`
- Create: `go/src/echo/inventory_archive/cache_reader.go`
- Create: `go/src/echo/inventory_archive/cache_test.go`

**Step 1: Write the test**

Test round-trip: write cache entries (hash, archive_checksum, offset, compressed_size), read them back, verify checksum.

**Step 2: Run test to verify it fails**

**Step 3: Implement cache_writer.go and cache_reader.go**

Simpler than the index file — no fan-out table. Sorted entries with a footer checksum. The reader returns a `map[string]CacheEntry` keyed by hex-encoded hash for the in-memory index.

**Step 4: Run tests**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/`
Expected: PASS.

**Step 5: Commit**

```
feat(echo/inventory_archive): implement cache file writer and reader
```

---

### Task 7: Implement `inventoryArchive` BlobStore — construction and delegation

**Files:**
- Create: `go/src/india/blob_stores/store_inventory_archive.go`
- Modify: `go/src/india/blob_stores/main.go`

**Step 1: Create the store struct and constructor**

Create `go/src/india/blob_stores/store_inventory_archive.go`:

```go
package blob_stores

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
)

type archiveEntry struct {
	ArchiveChecksum string // hex filename stem
	Offset          uint64
	CompressedSize  uint64
}

type inventoryArchive struct {
	config         blob_store_configs.ConfigInventoryArchive
	defaultHash    markl.FormatHash
	basePath       string
	cachePath      string
	looseBlobStore domain_interfaces.BlobStore
	index          map[string]archiveEntry // keyed by hex hash
}

var _ domain_interfaces.BlobStore = inventoryArchive{}

func makeInventoryArchive(
	envDir env_dir.Env,
	basePath string,
	config blob_store_configs.ConfigInventoryArchive,
	looseBlobStore domain_interfaces.BlobStore,
) (store inventoryArchive, err error) {
	store.config = config
	store.looseBlobStore = looseBlobStore
	store.basePath = basePath

	if store.defaultHash, err = markl.GetFormatHashOrError(
		config.GetDefaultHashTypeId(),
	); err != nil {
		err = errors.Wrap(err)
		return store, err
	}

	// Derive cache path from XDG
	// store.cachePath = ... (from envDir XDG cache dir + blob store id)

	// Load index from cache or rebuild
	store.index = make(map[string]archiveEntry)
	// TODO: load from cache file, rebuild from index files on miss

	return store, err
}
```

**Step 2: Implement delegating methods**

```go
func (store inventoryArchive) GetBlobStoreDescription() string {
	return "local inventory archive"
}

func (store inventoryArchive) GetBlobIOWrapper() domain_interfaces.BlobIOWrapper {
	return store.config
}

func (store inventoryArchive) GetDefaultHashType() domain_interfaces.FormatHash {
	return store.defaultHash
}

func (store inventoryArchive) HasBlob(id domain_interfaces.MarklId) bool {
	if id.IsNull() {
		return true
	}
	hexHash := id.String()
	if _, ok := store.index[hexHash]; ok {
		return true
	}
	return store.looseBlobStore.HasBlob(id)
}

func (store inventoryArchive) MakeBlobWriter(
	hashFormat domain_interfaces.FormatHash,
) (domain_interfaces.BlobWriter, error) {
	return store.looseBlobStore.MakeBlobWriter(hashFormat)
}

func (store inventoryArchive) AllBlobs() interfaces.SeqError[domain_interfaces.MarklId] {
	// TODO: union of index entries and loose store, deduplicated
	return store.looseBlobStore.AllBlobs()
}

func (store inventoryArchive) MakeBlobReader(
	id domain_interfaces.MarklId,
) (domain_interfaces.BlobReader, error) {
	if id.IsNull() {
		return store.looseBlobStore.MakeBlobReader(id)
	}

	hexHash := id.String()
	entry, ok := store.index[hexHash]
	if !ok {
		return store.looseBlobStore.MakeBlobReader(id)
	}

	// TODO: open archive file, seek to entry.Offset, decompress, return reader
	_ = entry
	return store.looseBlobStore.MakeBlobReader(id)
}
```

**Step 3: Wire into `MakeBlobStore` factory**

In `go/src/india/blob_stores/main.go`, add a new case before `default` in the switch (before line 204):

```go
case blob_store_configs.ConfigInventoryArchive:
    var looseBlobStore domain_interfaces.BlobStore
    if blobStores != nil {
        looseBlobStoreId := config.GetLooseBlobStoreId().String()
        if initialized, ok := blobStores[looseBlobStoreId]; ok {
            looseBlobStore = initialized.BlobStore
        }
    }
    if looseBlobStore == nil {
        err = errors.BadRequestf(
            "inventory archive store requires loose-blob-store-id %q but it was not found",
            config.GetLooseBlobStoreId(),
        )
        return store, err
    }
    return makeInventoryArchive(
        envDir,
        configNamed.Path.GetBase(),
        config,
        looseBlobStore,
    )
```

**Step 4: Verify it compiles**

Run: `cd go && go build ./...`
Expected: success.

**Step 5: Commit**

```
feat(india/blob_stores): add inventoryArchive BlobStore skeleton

Delegates writes to loose store, reads fall through to loose store.
Archive reading is stubbed for now.
```

---

### Task 8: Implement archive reading in `MakeBlobReader`

**Files:**
- Modify: `go/src/india/blob_stores/store_inventory_archive.go`

**Step 1: Write a test for reading a blob from an archive**

Create `go/src/india/blob_stores/store_inventory_archive_test.go` (or put the test in the `inventory_archive` package). The test should:
- Create a temporary directory
- Write a small archive data file using `inventory_archive.DataWriter`
- Write the corresponding index file
- Construct an `inventoryArchive` store with the index loaded
- Call `MakeBlobReader` for a hash that's in the archive
- Verify the returned data matches

**Step 2: Run test to verify it fails**

**Step 3: Implement `MakeBlobReader` for archived blobs**

Replace the TODO stub. Open the archive file at `basePath/<archiveChecksum>.inventory_archive-v0`, seek to `entry.Offset`, read the entry header to get compressed size, decompress using the store's compression type, and return a reader.

Use `os.Open` + `io.NewSectionReader` for the seek, then wrap with the compression reader from `compression_type.WrapReader`.

**Step 4: Run tests**

**Step 5: Commit**

```
feat(india/blob_stores): implement archive blob reading
```

---

### Task 9: Index loading — cache file and rebuild

**Files:**
- Modify: `go/src/india/blob_stores/store_inventory_archive.go`

**Step 1: Implement `loadIndex` method**

```go
func (store *inventoryArchive) loadIndex() error
```

Logic:
1. Try to read `cachePath/index_cache-v0` using `inventory_archive.CacheReader`
2. If successful, populate `store.index` from cache entries
3. If cache is missing or corrupt, call `store.rebuildIndex()`

**Step 2: Implement `rebuildIndex` method**

```go
func (store *inventoryArchive) rebuildIndex() error
```

Logic:
1. Glob `basePath/*.inventory_archive_index-v0`
2. For each index file, read all entries using `inventory_archive.IndexReader`
3. Merge into `store.index`
4. Write merged index to `cachePath/index_cache-v0` using `inventory_archive.CacheWriter`

**Step 3: Call `loadIndex` from constructor**

Replace the TODO in `makeInventoryArchive`.

**Step 4: Verify with a test**

Write a test that creates archive + index files on disk, constructs the store, and verifies `HasBlob` returns true for archived hashes.

**Step 5: Commit**

```
feat(india/blob_stores): implement index loading with cache rebuild
```

---

### Task 10: Implement `AllBlobs` with deduplication

**Files:**
- Modify: `go/src/india/blob_stores/store_inventory_archive.go`

**Step 1: Implement `AllBlobs`**

Return a `SeqError[MarklId]` that yields:
1. All hashes from the in-memory archive index
2. All hashes from the loose blob store's `AllBlobs()`
3. Deduplicated (skip loose hashes already in the archive index)

Follow the iterator patterns used elsewhere in the codebase (check how `localAllBlobs` works in `store_local_hash_bucketed.go` for the `SeqError` return pattern).

**Step 2: Test**

**Step 3: Commit**

```
feat(india/blob_stores): implement AllBlobs with archive+loose dedup
```

---

### Task 11: `DeletionPrecondition` interface and no-op implementation

**Files:**
- Create: `go/src/india/blob_stores/deletion_precondition.go`

**Step 1: Create the interface and no-op**

```go
package blob_stores

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
)

// DeletionPrecondition checks whether blobs are safe to delete from the
// loose store. The default implementation always returns nil (safe).
// Future implementations can verify off-host replication before allowing
// deletion.
type DeletionPrecondition interface {
	CheckBlobsSafeToDelete(
		blobs interfaces.SeqError[domain_interfaces.MarklId],
	) error
}

type nopDeletionPrecondition struct{}

func (nopDeletionPrecondition) CheckBlobsSafeToDelete(
	blobs interfaces.SeqError[domain_interfaces.MarklId],
) error {
	return nil
}

func NopDeletionPrecondition() DeletionPrecondition {
	return nopDeletionPrecondition{}
}
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./src/india/blob_stores/`

**Step 3: Commit**

```
feat(india/blob_stores): add DeletionPrecondition interface

No-op implementation for V0. Future implementations will verify
off-host replication before allowing loose blob deletion.
```

---

### Task 12: `madder pack` command — basic packing

**Files:**
- Create: `go/src/kilo/command_components_madder/pack.go`
- Create: `go/src/lima/commands_madder/pack.go`

**Step 1: Create the command component**

Create `go/src/kilo/command_components_madder/pack.go` with:
- A `Pack` struct with fields for the archive store ID and delete-loose flag
- A `RunPack` method containing the core pack algorithm:
  1. Iterate loose blobs not in archive index
  2. Write data file via `DataWriter`
  3. Build and write index file via `IndexWriter`
  4. Move to final paths atomically
  5. Update cache file
  6. If delete-loose: validate archive, check preconditions, delete

**Step 2: Create the command**

Create `go/src/lima/commands_madder/pack.go`:

```go
package commands_madder

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
)

func init() {
	utility.AddCmd("pack", &Pack{})
}

type Pack struct {
	command_components_madder.EnvBlobStore

	StoreId     blob_store_id.Id
	DeleteLoose bool
}

var _ interfaces.CommandComponentWriter = (*Pack)(nil)

func (cmd *Pack) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.Var(&cmd.StoreId, "store", "inventory archive store id")
	flagSet.BoolVar(&cmd.DeleteLoose, "delete-loose", false,
		"validate archive then delete packed loose blobs")
}

func (cmd Pack) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	// Resolve the inventory archive store from StoreId or find first one
	// Call pack logic from command_components_madder
	_ = envBlobStore
}
```

**Step 3: Verify it compiles**

Run: `cd go && go build ./cmd/madder/`

**Step 4: Commit**

```
feat(lima/commands_madder): add madder pack command skeleton
```

---

### Task 13: `madder pack` — implement core pack logic

**Files:**
- Modify: `go/src/kilo/command_components_madder/pack.go`
- Modify: `go/src/lima/commands_madder/pack.go`

**Step 1: Implement the pack algorithm**

In the command component, implement the full pack flow:
1. Get the `inventoryArchive` store and its loose blob store
2. Collect unpacked blob hashes
3. Write data file to temp, tracking entries
4. Write index file to temp
5. Move both to final paths
6. Update cache

**Step 2: Write a BATS integration test**

Create `zz-tests_bats/pack.bats`:
- Set up a repo with a loose blob store and an inventory archive store
- Write some blobs
- Run `madder pack`
- Verify archive files exist
- Verify blobs are still readable

**Step 3: Run tests**

Run: `just test-bats-targets pack.bats`

**Step 4: Commit**

```
feat(madder pack): implement core packing algorithm
```

---

### Task 14: `madder pack -delete-loose` — validation and deletion

**Files:**
- Modify: `go/src/kilo/command_components_madder/pack.go`

**Step 1: Implement archive validation**

After packing, if `-delete-loose` is set:
1. Reopen the newly written archive
2. Read every entry, decompress, rehash
3. Compare against expected hash
4. If any mismatch, abort with error

**Step 2: Wire `DeletionPrecondition`**

Call `NopDeletionPrecondition().CheckBlobsSafeToDelete()` with the packed blob hashes. If it returns error, abort.

**Step 3: Implement loose blob deletion**

For each validated blob, delete the loose file from the loose blob store's directory. Use the hash-bucketed path convention to locate the file.

**Step 4: Write a BATS test**

Test that:
- After `madder pack -delete-loose`, loose files are gone
- Blobs are still readable from the archive
- If archive is corrupted before deletion, deletion is aborted

**Step 5: Run tests**

**Step 6: Commit**

```
feat(madder pack): add -delete-loose with archive validation
```

---

### Task 15: Format and final verification

**Step 1: Format all new files**

Run: `cd go && just codemod-go-fmt`

**Step 2: Run full test suite**

Run: `just test`

**Step 3: Fix any issues**

**Step 4: Commit any formatting fixes**

```
style: format inventory archive files
```
