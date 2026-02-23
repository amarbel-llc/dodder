# Sync Cross-Hash Digest Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable madder `sync` to copy blobs between stores with different hash types, using symlinks in multi-hash stores for dual-digest lookup.

**Architecture:** New `BlobForeignDigestAdder` optional interface on blob stores. `CopyBlobIfNecessary` calls it after cross-hash copies. `localHashBucketed` implements it via relative symlinks. Sync command gates cross-hash behavior on destination type and user consent.

**Tech Stack:** Go, dodder blob store interfaces, `os.Symlink`, `filepath.Rel`, `env_ui.Confirm` for interactive prompts, `tap-dancer` for TAP-14 output.

---

### Task 1: Add `BlobForeignDigestAdder` interface

**Files:**
- Modify: `go/src/alfa/domain_interfaces/blob_store.go:91`

**Step 1: Add the interface**

Add to the end of `go/src/alfa/domain_interfaces/blob_store.go`, before the closing `)` or after the last type block:

```go
type BlobForeignDigestAdder interface {
	AddForeignBlobDigestForNativeDigest(foreign, native MarklId) error
}
```

**Step 2: Verify build**

Run: `go build ./go/src/alfa/...`
Expected: success (interface is unused so far)

**Step 3: Commit**

```
feat: add BlobForeignDigestAdder interface for cross-hash blob mapping
```

---

### Task 2: Fix `blobReaderFrom` to use digest's hash format

**Files:**
- Modify: `go/src/india/blob_stores/store_local_hash_bucketed.go:186-240`
- Test: `go/src/india/blob_stores/` (existing tests)

**Step 1: Write the failing test**

Create `go/src/india/blob_stores/local_hash_bucketed_test.go`:

```go
package blob_stores_test

import (
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func TestMakeBlobReaderUsesDigestHashFormat(t1 *testing.T) {
	t := ui.T{T: t1}

	// Create a multi-hash local store with blake2b-256 as default
	// Write a blob using sha256
	// Read back using the sha256 digest
	// Verify the reader's GetMarklId() returns a sha256 digest, not blake2b-256

	// This test requires setting up a local hash bucketed store with multiHash=true.
	// Use the existing test infrastructure to create a temp directory and config.

	_ = t
	_ = markl.FormatHashSha256
	// TODO: flesh out after understanding test setup patterns in blob_stores
	t.Skip("placeholder - implement with store test helpers")
}
```

Note: This test needs access to unexported `makeLocalHashBucketed`. The real test
may need to go through the public `MakeBlobStores` factory or use an exported
test helper. Check existing test patterns in `go/src/india/blob_stores/` and
adapt. The key assertion is: writing a blob with sha256 to a multi-hash store,
then reading it back, yields a sha256 digest from `GetMarklId()`.

**Step 2: Implement the fix**

In `go/src/india/blob_stores/store_local_hash_bucketed.go`, modify `blobReaderFrom`
(lines 186-240). Change the `NewFileReaderOrErrNotExist` call to use the digest's
hash format:

Before (line 218-220):
```go
if readCloser, err = env_dir.NewFileReaderOrErrNotExist(
	blobStore.makeEnvDirConfig(nil),
	basePath,
); err != nil {
```

After:
```go
var hashFormat markl.FormatHash

if hashFormat, err = markl.GetFormatHashOrError(
	digest.GetMarklFormat().GetMarklFormatId(),
); err != nil {
	err = errors.Wrap(err)
	return readCloser, err
}

if readCloser, err = env_dir.NewFileReaderOrErrNotExist(
	blobStore.makeEnvDirConfig(hashFormat),
	basePath,
); err != nil {
```

This requires adding `"code.linenisgreat.com/dodder/go/src/echo/markl"` to the
imports if not already present (it already is — line 14).

**Step 3: Verify build and tests**

Run: `just test-go`
Expected: all tests pass

**Step 4: Commit**

```
fix: use digest's hash format in blobReaderFrom instead of store default
```

---

### Task 3: Implement `AddForeignBlobDigestForNativeDigest` on `localHashBucketed`

**Files:**
- Modify: `go/src/india/blob_stores/store_local_hash_bucketed.go`

**Step 1: Write the failing test**

Add to `go/src/india/blob_stores/local_hash_bucketed_test.go`:

```go
func TestAddForeignBlobDigestCreatesSymlink(t1 *testing.T) {
	t := ui.T{T: t1}

	// 1. Create a multi-hash local store
	// 2. Write a blob using blake2b-256 (native)
	// 3. Call AddForeignBlobDigestForNativeDigest(sha256Digest, blake2bDigest)
	// 4. Assert: file at sha256 path is a symlink
	// 5. Assert: HasBlob(sha256Digest) returns true
	// 6. Assert: MakeBlobReader(sha256Digest) succeeds and reads correct content

	_ = t
	t.Skip("placeholder - implement with store test helpers")
}

func TestAddForeignBlobDigestErrorsOnSingleHash(t1 *testing.T) {
	t := ui.T{T: t1}

	// 1. Create a single-hash local store (multiHash=false)
	// 2. Call AddForeignBlobDigestForNativeDigest
	// 3. Assert: returns error

	_ = t
	t.Skip("placeholder - implement with store test helpers")
}
```

**Step 2: Implement `AddForeignBlobDigestForNativeDigest`**

Add to `go/src/india/blob_stores/store_local_hash_bucketed.go`:

```go
var _ domain_interfaces.BlobForeignDigestAdder = localHashBucketed{}

func (blobStore localHashBucketed) AddForeignBlobDigestForNativeDigest(
	foreign domain_interfaces.MarklId,
	native domain_interfaces.MarklId,
) (err error) {
	if !blobStore.multiHash {
		err = errors.Errorf(
			"single-hash store does not support foreign digest mapping",
		)
		return err
	}

	nativePath := env_dir.MakeHashBucketPathFromMerkleId(
		native,
		blobStore.buckets,
		blobStore.multiHash,
		blobStore.basePath,
	)

	foreignPath := env_dir.MakeHashBucketPathFromMerkleId(
		foreign,
		blobStore.buckets,
		blobStore.multiHash,
		blobStore.basePath,
	)

	foreignDir := filepath.Dir(foreignPath)

	if err = os.MkdirAll(foreignDir, os.ModeDir|0o755); err != nil {
		err = errors.Wrap(err)
		return err
	}

	relTarget, err := filepath.Rel(foreignDir, nativePath)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = os.Symlink(relTarget, foreignPath); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
```

Add `"os"` to imports (already present — line 5). Add
`"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"` if needed (check
existing imports).

**Step 3: Verify build and tests**

Run: `just test-go`
Expected: all tests pass

**Step 4: Commit**

```
feat: implement BlobForeignDigestAdder for localHashBucketed via symlinks
```

---

### Task 4: Modify `CopyBlobIfNecessary` for cross-hash support

**Files:**
- Modify: `go/src/india/blob_stores/copy.go:104-128`

**Step 1: Write the failing test**

This is best tested via integration (BATS) since `CopyBlobIfNecessary` operates
on full blob stores. Add a unit test if a test helper exists for creating in-memory
or temp-dir blob stores. Otherwise, rely on the BATS test in Task 6.

**Step 2: Implement cross-hash verification**

Modify `go/src/india/blob_stores/copy.go` lines 104-128. Replace the verification
block:

```go
readerDigest := readCloser.GetMarklId()
writerDigest := writeCloser.GetMarklId()

if !markl.Equals(readerDigest, expectedDigest) {
	copyResult.setErrorAfterCopy(
		copyResult.bytesWritten,
		errors.Errorf(
			"lookup digest was %s while read digest was %s",
			expectedDigest,
			readerDigest,
		),
	)

	return copyResult
}

crossHash := expectedDigest.GetMarklFormat().GetMarklFormatId() !=
	writerDigest.GetMarklFormat().GetMarklFormatId()

if crossHash {
	if adder, ok := dst.(domain_interfaces.BlobForeignDigestAdder); ok {
		if err := adder.AddForeignBlobDigestForNativeDigest(
			expectedDigest,
			writerDigest,
		); err != nil {
			copyResult.setErrorAfterCopy(copyResult.bytesWritten, err)
			return copyResult
		}
	}
} else {
	if err := markl.AssertEqual(expectedDigest, writerDigest); err != nil {
		copyResult.setErrorAfterCopy(
			copyResult.bytesWritten,
			err,
		)

		return copyResult
	}
}
```

Add `"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"` to imports.

**Step 3: Verify build and tests**

Run: `just test-go`
Expected: all tests pass (no behavioral change for same-hash copies)

**Step 4: Commit**

```
feat: support cross-hash blob copy with foreign digest mapping
```

---

### Task 5: Add `-allow-rehashing` flag and cross-hash gating to sync command

**Files:**
- Modify: `go/src/lima/commands_madder/sync.go`
- Modify: `go/src/kilo/blob_transfers/main.go`

**Step 1: Add `AllowRehashing` field to `Sync` struct and flag definition**

In `go/src/lima/commands_madder/sync.go`:

```go
type Sync struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore

	AllowRehashing bool
	Limit          int
}
```

Add to `SetFlagDefinitions`:

```go
flagSet.BoolVar(
	&cmd.AllowRehashing,
	"allow-rehashing",
	false,
	"allow syncing to stores with a different hash type (source digests not preserved in single-hash destinations)",
)
```

**Step 2: Add cross-hash detection and gating in `runStore`**

In `runStore`, after getting `source` and `destination`, before creating the
`blobImporter`, add cross-hash detection:

```go
func (cmd Sync) runStore(
	req command.Request,
	envBlobStore env_repo.BlobStoreEnv,
	source blob_stores.BlobStoreInitialized,
	destination blob_stores.BlobStoreMap,
) {
	if len(destination) == 0 {
		errors.ContextCancelWithBadRequestf(
			req,
			"only one blob store, nothing to sync",
		)
		return
	}

	sourceHashType := source.GetBlobStore().GetDefaultHashType()
	useDestinationHashType := false

	for _, dst := range destination {
		dstHashType := dst.GetBlobStore().GetDefaultHashType()

		if sourceHashType.GetMarklFormatId() == dstHashType.GetMarklFormatId() {
			continue
		}

		// Cross-hash detected
		_, isAdder := dst.GetBlobStore().(domain_interfaces.BlobForeignDigestAdder)

		if !isAdder && !cmd.AllowRehashing {
			if !envBlobStore.Confirm(
				fmt.Sprintf(
					"Destination %q uses %s but source uses %s. Rehashing will not preserve source digests. Continue?",
					dst.Path.GetId(),
					dstHashType.GetMarklFormatId(),
					sourceHashType.GetMarklFormatId(),
				),
				"",
			) {
				errors.ContextCancelWithBadRequestf(
					req,
					"cross-hash sync refused: destination %q uses %s, source uses %s. Use -allow-rehashing to skip this check",
					dst.Path.GetId(),
					dstHashType.GetMarklFormatId(),
					sourceHashType.GetMarklFormatId(),
				)
				return
			}
		}

		useDestinationHashType = true
	}

	blobImporter := blob_transfers.MakeBlobImporter(
		envBlobStore,
		source,
		destination,
	)

	blobImporter.UseDestinationHashType = useDestinationHashType

	// ... rest of existing runStore code (CopierDelegate, defer, loop)
```

Add imports: `"fmt"`, `"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"`.

**Step 3: Verify build and tests**

Run: `just test-go`
Expected: all tests pass

**Step 4: Commit**

```
feat: add -allow-rehashing flag and cross-hash gating to sync command
```

---

### Task 6: Add BATS integration test for cross-hash sync

**Files:**
- Modify: `zz-tests_bats/blob_store_sync.bats`

**Step 1: Write the integration test**

Add to `zz-tests_bats/blob_store_sync.bats`:

```bash
function blob_store_sync_cross_hash_multi_hash { # @test
	# Init source store with sha256
	run_madder init --hash-type sha256 source
	assert_success

	# Init destination store with blake2b-256 and multi-hash
	run_madder init --hash-type blake2b-256 --multi-hash dest
	assert_success

	# Write a blob to source
	echo "test content" | run_madder write source
	assert_success

	# Sync source -> dest
	run_madder sync source dest
	assert_success

	# Verify blob exists in destination under both digests
	# TODO: assert that HasBlob works for both sha256 and blake2b-256
}
```

Note: The exact test setup depends on how madder commands accept store IDs and
hash type flags. Adapt based on actual CLI interface. Check existing BATS helpers
in `zz-tests_bats/common.bash` for `run_madder` and store setup patterns.

**Step 2: Run the test**

Run: `just test-bats-targets blob_store_sync.bats`
Expected: new test passes

**Step 3: Commit**

```
test: add BATS integration test for cross-hash blob sync
```

---

### Task 7: Convert sync command to TAP-14 output

**Files:**
- Modify: `go/src/lima/commands_madder/sync.go`

**Step 1: Write the failing test**

Update the BATS test from Task 6 to assert TAP-14 output format:

```bash
function blob_store_sync_outputs_tap { # @test
	# Setup: create two stores and write a blob
	# ...

	run_madder sync source dest
	assert_success
	assert_line --index 0 "TAP version 14"
	assert_line --regexp "^ok 1 -"
	assert_line --regexp "^1\.\."
}
```

**Step 2: Convert sync output to TAP-14**

In `go/src/lima/commands_madder/sync.go`, replace `ui.Out().Print` and
`ui.Err().Printf` with TAP writer calls:

```go
import (
	"fmt"
	"os"

	tap "github.com/amarbel-llc/tap-dancer/go"
	// ... existing imports
)
```

In `runStore`:

```go
tw := tap.NewWriter(os.Stdout)

// ... cross-hash detection ...

// ... copy loop: replace ui.Err().Print with tw.Ok / tw.NotOk ...

for blobId, errIter := range source.AllBlobs() {
	if errIter != nil {
		tw.NotOk(
			fmt.Sprintf("read %s", blobId),
			map[string]string{
				"severity": "fail",
				"message":  errIter.Error(),
			},
		)
		continue
	}

	if err := blobImporter.ImportBlobIfNecessary(blobId, nil); err != nil {
		// ... handle error with tw.NotOk ...
	} else {
		// ... emit tw.Ok or tw.Skip based on copy result ...
	}

	// ... limit check ...
}

tw.Comment(fmt.Sprintf(
	"Successes: %d, Failures: %d, Ignored: %d, Total: %d",
	blobImporter.Counts.Succeeded,
	blobImporter.Counts.Failed,
	blobImporter.Counts.Ignored,
	blobImporter.Counts.Total,
))

tw.Plan()
```

Note: The exact TAP mapping depends on whether ignored (already-exists) blobs
should emit `ok` with a skip directive or be silent. Follow the pattern in
`pack.go` for reference.

**Step 3: Verify tests**

Run: `just test`
Expected: all tests pass (unit + BATS)

**Step 4: Commit**

```
refactor: output TAP-14 from madder sync command
```

---

### Task 8: Update blob_store_sync.bats skipped test

**Files:**
- Modify: `zz-tests_bats/blob_store_sync.bats:18-38`

The existing `blob_store_sync_twice` test is skipped with a TODO about migrating
to madder blob stores. If the TAP-14 output changes break its assertions, update
the expected output patterns to match TAP-14. If it remains skipped, leave it.

**Step 1: Evaluate**

Check if `blob_store_sync_twice` can be unskipped with updated assertions. If not,
leave the skip and update the TODO comment.

**Step 2: Commit (if changes)**

```
test: update blob_store_sync assertions for TAP-14 output
```

---

## Dependency Order

```
Task 1 (interface)
  └→ Task 2 (reader fix) — independent of interface but ship together
  └→ Task 3 (symlink impl) — depends on Task 1
       └→ Task 4 (copy.go) — depends on Tasks 1, 2, 3
            └→ Task 5 (sync flags) — depends on Task 4
                 └→ Task 6 (BATS test) — depends on Task 5
                      └→ Task 7 (TAP output) — depends on Task 6
                           └→ Task 8 (update old test) — depends on Task 7
```

## Notes

- Tasks 1-4 are the core mechanism. Tasks 5-6 wire it to the CLI. Tasks 7-8 are
  the TAP conversion (separate concern).
- The BATS tests (Tasks 6, 8) may need adaptation based on how madder CLI flags
  actually work for store selection and hash type configuration. Check existing
  test patterns.
- Archive stores do NOT implement `BlobForeignDigestAdder` in this iteration.
  See `TODO.md` for future work.
