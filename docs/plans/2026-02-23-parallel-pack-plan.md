# Parallel Packfile Generation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Parallelize blob size collection (V0 + V1) and delta computation (V1) in the pack pipeline using goroutine pools with indexed result slices.

**Architecture:** Extract a shared `collectBlobMetasParallel` function that fans out `GetBlobSize` calls across `runtime.NumCPU()` workers. Extract delta computation into a parallel pre-computation step using `sync.WaitGroup` + `[]deltaResult` indexed by assignment position. Both use a buffered channel semaphore for concurrency limiting and `sync.Once` for first-error capture.

**Tech Stack:** Go, `sync.WaitGroup`, `runtime.NumCPU()`, buffered channel semaphore, existing `india/blob_stores` package

---

### Task 1: Extract `collectBlobMetasParallel` with serial fallback

**Files:**
- Create: `go/internal/india/blob_stores/pack_parallel.go`
- Modify: `go/internal/india/blob_stores/pack_v0.go:63-114` (replace Phase 1 collection loop)
- Modify: `go/internal/india/blob_stores/pack_v1.go:46-97` (replace Phase 1 collection loop)

**Step 1: Create `pack_parallel.go` with the shared collection function**

```go
package blob_stores

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

// blobSizeFn returns the uncompressed size of a blob given its ID.
// Implementations may read through the full decompression pipeline or use
// a fast path like os.Stat when available.
type blobSizeFn func(domain_interfaces.MarklId) (uint64, error)

// collectBlobMetasParallel iterates the loose blob store to find packing
// candidates, then fans out size lookups across multiple goroutines.
//
// The AllBlobs iterator is consumed serially (it is not concurrent-safe).
// Size lookups are parallel with min(NumCPU, len(candidates)) workers.
func collectBlobMetasParallel(
	ctx interfaces.ActiveContext,
	tw *tap.Writer,
	looseBlobStore domain_interfaces.BlobStore,
	defaultHash markl.FormatHash,
	index map[string]bool,
	options PackOptions,
	sizeFn blobSizeFn,
) (metas []packedBlobMeta, err error) {
	// Phase 1a: Serial iteration to collect candidate IDs.
	type candidate struct {
		id    domain_interfaces.MarklId
		digest []byte
	}

	var candidates []candidate

	for looseId, iterErr := range looseBlobStore.AllBlobs() {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			tapNotOk(tw, "collect loose blobs", err)
			return nil, err
		}

		if iterErr != nil {
			err = errors.Wrap(iterErr)
			tapNotOk(tw, "collect loose blobs", err)
			return nil, err
		}

		if looseId.IsNull() {
			continue
		}

		if index[looseId.String()] {
			continue
		}

		if options.BlobFilter != nil {
			if _, inFilter := options.BlobFilter[looseId.String()]; !inFilter {
				continue
			}
		}

		digestBytes := make([]byte, len(looseId.GetBytes()))
		copy(digestBytes, looseId.GetBytes())

		candidates = append(candidates, candidate{
			id:    looseId,
			digest: digestBytes,
		})
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Phase 1b: Parallel size lookups.
	metas = make([]packedBlobMeta, len(candidates))

	numWorkers := runtime.NumCPU()
	if numWorkers > len(candidates) {
		numWorkers = len(candidates)
	}

	sem := make(chan struct{}, numWorkers)

	var (
		wg       sync.WaitGroup
		firstErr error
		errOnce  sync.Once
		cancel   context.CancelFunc
	)

	sizeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i, c := range candidates {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, cand candidate) {
			defer wg.Done()
			defer func() { <-sem }()

			select {
			case <-sizeCtx.Done():
				return
			default:
			}

			blobSize, sizeErr := sizeFn(cand.id)
			if sizeErr != nil {
				if options.SkipMissingBlobs {
					tapComment(tw, fmt.Sprintf("blob skipped: %s", cand.id))
					// Mark as zero-size; filtered out below.
					metas[idx] = packedBlobMeta{digest: nil, size: 0}
					return
				}

				errOnce.Do(func() {
					firstErr = errors.Wrapf(
						sizeErr,
						"getting size of loose blob %s",
						cand.id,
					)
					cancel()
				})

				return
			}

			metas[idx] = packedBlobMeta{
				digest: cand.digest,
				size:   blobSize,
			}
		}(i, c)
	}

	wg.Wait()

	if firstErr != nil {
		tapNotOk(tw, "collect loose blobs", firstErr)
		return nil, firstErr
	}

	// Filter out skipped blobs (nil digest from SkipMissingBlobs).
	filtered := metas[:0]
	for _, m := range metas {
		if m.digest != nil {
			filtered = append(filtered, m)
		}
	}

	metas = filtered

	if len(metas) == 0 {
		return nil, nil
	}

	tapOk(tw, fmt.Sprintf("collect %d loose blobs", len(metas)))

	sort.Slice(metas, func(i, j int) bool {
		return bytes.Compare(metas[i].digest, metas[j].digest) < 0
	})

	return metas, nil
}
```

**Step 2: Create helper to build the index-presence map**

In the same file, add two helpers that V0 and V1 call to build the `map[string]bool` from their typed index maps:

```go
func indexPresenceFromV0(index map[string]archiveEntry) map[string]bool {
	m := make(map[string]bool, len(index))
	for k := range index {
		m[k] = true
	}
	return m
}

func indexPresenceFromV1(index map[string]archiveEntryV1) map[string]bool {
	m := make(map[string]bool, len(index))
	for k := range index {
		m[k] = true
	}
	return m
}
```

**Step 3: Update V0 `Pack()` to call `collectBlobMetasParallel`**

In `pack_v0.go`, replace lines 67-124 (the serial Phase 1 collection + sort) with:

```go
	metas, err := collectBlobMetasParallel(
		ctx,
		tw,
		store.looseBlobStore,
		store.defaultHash,
		indexPresenceFromV0(store.index),
		options,
		store.GetBlobSize,
	)
	if err != nil {
		return err
	}

	if len(metas) == 0 {
		return nil
	}
```

Remove the `tapOk(tw, ...)` and `sort.Slice(...)` lines that were below the old collection loop — they are now inside `collectBlobMetasParallel`.

**Step 4: Update V1 `Pack()` to call `collectBlobMetasParallel`**

In `pack_v1.go`, replace lines 52-107 (the serial Phase 1 collection + sort) with:

```go
	metas, err := collectBlobMetasParallel(
		ctx,
		tw,
		store.looseBlobStore,
		store.defaultHash,
		indexPresenceFromV1(store.index),
		options,
		store.GetBlobSize,
	)
	if err != nil {
		return err
	}

	if len(metas) == 0 {
		return nil
	}
```

Remove the `tapOk(tw, ...)` and `sort.Slice(...)` lines that were below the old collection loop.

**Step 5: Build to verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go build -tags debug ./...`
Expected: BUILD SUCCESS

**Step 6: Run unit tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go test -v -tags test,debug ./src/india/blob_stores/ -count=1`
Expected: All tests pass (TestPack, TestPackV1WithDelta, TestPackV1WithoutDelta, TestPackDeleteLoose, TestPackSkipsAlreadyArchivedBlobs, etc.)

**Step 7: Format**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads && just codemod-go-fmt`

**Step 8: Commit**

```
feat: parallelize blob size collection during pack

Extract collectBlobMetasParallel into pack_parallel.go. The AllBlobs
iterator runs serially (not concurrent-safe), then GetBlobSize calls
fan out across min(NumCPU, candidateCount) goroutines using a
buffered channel semaphore. First error cancels remaining workers.
Both V0 and V1 Pack() now use the shared function.
```

---

### Task 2: Add unit test for parallel size collection

**Files:**
- Create: `go/internal/india/blob_stores/pack_parallel_test.go`

**Step 1: Write the test file**

```go
//go:build test && debug

package blob_stores

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

func TestCollectBlobMetasParallelBasic(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	testData := map[string][]byte{
		"blob1": []byte("parallel test blob one"),
		"blob2": []byte("parallel test blob two"),
		"blob3": []byte("parallel test blob three"),
	}

	var allIds []domain_interfaces.MarklId
	blobData := make(map[string][]byte)

	for _, data := range testData {
		rawHash := sha256.Sum256(data)
		id, repool := hashFormat.GetBlobIdForHexString(
			hex.EncodeToString(rawHash[:]),
		)
		defer repool()

		allIds = append(allIds, id)
		blobData[id.String()] = data
	}

	stub := &stubBlobStore{
		allBlobIds: allIds,
		blobData:   blobData,
	}

	sizeFn := func(id domain_interfaces.MarklId) (uint64, error) {
		if data, ok := blobData[id.String()]; ok {
			return uint64(len(data)), nil
		}
		return 0, nil
	}

	metas, err := collectBlobMetasParallel(
		nil,
		nil,
		stub,
		hashFormat,
		make(map[string]bool),
		PackOptions{},
		sizeFn,
	)
	if err != nil {
		t.Fatalf("collectBlobMetasParallel: %v", err)
	}

	if len(metas) != 3 {
		t.Fatalf("expected 3 metas, got %d", len(metas))
	}

	// Verify sorted by digest.
	for i := 1; i < len(metas); i++ {
		if bytes.Compare(metas[i-1].digest, metas[i].digest) >= 0 {
			t.Errorf("metas not sorted at index %d", i)
		}
	}

	// Verify sizes are non-zero.
	for i, m := range metas {
		if m.size == 0 {
			t.Errorf("meta %d has zero size", i)
		}
	}
}

func TestCollectBlobMetasParallelSkipsArchived(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	testData1 := []byte("archived blob data")
	testData2 := []byte("loose only blob data")

	rawHash1 := sha256.Sum256(testData1)
	rawHash2 := sha256.Sum256(testData2)

	id1, repool1 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash1[:]),
	)
	defer repool1()

	id2, repool2 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash2[:]),
	)
	defer repool2()

	stub := &stubBlobStore{
		allBlobIds: []domain_interfaces.MarklId{id1, id2},
		blobData: map[string][]byte{
			id1.String(): testData1,
			id2.String(): testData2,
		},
	}

	// Mark id1 as already archived.
	indexPresence := map[string]bool{
		id1.String(): true,
	}

	sizeFn := func(id domain_interfaces.MarklId) (uint64, error) {
		if data, ok := stub.blobData[id.String()]; ok {
			return uint64(len(data)), nil
		}
		return 0, nil
	}

	metas, err := collectBlobMetasParallel(
		nil,
		nil,
		stub,
		hashFormat,
		indexPresence,
		PackOptions{},
		sizeFn,
	)
	if err != nil {
		t.Fatalf("collectBlobMetasParallel: %v", err)
	}

	if len(metas) != 1 {
		t.Fatalf("expected 1 meta (archived blob skipped), got %d", len(metas))
	}
}

func TestCollectBlobMetasParallelEmpty(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	stub := &stubBlobStore{}

	metas, err := collectBlobMetasParallel(
		nil,
		nil,
		stub,
		hashFormat,
		make(map[string]bool),
		PackOptions{},
		func(domain_interfaces.MarklId) (uint64, error) { return 0, nil },
	)
	if err != nil {
		t.Fatalf("collectBlobMetasParallel: %v", err)
	}

	if metas != nil {
		t.Fatalf("expected nil metas for empty store, got %d", len(metas))
	}
}
```

**Step 2: Run the new test**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go test -v -tags test,debug -run TestCollectBlobMetasParallel ./src/india/blob_stores/ -count=1`
Expected: All 3 tests pass

**Step 3: Run full pack test suite to verify no regressions**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go test -v -tags test,debug ./src/india/blob_stores/ -count=1`
Expected: All tests pass

**Step 4: Format**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads && just codemod-go-fmt`

**Step 5: Commit**

```
test: add unit tests for parallel blob size collection
```

---

### Task 3: Parallelize delta computation in `packChunkArchiveV1`

**Files:**
- Modify: `go/internal/india/blob_stores/pack_v1.go:257-447` (`packChunkArchiveV1` method)

**Step 1: Add `deltaResult` type to `pack_parallel.go`**

At the bottom of `pack_parallel.go`, add:

```go
// deltaResult holds the output of a parallel delta computation.
// When deltaData is nil, the blob should be stored as a full entry
// (either because delta computation failed or the delta was larger
// than the original).
type deltaResult struct {
	blobIdx   int
	baseIdx   int
	deltaData []byte
}
```

**Step 2: Refactor the delta pass in `packChunkArchiveV1`**

Replace the serial delta loop (lines 381-447 in `pack_v1.go`, the "Second pass: write delta entries" section) with a parallel computation followed by a sequential write:

```go
	// Second pass: compute deltas in parallel, then write sequentially.
	type indexedAssignment struct {
		resultIdx int
		blobIdx   int
		baseIdx   int
	}

	var orderedAssignments []indexedAssignment
	for blobIdx, baseIdx := range assignments {
		orderedAssignments = append(orderedAssignments, indexedAssignment{
			resultIdx: len(orderedAssignments),
			blobIdx:   blobIdx,
			baseIdx:   baseIdx,
		})
	}

	results := make([]deltaResult, len(orderedAssignments))

	numWorkers := runtime.NumCPU()
	if numWorkers > len(orderedAssignments) {
		numWorkers = len(orderedAssignments)
	}

	if len(orderedAssignments) > 0 {
		sem := make(chan struct{}, numWorkers)

		var (
			wg       sync.WaitGroup
			firstErr error
			errOnce  sync.Once
			cancel   context.CancelFunc
		)

		deltaCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for _, assignment := range orderedAssignments {
			wg.Add(1)
			sem <- struct{}{}

			go func(a indexedAssignment) {
				defer wg.Done()
				defer func() { <-sem }()

				select {
				case <-deltaCtx.Done():
					return
				default:
				}

				targetBlob := blobs[a.blobIdx]
				baseBlob := blobs[a.baseIdx]

				baseHash, _ := store.defaultHash.Get() //repool:owned
				baseReader := markl_io.MakeReadCloser(
					baseHash,
					bytes.NewReader(baseBlob.data),
				)

				var deltaBuf bytes.Buffer

				computeErr := alg.Compute(
					baseReader,
					int64(len(baseBlob.data)),
					bytes.NewReader(targetBlob.data),
					&deltaBuf,
				)
				if computeErr != nil {
					// Delta computation failed — store as full.
					results[a.resultIdx] = deltaResult{
						blobIdx: a.blobIdx,
						baseIdx: a.baseIdx,
					}
					return
				}

				rawDelta := deltaBuf.Bytes()

				// Trial-and-discard: if delta is not smaller, store as full.
				if len(rawDelta) >= len(targetBlob.data) {
					results[a.resultIdx] = deltaResult{
						blobIdx: a.blobIdx,
						baseIdx: a.baseIdx,
					}
					return
				}

				results[a.resultIdx] = deltaResult{
					blobIdx:   a.blobIdx,
					baseIdx:   a.baseIdx,
					deltaData: rawDelta,
				}
			}(assignment)
		}

		wg.Wait()

		if firstErr != nil {
			err = firstErr
			return dataPath, 0, 0, err
		}
	}

	// Sequential write pass: write deltas (or full fallbacks) in order.
	for _, dr := range results {
		if err = packContextCancelled(ctx); err != nil {
			tmpFile.Close()
			err = errors.Wrap(err)
			return dataPath, 0, 0, err
		}

		targetBlob := blobs[dr.blobIdx]
		baseBlob := blobs[dr.baseIdx]

		if dr.deltaData == nil {
			// Store as full entry (delta failed or was larger).
			if writeErr := dataWriter.WriteFullEntry(
				targetBlob.digest,
				targetBlob.data,
			); writeErr != nil {
				tmpFile.Close()
				err = errors.Wrap(writeErr)
				return dataPath, 0, 0, err
			}

			continue
		}

		if writeErr := dataWriter.WriteDeltaEntry(
			targetBlob.digest,
			algByte,
			baseBlob.digest,
			uint64(len(targetBlob.data)),
			dr.deltaData,
		); writeErr != nil {
			tmpFile.Close()
			err = errors.Wrap(writeErr)
			return dataPath, 0, 0, err
		}
	}
```

**Step 3: Add required imports to `pack_v1.go`**

Add `"context"`, `"runtime"`, and `"sync"` to the import block if not already present.

**Step 4: Build to verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go build -tags debug ./...`
Expected: BUILD SUCCESS

**Step 5: Run V1 pack tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go test -v -tags test,debug -run 'TestPackV1' ./src/india/blob_stores/ -count=1`
Expected: All V1 tests pass (TestPackV1WithDelta, TestPackV1WithoutDelta, TestPackV1DeltaFallsBackToFullWhenLarger, TestPackV1SplitsWhenExceedingMaxPackSize)

**Step 6: Run full pack test suite**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go test -v -tags test,debug ./src/india/blob_stores/ -count=1`
Expected: All tests pass

**Step 7: Format**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads && just codemod-go-fmt`

**Step 8: Commit**

```
feat: parallelize delta computation in V1 pack

Compute bsdiff deltas across min(NumCPU, assignmentCount) goroutines
using sync.WaitGroup + indexed deltaResult slice. Trial-and-discard
runs inside each goroutine. Sequential write pass preserves
deterministic archive output. The deltaResult struct creates a seam
for future pipeline evolution (channel-based, ring buffer, etc).
```

---

### Task 4: Add TODO markers for fast blob size

**Files:**
- Modify: `go/internal/india/blob_stores/pack_parallel.go` (add TODO comment)
- Modify: `go/internal/india/blob_stores/pack_v0.go` (add TODO on `GetBlobSize`)
- Modify: `go/internal/india/blob_stores/pack_v1.go` (add TODO on `GetBlobSize`)

**Step 1: Add TODO to `pack_parallel.go`**

Above the `blobSizeFn` type definition, add:

```go
// TODO(near-future): Add BlobSizer capability interface. Local
// hash-bucketed stores without compression/encryption can implement
// GetBlobSize via os.Stat (single syscall) instead of the current
// read-and-discard fallback. The sizeFn callback already accepts this
// signature — a fast implementation plugs in without changing the
// parallel machinery.
```

**Step 2: Add TODO to both `GetBlobSize` methods**

In `pack_v0.go` (line 551) and `pack_v1.go` (line 780), add above each `GetBlobSize` method:

```go
// TODO(near-future): Replace read-and-discard with os.Stat for local
// filesystem stores without compression/encryption. See BlobSizer
// interface comment in pack_parallel.go.
```

**Step 3: Build to verify**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go build -tags debug ./...`
Expected: BUILD SUCCESS

**Step 4: Commit**

```
chore: add TODO markers for fast blob size via os.Stat
```

---

### Task 5: Run full test suite (unit + integration)

**Step 1: Run Go unit tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads/go && go test -v -tags test,debug ./... 2>&1 | tail -50`
Expected: All packages pass

**Step 2: Format check**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads && just codemod-go-fmt`
Expected: No changes

**Step 3: Build binaries**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads && just build`
Expected: BUILD SUCCESS

**Step 4: Run BATS integration tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads && just test-bats`
Expected: All BATS tests pass

**Step 5: If BATS fixture diff, regenerate and commit**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/delta-threads && just test-bats-update-fixtures`
Review diff. If fixtures changed, commit:

```
chore: regenerate migration fixtures
```

---
