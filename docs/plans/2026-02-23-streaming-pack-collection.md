# Streaming Pack Collection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate all-blobs-in-RAM peak memory by collecting only metadata first, then loading blob data one chunk at a time.

**Architecture:** Two-phase pack: Phase 1 iterates loose blobs to collect digest + uncompressed size without retaining data (via `GetBlobSize`). `splitBlobChunks` partitions this metadata. Phase 2 re-reads blob data per chunk. Also renames `hash` to `digest` throughout pack types for correctness (hashes are functions, digests are outputs).

**Tech Stack:** Go, dodder blob store layer (`india/blob_stores`)

---

### Task 1: Rename `hash` to `digest` in `packedBlob`

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v0.go:18-21` (struct definition)
- Modify: `go/internal/india/blob_stores/pack_v0.go` (all `blob.hash` → `blob.digest`, `blobs[i].hash` → `blobs[i].digest`)
- Modify: `go/internal/india/blob_stores/pack_v1.go` (all `blob.hash` → `blob.digest`, `targetBlob.hash` → `targetBlob.digest`, `baseBlob.hash` → `baseBlob.digest`)
- Modify: `go/internal/india/blob_stores/pack_split_test.go` (all `hash:` → `digest:`)

**Step 1: Rename the struct field**

In `pack_v0.go`, change:
```go
type packedBlob struct {
	digest []byte
	data   []byte
}
```

**Step 2: Update all references in `pack_v0.go`**

Every `blob.hash`, `blobs[i].hash`, `.hash` on packedBlob → `.digest`. Affected lines:
- `splitBlobChunks`: `blobs[i].hash` → `blobs[i].digest` in sort comparator (line ~125)
- Collection loop: `packedBlob{hash: hashBytes, data: data}` → `packedBlob{digest: hashBytes, data: data}` (line ~115)
- `packChunkArchive`: `blob.hash` → `blob.digest` in WriteEntry (line ~263)
- `validateArchive`: no packedBlob field access (compares entry hashes)
- `deleteLooseBlobs`: `blob.hash` → `blob.digest` (line ~490)
- `DeletionPrecondition` loop: `blob.hash` → `blob.digest` (line ~196)

**Step 3: Update all references in `pack_v1.go`**

- Collection loop: `packedBlob{hash: hashBytes, data: data}` → `packedBlob{digest: hashBytes, data: data}` (line ~105)
- Sort comparator: `blobs[i].hash` → `blobs[i].digest` (line ~115)
- `packChunkArchiveV1`: `blob.hash` → `blob.digest` in BlobMetadata (line ~257), `blob.hash` → `blob.digest` in WriteFullEntry calls, `targetBlob.hash` → `targetBlob.digest`, `baseBlob.hash` → `baseBlob.digest` in delta pass
- `deleteLooseBlobsV1`: `blob.hash` → `blob.digest` (line ~726)
- `DeletionPrecondition` loop: `blob.hash` → `blob.digest` (line ~189)

**Step 4: Update test file**

In `pack_split_test.go`, change all `hash:` field initializers to `digest:`:
```go
// Before
{hash: []byte{0x01}, data: make([]byte, 100)},
// After
{digest: []byte{0x01}, data: make([]byte, 100)},
```

**Step 5: Build to verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/go && go build -tags debug ./...`
Expected: BUILD SUCCESS

**Step 6: Run unit tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./src/india/blob_stores/`
Expected: All tests pass

**Step 7: Commit**

```bash
git add go/internal/india/blob_stores/pack_v0.go go/internal/india/blob_stores/pack_v1.go go/internal/india/blob_stores/pack_split_test.go
git commit -m "refactor: rename packedBlob.hash to packedBlob.digest"
```

---

### Task 2: Add `packedBlobMeta` struct and refactor `splitBlobChunks`

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v0.go` (add struct, change `splitBlobChunks` signature)
- Modify: `go/internal/india/blob_stores/pack_split_test.go` (update tests for new type)

**Step 1: Add `packedBlobMeta` struct**

In `pack_v0.go`, below `packedBlob`, add:
```go
type packedBlobMeta struct {
	digest []byte
	size   uint64
}
```

**Step 2: Change `splitBlobChunks` to accept `[]packedBlobMeta`**

```go
func splitBlobChunks(metas []packedBlobMeta, maxPackSize uint64) [][]packedBlobMeta {
	if len(metas) == 0 {
		return nil
	}

	if maxPackSize == 0 {
		return [][]packedBlobMeta{metas}
	}

	var chunks [][]packedBlobMeta
	var current []packedBlobMeta
	var currentSize uint64

	for _, meta := range metas {
		if len(current) > 0 && currentSize+meta.size > maxPackSize {
			chunks = append(chunks, current)
			current = nil
			currentSize = 0
		}

		current = append(current, meta)
		currentSize += meta.size
	}

	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}
```

**Step 3: Update `pack_split_test.go`**

Change all tests to use `packedBlobMeta` instead of `packedBlob`:
```go
func TestSplitBlobChunksUnlimited(t *testing.T) {
	metas := []packedBlobMeta{
		{digest: []byte{0x01}, size: 100},
		{digest: []byte{0x02}, size: 200},
		{digest: []byte{0x03}, size: 300},
	}

	chunks := splitBlobChunks(metas, 0)
	// ... rest unchanged
}

func TestSplitBlobChunksSplitsAtLimit(t *testing.T) {
	metas := []packedBlobMeta{
		{digest: []byte{0x01}, size: 100},
		{digest: []byte{0x02}, size: 100},
		{digest: []byte{0x03}, size: 100},
		{digest: []byte{0x04}, size: 100},
	}

	chunks := splitBlobChunks(metas, 250)
	// ... rest unchanged
}

func TestSplitBlobChunksSingleBlobExceedsLimit(t *testing.T) {
	metas := []packedBlobMeta{
		{digest: []byte{0x01}, size: 500},
		{digest: []byte{0x02}, size: 100},
	}

	chunks := splitBlobChunks(metas, 200)
	// ... rest unchanged
}

func TestSplitBlobChunksEmpty(t *testing.T) {
	chunks := splitBlobChunks(nil, 100)
	// ... rest unchanged
}
```

**Step 4: Build to verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/go && go build -tags debug ./...`
Expected: BUILD FAILURE — `Pack()` in both V0 and V1 still calls `splitBlobChunks` with `[]packedBlob`. This is expected; we fix the callers in Task 3.

Note: The build failure confirms the type change is correct. The split test file has build tag `test && debug` so it won't cause compilation failure in a normal build. Verify the test file compiles:

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug -run TestSplitBlobChunks ./src/india/blob_stores/ -count=1`
Expected: This will fail to compile because `Pack()` references the old signature. We proceed to Task 3 to fix callers.

**Step 5: Commit (partial — test-only change)**

Do NOT commit yet. Proceed to Task 3 which makes the callers compile.

---

### Task 3: Add `GetBlobSize` and refactor V0 `Pack()` to two-phase collection

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v0.go` (add `GetBlobSize`, rewrite `Pack()` collection + chunking)

**Step 1: Add `GetBlobSize` method on `inventoryArchiveV0`**

Below `deleteLooseBlobs` in `pack_v0.go`, add:

```go
func (store inventoryArchiveV0) GetBlobSize(
	id domain_interfaces.MarklId,
) (size uint64, err error) {
	reader, err := store.looseBlobStore.MakeBlobReader(id)
	if err != nil {
		err = errors.Wrapf(err, "opening blob %s for size", id)
		return size, err
	}

	defer errors.DeferredCloser(&err, reader)

	n, err := io.Copy(io.Discard, reader)
	if err != nil {
		err = errors.Wrapf(err, "reading blob %s for size", id)
		return size, err
	}

	return uint64(n), nil
}
```

**Step 2: Rewrite the V0 `Pack()` collection phase**

Replace the collection loop and chunking in `Pack()`. The new structure:

**Phase 1: Collect metadata only**
```go
var metas []packedBlobMeta

for looseId, iterErr := range store.looseBlobStore.AllBlobs() {
	if err = packContextCancelled(ctx); err != nil {
		err = errors.Wrap(err)
		tapNotOk(tw, "collect loose blobs", err)
		return err
	}

	if iterErr != nil {
		err = errors.Wrap(iterErr)
		tapNotOk(tw, "collect loose blobs", err)
		return err
	}

	if looseId.IsNull() {
		continue
	}

	if _, inArchive := store.index[looseId.String()]; inArchive {
		continue
	}

	if options.BlobFilter != nil {
		if _, inFilter := options.BlobFilter[looseId.String()]; !inFilter {
			continue
		}
	}

	blobSize, sizeErr := store.GetBlobSize(looseId)
	if sizeErr != nil {
		err = errors.Wrapf(sizeErr, "getting size of loose blob %s", looseId)
		tapNotOk(tw, "collect loose blobs", err)
		return err
	}

	digestBytes := make([]byte, len(looseId.GetBytes()))
	copy(digestBytes, looseId.GetBytes())

	metas = append(metas, packedBlobMeta{digest: digestBytes, size: blobSize})
}

if len(metas) == 0 {
	return nil
}

tapOk(tw, fmt.Sprintf("collect %d loose blobs", len(metas)))

sort.Slice(metas, func(i, j int) bool {
	return bytes.Compare(metas[i].digest, metas[j].digest) < 0
})
```

**Phase 1b: Split into chunks using metadata**
```go
maxPackSize := options.MaxPackSize
if maxPackSize == 0 {
	maxPackSize = store.config.GetMaxPackSize()
}

chunks := splitBlobChunks(metas, maxPackSize)
totalChunks := len(chunks)
```

**Phase 2: Load data per chunk and write archive**

Replace the old chunk-writing loop with a new one that loads data per chunk:

```go
type chunkResult struct {
	dataPath string
	metas    []packedBlobMeta
}

var results []chunkResult

for chunkIdx, chunkMetas := range chunks {
	if err = packContextCancelled(ctx); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Load blob data for this chunk only.
	blobs := make([]packedBlob, len(chunkMetas))

	for i, meta := range chunkMetas {
		marklId, repool := store.defaultHash.GetBlobIdForHexString(
			hex.EncodeToString(meta.digest),
		)

		reader, readErr := store.looseBlobStore.MakeBlobReader(marklId)
		repool()

		if readErr != nil {
			err = errors.Wrapf(readErr, "reading loose blob %x", meta.digest)
			tapNotOk(tw, fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks), err)
			return err
		}

		data, readAllErr := io.ReadAll(reader)
		reader.Close()

		if readAllErr != nil {
			err = errors.Wrapf(readAllErr, "reading loose blob data %x", meta.digest)
			tapNotOk(tw, fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks), err)
			return err
		}

		blobs[i] = packedBlob{digest: meta.digest, data: data}
	}

	dataPath, entryCount, packErr := store.packChunkArchive(blobs)
	if packErr != nil {
		desc := fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks)
		tapNotOk(tw, desc, packErr)
		return packErr
	}

	tapOk(tw, fmt.Sprintf(
		"write chunk %d/%d (%d entries, 0 delta)",
		chunkIdx+1, totalChunks, entryCount,
	))

	// Release blob data — let GC reclaim before next chunk.
	blobs = nil

	results = append(results, chunkResult{dataPath: dataPath, metas: chunkMetas})
}
```

**Step 3: Update `writeCache`, `validateArchive`, `DeletionPrecondition`, and `deleteLooseBlobs` calls**

These functions currently iterate `[]packedBlob` for digests only (no data access). Update the post-packing phases to iterate `metas` (the flat list) and `chunkResult.metas`:

- `validateArchive` signature remains `(dataPath string, blobs []packedBlob)` — but we no longer have blobs at validation time. Change its signature to accept `entryCount int` instead:

  Actually, `validateArchive` re-reads the archive from disk and re-hashes each entry against `entry.Hash` from the archive itself. It doesn't compare against the in-memory `blobs[i].data`. The only thing it uses `blobs` for is `len(blobs)` to check entry count. So change the signature to take `expectedCount int`:

  ```go
  func (store inventoryArchiveV0) validateArchive(
  	dataPath string,
  	expectedCount int,
  ) (err error) {
  ```

  And change `len(blobs)` → `expectedCount` in the mismatch check.

- Validation loop in `Pack()`:
  ```go
  for chunkIdx, r := range results {
  	// ...
  	if err = store.validateArchive(r.dataPath, len(r.metas)); err != nil {
  ```

- `DeletionPrecondition` loop: iterate `metas` instead of `blobs`:
  ```go
  blobSeq := func(
  	yield func(domain_interfaces.MarklId, error) bool,
  ) {
  	for _, meta := range metas {
  		marklId, repool := store.defaultHash.GetBlobIdForHexString(
  			hex.EncodeToString(meta.digest),
  		)

  		if !yield(marklId, nil) {
  			repool()
  			return
  		}

  		repool()
  	}
  }
  ```

- `deleteLooseBlobs`: change to accept `[]packedBlobMeta`:
  ```go
  func (store inventoryArchiveV0) deleteLooseBlobs(
  	ctx interfaces.ActiveContext,
  	metas []packedBlobMeta,
  ) (err error) {
  	deleter, ok := store.looseBlobStore.(BlobDeleter)
  	if !ok {
  		err = errors.Errorf("loose blob store does not support deletion")
  		return err
  	}

  	for _, meta := range metas {
  		if err = packContextCancelled(ctx); err != nil {
  			err = errors.Wrap(err)
  			return err
  		}

  		marklId, repool := store.defaultHash.GetBlobIdForHexString(
  			hex.EncodeToString(meta.digest),
  		)

  		if deleteErr := deleter.DeleteBlob(marklId); deleteErr != nil {
  			repool()
  			err = errors.Wrap(deleteErr)
  			return err
  		}

  		repool()
  	}

  	return nil
  }
  ```

  Call site: `store.deleteLooseBlobs(ctx, metas)` with the `metas` flat list.
  TAP line: `fmt.Sprintf("delete %d loose blobs", len(metas))`

**Step 4: Remove the TODO(P2) comment**

Delete the `TODO(P2)` comment from V0 `Pack()` since this task implements it.

**Step 5: Build to verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/go && go build -tags debug ./...`
Expected: BUILD FAILURE — V1 `Pack()` still calls `splitBlobChunks` with old type. Expected; fix in Task 4.

**Step 6: Run V0 tests only**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug -run 'TestPack$|TestPackSkips|TestPackDelete|TestSplitBlobChunks' ./src/india/blob_stores/ -count=1`
Expected: This may fail to compile because V1 Pack() is broken. If so, defer test verification to after Task 4.

**Step 7: Commit (hold until Task 4 completes for green build)**

---

### Task 4: Refactor V1 `Pack()` to two-phase collection

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v1.go` (rewrite collection + chunking, update helpers)

**Step 1: Add `GetBlobSize` method on `inventoryArchiveV1`**

In `pack_v1.go`, below `deleteLooseBlobsV1`:

```go
func (store inventoryArchiveV1) GetBlobSize(
	id domain_interfaces.MarklId,
) (size uint64, err error) {
	reader, err := store.looseBlobStore.MakeBlobReader(id)
	if err != nil {
		err = errors.Wrapf(err, "opening blob %s for size", id)
		return size, err
	}

	defer errors.DeferredCloser(&err, reader)

	n, err := io.Copy(io.Discard, reader)
	if err != nil {
		err = errors.Wrapf(err, "reading blob %s for size", id)
		return size, err
	}

	return uint64(n), nil
}
```

**Step 2: Rewrite the V1 `Pack()` collection phase**

Apply the same two-phase pattern as V0. Phase 1 collects `[]packedBlobMeta`, sorts, splits. Phase 2 loads data per chunk. The chunk-writing loop body loads blob data then calls `packChunkArchiveV1`:

```go
for chunkIdx, chunkMetas := range chunks {
	if err = packContextCancelled(ctx); err != nil {
		err = errors.Wrap(err)
		return err
	}

	blobs := make([]packedBlob, len(chunkMetas))

	for i, meta := range chunkMetas {
		marklId, repool := store.defaultHash.GetBlobIdForHexString(
			hex.EncodeToString(meta.digest),
		)

		reader, readErr := store.looseBlobStore.MakeBlobReader(marklId)
		repool()

		if readErr != nil {
			err = errors.Wrapf(readErr, "reading loose blob %x", meta.digest)
			tapNotOk(tw, fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks), err)
			return err
		}

		data, readAllErr := io.ReadAll(reader)
		reader.Close()

		if readAllErr != nil {
			err = errors.Wrapf(readAllErr, "reading loose blob data %x", meta.digest)
			tapNotOk(tw, fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks), err)
			return err
		}

		blobs[i] = packedBlob{digest: meta.digest, data: data}
	}

	dataPath, fullCount, deltaCount, packErr := store.packChunkArchiveV1(ctx, blobs)
	if packErr != nil {
		desc := fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks)
		tapNotOk(tw, desc, packErr)
		return packErr
	}

	tapOk(tw, fmt.Sprintf(
		"write chunk %d/%d (%d entries, %d delta)",
		chunkIdx+1, totalChunks, fullCount+deltaCount, deltaCount,
	))

	blobs = nil

	results = append(results, chunkResult{dataPath: dataPath, metas: chunkMetas})
}
```

**Step 3: Update `validateArchiveV1` signature**

Same as V0: change to accept `expectedCount int` instead of `[]packedBlob`:

```go
func (store inventoryArchiveV1) validateArchiveV1(
	dataPath string,
	expectedCount int,
) (err error) {
```

Change `len(blobs)` → `expectedCount`. The rest of the validation logic (re-reading, re-hashing, delta reconstruction) stays the same since it reads from disk.

**Step 4: Update `deleteLooseBlobsV1` to accept `[]packedBlobMeta`**

```go
func (store inventoryArchiveV1) deleteLooseBlobsV1(
	ctx interfaces.ActiveContext,
	metas []packedBlobMeta,
) (err error) {
```

Update body: `for _, meta := range metas`, `meta.digest` instead of `blob.hash`.

**Step 5: Update DeletionPrecondition and delete calls in V1 Pack()**

Same pattern as V0: iterate `metas` for precondition check and deletion.

**Step 6: Remove the TODO(P2) comment from V1 Pack()**

**Step 7: Build to verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/go && go build -tags debug ./...`
Expected: BUILD SUCCESS

**Step 8: Run all pack-related unit tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./src/india/blob_stores/ -count=1`
Expected: All tests pass

**Step 9: Commit Tasks 2-4 together**

```bash
git add go/internal/india/blob_stores/pack_v0.go go/internal/india/blob_stores/pack_v1.go go/internal/india/blob_stores/pack_split_test.go
git commit -m "feat: stream pack collection to bound memory to one chunk at a time

Collect only digest + size metadata in the first pass, split into
chunks, then re-read blob data per chunk. Peak memory is now bounded
by MaxPackSize instead of total loose blob size.

- Add packedBlobMeta struct (digest + size)
- Add GetBlobSize method on inventoryArchiveV0 and inventoryArchiveV1
- Refactor splitBlobChunks to operate on []packedBlobMeta
- Refactor Pack() in both V0 and V1 for two-phase collection
- Update validateArchive/deleteLooseBlobs to use metadata-only types"
```

---

### Task 5: Run full test suite

**Step 1: Run Go unit tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./... 2>&1 | tail -50`
Expected: All packages pass

**Step 2: Run format check**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just codemod-go-fmt`
Expected: No changes (or apply and commit if formatting changed)

**Step 3: Build binaries**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just build`
Expected: BUILD SUCCESS

**Step 4: Run BATS integration tests**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just test-bats`
Expected: All BATS tests pass

**Step 5: If BATS fixture diff, regenerate and commit**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just test-bats-update-fixtures`
Review diff. If fixtures changed, commit them.
