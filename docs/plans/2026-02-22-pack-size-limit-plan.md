# Pack Size Limit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Split pack files when the sum of loose blob data exceeds a configurable limit, bounding peak memory.

**Architecture:** Add `MaxPackSize uint64` to inventory archive config structs (V1, V2) and their interfaces. Extract a `splitBlobChunks` helper that partitions sorted blobs by cumulative size. Wrap existing per-archive write logic in a loop over chunks in both V0 and V1 `Pack()`.

**Tech Stack:** Go, TOML config, existing `blob_store_configs` interface hierarchy, existing `stubBlobStore` test infrastructure.

---

### Task 1: Add `MaxPackSize` to config interface and V2 struct

**Files:**
- Modify: `go/internal/golf/blob_store_configs/main.go:69-80` (add to `ConfigInventoryArchiveDelta`)
- Modify: `go/internal/golf/blob_store_configs/toml_inventory_archive_v2.go:12-17` (add field)
- Modify: `go/internal/golf/blob_store_configs/toml_inventory_archive_v2.go` (add getter)

**Step 1: Add `GetMaxPackSize()` to `ConfigInventoryArchiveDelta` interface**

In `go/internal/golf/blob_store_configs/main.go`, add to the `ConfigInventoryArchiveDelta` interface:

```go
ConfigInventoryArchiveDelta interface {
    ConfigInventoryArchive
    DeltaConfigImmutable
    GetMaxPackSize() uint64
}
```

**Step 2: Add field to `TomlInventoryArchiveV2`**

In `go/internal/golf/blob_store_configs/toml_inventory_archive_v2.go`, add to the struct:

```go
type TomlInventoryArchiveV2 struct {
    HashTypeId      string                           `toml:"hash_type-id"`
    CompressionType compression_type.CompressionType `toml:"compression-type"`
    Encryption      markl.Id                         `toml:"encryption"`
    Delta           DeltaConfig                      `toml:"delta"`
    MaxPackSize     uint64                           `toml:"max-pack-size"`
}
```

**Step 3: Add getter method on V2**

```go
func (config TomlInventoryArchiveV2) GetMaxPackSize() uint64 {
    return config.MaxPackSize
}
```

**Step 4: Verify compilation**

Run: `just build` from `go/`

Expected: compile errors in V1 (missing `GetMaxPackSize`) -- that's expected, Task 2 fixes it.

### Task 2: Add `MaxPackSize` to V1 struct, getter, and upgrade paths

**Files:**
- Modify: `go/internal/golf/blob_store_configs/toml_inventory_archive_v1.go:23-29` (add field)
- Modify: `go/internal/golf/blob_store_configs/toml_inventory_archive_v1.go:114-126` (upgrade path)
- Modify: `go/internal/golf/blob_store_configs/toml_inventory_archive_v0.go:80-99` (upgrade path)

**Step 1: Add field and getter to V1**

In `go/internal/golf/blob_store_configs/toml_inventory_archive_v1.go`, add `MaxPackSize` field:

```go
type TomlInventoryArchiveV1 struct {
    HashTypeId       string                           `toml:"hash_type-id"`
    CompressionType  compression_type.CompressionType `toml:"compression-type"`
    LooseBlobStoreId blob_store_id.Id                 `toml:"loose-blob-store-id"`
    Encryption       markl.Id                         `toml:"encryption"`
    Delta            DeltaConfig                      `toml:"delta"`
    MaxPackSize      uint64                           `toml:"max-pack-size"`
}
```

Add getter:

```go
func (config TomlInventoryArchiveV1) GetMaxPackSize() uint64 {
    return config.MaxPackSize
}
```

**Step 2: Update V0 -> V1 upgrade to set default**

In `go/internal/golf/blob_store_configs/toml_inventory_archive_v0.go`, the `Upgrade()` method:

```go
upgraded := &TomlInventoryArchiveV1{
    HashTypeId:       config.HashTypeId,
    CompressionType:  config.CompressionType,
    LooseBlobStoreId: config.LooseBlobStoreId,
    Delta: DeltaConfig{
        Enabled:     false,
        Algorithm:   "bsdiff",
        MinBlobSize: 256,
        MaxBlobSize: 10485760,
        SizeRatio:   2.0,
    },
    MaxPackSize: 536870912, // 512 MiB
}
```

**Step 3: Update V1 -> V2 upgrade to carry field forward**

In `go/internal/golf/blob_store_configs/toml_inventory_archive_v1.go`, the `Upgrade()` method:

```go
upgraded := &TomlInventoryArchiveV2{
    HashTypeId:      config.HashTypeId,
    CompressionType: config.CompressionType,
    Delta:           config.Delta,
    MaxPackSize:     config.MaxPackSize,
}
```

**Step 4: Verify compilation**

Run: `just build` from `go/`

Expected: clean build, no errors.

**Step 5: Commit**

```
feat: add MaxPackSize config field to inventory archive V1/V2

Default 512 MiB. Carried through V0->V1->V2 upgrade paths.
```

### Task 3: Add `splitBlobChunks` helper with unit test

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v0.go` (add helper after `packedBlob` type)
- Create: `go/internal/india/blob_stores/pack_split_test.go`

**Step 1: Write failing test**

Create `go/internal/india/blob_stores/pack_split_test.go`:

```go
//go:build test && debug

package blob_stores

import (
    "testing"
)

func TestSplitBlobChunksUnlimited(t *testing.T) {
    blobs := []packedBlob{
        {hash: []byte{0x01}, data: make([]byte, 100)},
        {hash: []byte{0x02}, data: make([]byte, 200)},
        {hash: []byte{0x03}, data: make([]byte, 300)},
    }

    chunks := splitBlobChunks(blobs, 0)

    if len(chunks) != 1 {
        t.Fatalf("expected 1 chunk for unlimited, got %d", len(chunks))
    }

    if len(chunks[0]) != 3 {
        t.Fatalf("expected 3 blobs in chunk, got %d", len(chunks[0]))
    }
}

func TestSplitBlobChunksSplitsAtLimit(t *testing.T) {
    blobs := []packedBlob{
        {hash: []byte{0x01}, data: make([]byte, 100)},
        {hash: []byte{0x02}, data: make([]byte, 100)},
        {hash: []byte{0x03}, data: make([]byte, 100)},
        {hash: []byte{0x04}, data: make([]byte, 100)},
    }

    // Limit 250 means first chunk gets blobs 1+2 (200 bytes),
    // blob 3 would push to 300 so it starts a new chunk.
    chunks := splitBlobChunks(blobs, 250)

    if len(chunks) != 2 {
        t.Fatalf("expected 2 chunks, got %d", len(chunks))
    }

    if len(chunks[0]) != 2 {
        t.Fatalf("expected 2 blobs in first chunk, got %d", len(chunks[0]))
    }

    if len(chunks[1]) != 2 {
        t.Fatalf("expected 2 blobs in second chunk, got %d", len(chunks[1]))
    }
}

func TestSplitBlobChunksSingleBlobExceedsLimit(t *testing.T) {
    blobs := []packedBlob{
        {hash: []byte{0x01}, data: make([]byte, 500)},
        {hash: []byte{0x02}, data: make([]byte, 100)},
    }

    // Limit 200 but first blob is 500 -- it still gets its own chunk.
    chunks := splitBlobChunks(blobs, 200)

    if len(chunks) != 2 {
        t.Fatalf("expected 2 chunks, got %d", len(chunks))
    }

    if len(chunks[0]) != 1 {
        t.Fatalf("expected 1 blob in first chunk (oversized), got %d", len(chunks[0]))
    }

    if len(chunks[1]) != 1 {
        t.Fatalf("expected 1 blob in second chunk, got %d", len(chunks[1]))
    }
}

func TestSplitBlobChunksEmpty(t *testing.T) {
    chunks := splitBlobChunks(nil, 100)

    if len(chunks) != 0 {
        t.Fatalf("expected 0 chunks for empty input, got %d", len(chunks))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/india/blob_stores/ -run TestSplitBlobChunks`

Expected: FAIL with "undefined: splitBlobChunks"

**Step 3: Write implementation**

In `go/internal/india/blob_stores/pack_v0.go`, after the `packedBlob` struct (line 19):

```go
// splitBlobChunks partitions sorted blobs into chunks where each chunk's
// total data size does not exceed maxPackSize. A maxPackSize of 0 means
// unlimited (all blobs in one chunk). A single blob larger than the limit
// gets its own chunk.
func splitBlobChunks(blobs []packedBlob, maxPackSize uint64) [][]packedBlob {
    if len(blobs) == 0 {
        return nil
    }

    if maxPackSize == 0 {
        return [][]packedBlob{blobs}
    }

    var chunks [][]packedBlob
    var current []packedBlob
    var currentSize uint64

    for _, blob := range blobs {
        blobSize := uint64(len(blob.data))

        if len(current) > 0 && currentSize+blobSize > maxPackSize {
            chunks = append(chunks, current)
            current = nil
            currentSize = 0
        }

        current = append(current, blob)
        currentSize += blobSize
    }

    if len(current) > 0 {
        chunks = append(chunks, current)
    }

    return chunks
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test -v -tags test,debug ./src/india/blob_stores/ -run TestSplitBlobChunks`

Expected: all 4 tests PASS

**Step 5: Commit**

```
feat: add splitBlobChunks helper for pack size limiting
```

### Task 4: Integrate splitting into V1 `Pack()`

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v1.go:44-461`

**Step 1: Write failing test**

Add to `go/internal/india/blob_stores/pack_v1_test.go`:

```go
func TestPackV1SplitsWhenExceedingMaxPackSize(t *testing.T) {
    basePath := t.TempDir()
    cachePath := t.TempDir()

    hashFormat := markl.FormatHashSha256

    // Create 4 blobs of ~100 bytes each. Set MaxPackSize to 250 so we
    // get 2 pack files (2 blobs each).
    var blobIds []domain_interfaces.MarklId
    blobData := make(map[string][]byte)
    var allData [][]byte

    for i := range 4 {
        data := bytes.Repeat([]byte{byte('a' + i)}, 100)
        allData = append(allData, data)

        rawHash := sha256.Sum256(data)
        id, repool := hashFormat.GetBlobIdForHexString(
            hex.EncodeToString(rawHash[:]),
        )
        defer repool()

        blobIds = append(blobIds, id)
        blobData[id.String()] = data
    }

    stub := &stubBlobStore{
        allBlobIds: blobIds,
        blobData:   blobData,
    }

    config := blob_store_configs.TomlInventoryArchiveV1{
        HashTypeId:      markl.FormatIdHashSha256,
        CompressionType: compression_type.CompressionTypeNone,
        MaxPackSize:     250,
        Delta: blob_store_configs.DeltaConfig{
            Enabled:     false,
            Algorithm:   "bsdiff",
            MinBlobSize: 1,
            MaxBlobSize: 10485760,
            SizeRatio:   2.0,
        },
    }

    store := inventoryArchiveV1{
        defaultHash:    hashFormat,
        basePath:       basePath,
        cachePath:      cachePath,
        looseBlobStore: stub,
        index:          make(map[string]archiveEntryV1),
        config:         config,
    }

    if err := store.Pack(PackOptions{}); err != nil {
        t.Fatalf("Pack: %v", err)
    }

    // Verify all blobs are in the index.
    for i, id := range blobIds {
        if !store.HasBlob(id) {
            t.Fatalf("expected blob %d in index after pack", i)
        }
    }

    // Verify all blobs are readable with correct data.
    for i, id := range blobIds {
        reader, err := store.MakeBlobReader(id)
        if err != nil {
            t.Fatalf("MakeBlobReader for blob %d: %v", i, err)
        }

        got, err := io.ReadAll(reader)
        reader.Close()

        if err != nil {
            t.Fatalf("ReadAll for blob %d: %v", i, err)
        }

        if !bytes.Equal(got, allData[i]) {
            t.Errorf("blob %d data mismatch", i)
        }
    }

    // Verify multiple data files were created (split happened).
    dataMatches, err := filepath.Glob(
        filepath.Join(basePath, "*"+inventory_archive.DataFileExtensionV1),
    )
    if err != nil {
        t.Fatalf("globbing data files: %v", err)
    }

    if len(dataMatches) < 2 {
        t.Fatalf("expected at least 2 data files (split), got %d", len(dataMatches))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/india/blob_stores/ -run TestPackV1SplitsWhenExceedingMaxPackSize`

Expected: FAIL -- only 1 data file created (no splitting yet).

**Step 3: Refactor V1 `Pack()` to use chunk loop**

In `go/internal/india/blob_stores/pack_v1.go`, after the Phase 1 sort (line 100), replace the rest of the method body. Extract the archive-writing logic (Phases 2-7) into a `packChunk` method on `inventoryArchiveV1`, then loop:

```go
// After sort and empty check...
maxPackSize := store.config.GetMaxPackSize()
chunks := splitBlobChunks(blobs, maxPackSize)

for _, chunk := range chunks {
    if err = store.packChunk(chunk, options); err != nil {
        return err
    }
}

return nil
```

The `packChunk` method contains the existing Phase 2-7 logic, operating on its `chunk []packedBlob` parameter instead of the full `blobs` slice.

**Step 4: Run all pack V1 tests**

Run: `cd go && go test -v -tags test,debug ./src/india/blob_stores/ -run TestPackV1`

Expected: all PASS (existing + new split test)

**Step 5: Commit**

```
feat: V1 pack splits into multiple archives when exceeding MaxPackSize
```

### Task 5: Integrate splitting into V0 `Pack()`

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v0.go:21-267`
- Modify: `go/internal/india/blob_stores/main.go` or `store_inventory_archive.go` (add `GetMaxPackSize` to V0 config path)

**Step 1: Handle V0's config type**

V0's `store.config` is `ConfigInventoryArchive` (no `GetMaxPackSize()`). Add `GetMaxPackSize()` to the `ConfigInventoryArchive` interface in `main.go`:

```go
ConfigInventoryArchive interface {
    configLocal
    ConfigHashType
    domain_interfaces.BlobIOWrapper
    GetLooseBlobStoreId() blob_store_id.Id
    GetCompressionType() compression_type.CompressionType
    GetMaxPackSize() uint64
}
```

Add getter on `TomlInventoryArchiveV0`:

```go
func (config TomlInventoryArchiveV0) GetMaxPackSize() uint64 {
    return 536870912 // 512 MiB default for V0
}
```

**Step 2: Refactor V0 `Pack()` to use chunk loop**

Same pattern as V1: extract `packChunk` method, loop over `splitBlobChunks(blobs, store.config.GetMaxPackSize())`.

**Step 3: Run full test suite**

Run: `cd go && go test -v -tags test,debug ./src/india/blob_stores/`

Expected: all PASS

**Step 4: Commit**

```
feat: V0 pack splits into multiple archives when exceeding MaxPackSize
```

### Task 6: Update BATS integration test config helper

**Files:**
- Modify: `zz-tests_bats/pack.bats:18-44` (add `max-pack-size` to TOML config)

**Step 1: Update config helper**

In `zz-tests_bats/pack.bats`, update `create_inventory_archive_v1_config` to include the new field:

```bash
cat >>"${config_dir}/dodder-blob_store-config" <<-EOM

    hash_type-id = 'blake2b256'
    compression-type = 'zstd'
    loose-blob-store-id = '.default'
    encryption = ''
    max-pack-size = 0

    [delta]
    enabled = ${delta_enabled}
    algorithm = 'bsdiff'
    min-blob-size = 0
    max-blob-size = 0
    size-ratio = 0.0
EOM
```

Setting `max-pack-size = 0` (unlimited) preserves existing test behavior.

**Step 2: Run BATS integration tests**

Run: `just test-bats-targets pack.bats` from repo root

Expected: existing tests PASS

**Step 3: Commit**

```
test: add max-pack-size field to BATS pack test config helper
```

### Task 7: Update fixtures and run full test suite

**Step 1: Rebuild**

Run: `just build` from repo root

**Step 2: Regenerate fixtures if needed**

Run: `just test-bats-update-fixtures` from repo root

Review: `git diff -- zz-tests_bats/migration/`

**Step 3: Run full test suite**

Run: `just test` from repo root

Expected: all PASS

**Step 4: Commit (if fixtures changed)**

```
test: regenerate fixtures for max-pack-size config field
```
