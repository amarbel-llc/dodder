# Edge Explorer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Refactor `expandEdges` into a pure graph walker that delegates edge
discovery to an `EdgeExplorer` interface, enabling recursive blob reference
traversal and completing FDR-0001 promotion criteria.

**Architecture:** Define `Edges` struct and `EdgeExplorer` interface in
`golf/sku`. Rewrite `expandEdges` in `sierra/local_working_copy` as a generic
walker. Implement the concrete explorer in `papa/store` reusing existing
`discoverReferences` infrastructure. Wire into `pullQueryGroupFromWorkingCopy`
with `AdditionalBlobs` on the importer.

**Tech Stack:** Go, existing `script_config`, `type_blobs`, `blob_stores`
packages.

**Rollback:** Pass `nil` as the explorer to restore previous behavior. No
persistent format changes.

--------------------------------------------------------------------------------

### Task 1: Add `Edges` struct and `EdgeExplorer` interface to golf/sku

**Files:** - Create: `go/internal/golf/sku/edge_explorer.go`

**Step 1: Create the file**

``` go
package sku

import (
    "code.linenisgreat.com/dodder/go/internal/bravo/ids"
    "code.linenisgreat.com/dodder/go/internal/bravo/markl"
)

type Edges struct {
    Objects []ids.ObjectId
    Blobs   []markl.Id
    Skipped []error
}

type EdgeExplorer interface {
    ExploreEdges(object *Transacted) (Edges, error)
}
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./internal/golf/sku/` Expected: clean compile, no
errors.

**Step 3: Commit**

Message: `feat(sku): add Edges struct and EdgeExplorer interface`

--------------------------------------------------------------------------------

### Task 2: Refactor `expandEdges` to use `EdgeExplorer`

**Files:** - Modify: `go/internal/sierra/local_working_copy/expand_edges.go`

**Step 1: Rewrite `expandEdges`**

Replace the entire file with a generic walker that delegates to the explorer.
The new signature:

``` go
func expandEdges(
    list *sku.HeapTransacted,
    objectStore sku.RepoStore,
    explorer sku.EdgeExplorer,
) (allEdges sku.Edges, err error) {
```

Algorithm (preserving the existing structure):

1.  If `explorer == nil`, return empty edges (backward compat).
2.  Seed `seen` map (string → struct{}) from existing heap entries and a
    `seenBlobs` map for blob dedup.
3.  Depth loop (0 to `maxEdgeExpansionDepth`):
    a.  For each object in `list.All()`, call `explorer.ExploreEdges(object)`.
    b.  For each `edges.Objects` entry not in `seen`: mark seen, append to
        `pendingIds`.
    c.  For each `edges.Blobs` entry not in `seenBlobs`: mark seen, append to
        `allEdges.Blobs`.
    d.  Append `edges.Skipped` to `allEdges.Skipped`.
    e.  If no pending IDs, break.
    f.  For each pending ID: fetch from `objectStore` via
        `sku.GetTransactedPool().GetWithRepool()`, add to `list`. On
        `errors.IsErrNotFound`, skip. On other error, return.
4.  Copy accumulated objects into `allEdges.Objects` (for completeness).
5.  Return `allEdges`.

**Important:** The existing `errors.IsErrNotFound` skip behavior for objects not
in the remote store must be preserved --- this handles objects that exist
locally but not on the remote.

**Step 2: Verify it compiles**

Run: `cd go && go build ./internal/sierra/local_working_copy/` Expected: compile
error in `local_op_pull.go` (call site has wrong arity). This is expected --- we
fix it in Task 4.

**Step 3: Commit (WIP, won't compile yet)**

Message: `refactor(expand_edges): delegate to EdgeExplorer interface`

--------------------------------------------------------------------------------

### Task 3: Implement concrete `EdgeExplorer` in papa/store

**Files:** - Create: `go/internal/papa/store/edge_explorer.go`

**Step 1: Create the file**

The concrete explorer needs three dependencies: - Object store (to read type
objects and get their blob configs) - Blob store (to read blob content for
discovery scripts) - Typed blob store (to parse type blobs and get
`ReferencesConfig`)

``` go
package store

import (
    "bytes"
    "io"

    "code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
    "code.linenisgreat.com/dodder/go/internal/bravo/ids"
    "code.linenisgreat.com/dodder/go/internal/bravo/markl"
    "code.linenisgreat.com/dodder/go/internal/golf/sku"
    "code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
    "code.linenisgreat.com/dodder/go/internal/juliett/typed_blob_store"
    "code.linenisgreat.com/dodder/go/lib/0/interfaces"
    "code.linenisgreat.com/dodder/go/lib/bravo/errors"
    "code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

type edgeExplorer struct {
    objectStore    sku.RepoStore
    blobStore      domain_interfaces.BlobStore
    typedBlobStore typed_blob_store.Stores
}

func MakeEdgeExplorer(
    objectStore sku.RepoStore,
    blobStore domain_interfaces.BlobStore,
    typedBlobStore typed_blob_store.Stores,
) sku.EdgeExplorer {
    return &edgeExplorer{
        objectStore:    objectStore,
        blobStore:      blobStore,
        typedBlobStore: typedBlobStore,
    }
}
```

**Step 2: Implement `ExploreEdges`**

``` go
func (e *edgeExplorer) ExploreEdges(
    object *sku.Transacted,
) (edges sku.Edges, err error) {
    // 1. Type edge
    if typeId := object.GetType(); !typeId.IsEmpty() && !ids.IsBuiltin(typeId) {
        var oid ids.ObjectId
        if err = oid.SetWithId(typeId); err != nil {
            return edges, errors.Wrap(err)
        }
        edges.Objects = append(edges.Objects, oid)
    }

    // 2. Tag edges
    for tag := range object.AllTags() {
        var oid ids.ObjectId
        if err = oid.SetWithId(tag); err != nil {
            return edges, errors.Wrap(err)
        }
        edges.Objects = append(edges.Objects, oid)
    }

    // 3. Referenced object edges
    for ref := range object.GetMetadata().AllReferencedObjects() {
        refCopy := ref
        edges.Objects = append(edges.Objects, refCopy)
    }

    // 4. Blob reference edges
    for blobDigest := range object.GetMetadata().AllBlobReferences() {
        blobCopy := blobDigest
        edges.Blobs = append(edges.Blobs, blobCopy)

        typeLock := object.GetMetadata().GetBlobReferenceTypeLock(blobDigest)
        if typeLock.GetKey().IsEmpty() {
            continue
        }

        nestedEdges, discoverErr := e.discoverBlobEdges(blobDigest, typeLock)
        if discoverErr != nil {
            edges.Skipped = append(edges.Skipped,
                errors.Wrapf(discoverErr, "blob %s", blobDigest.String()))
            continue
        }

        edges.Objects = append(edges.Objects, nestedEdges.Objects...)
        edges.Blobs = append(edges.Blobs, nestedEdges.Blobs...)
    }

    return edges, nil
}
```

**Step 3: Implement `discoverBlobEdges` (private helper)**

This reuses `parseReferenceOutput` (already in `reference_discovery.go`). The
method reads the blob content, looks up the type's `[references]` config, runs
the discovery script, and parses the output into edges.

``` go
func (e *edgeExplorer) discoverBlobEdges(
    blobDigest markl.Id,
    typeLock markl.Lock[ids.SeqId, *ids.SeqId],
) (edges sku.Edges, err error) {
    // Read the type object to get its blob config
    typeId := typeLock.GetKey()

    var typeObject sku.Transacted
    var typeOid ids.ObjectId
    if err = typeOid.SetWithId(&typeId); err != nil {
        return edges, errors.Wrap(err)
    }

    fetched, repool := sku.GetTransactedPool().GetWithRepool()
    defer repool()

    if err = e.objectStore.ReadOneInto(&typeOid, fetched); err != nil {
        return edges, errors.Wrap(err)
    }

    // Parse the type blob to get ReferencesConfig
    var blob type_blobs.Blob
    {
        var blobRepool interfaces.FuncRepool

        if blob, blobRepool, _, err = e.typedBlobStore.Type.ParseTypedBlob(
            fetched.GetType(),
            fetched.GetBlobDigest(),
        ); err != nil {
            return edges, errors.Wrap(err)
        }

        defer blobRepool()
    }

    referencesConfig := blob.GetReferences()
    if referencesConfig == nil {
        return edges, nil // type has no discovery config, nothing to follow
    }

    // Read the blob content
    blobReader, err := e.blobStore.MakeBlobReader(blobDigest)
    if err != nil {
        return edges, errors.Wrap(err)
    }

    defer errors.DeferredCloser(&err, blobReader)

    var stdout io.WriterTo

    if stdout, err = script_config.MakeWriterToWithStdin(
        &referencesConfig.ScriptConfig,
        nil,
        blobReader,
    ); err != nil {
        if referencesConfig.Optional {
            return edges, nil
        }

        return edges, errors.Wrap(err)
    }

    var buf bytes.Buffer
    if _, err = stdout.WriteTo(&buf); err != nil {
        if referencesConfig.Optional {
            return edges, nil
        }

        return edges, errors.Wrap(err)
    }

    var refs []discoveredReference
    if refs, err = parseReferenceOutput(buf.String()); err != nil {
        return edges, errors.Wrap(err)
    }

    for _, ref := range refs {
        if ref.BlobId != "" {
            var id markl.Id
            if err = id.Set(ref.BlobId); err != nil {
                return edges, errors.Wrapf(err, "invalid blob ref: %q", ref.BlobId)
            }
            edges.Blobs = append(edges.Blobs, id)
        } else if ref.ObjectId != "" {
            var oid ids.ObjectId
            if err = oid.Set(ref.ObjectId); err != nil {
                return edges, errors.Wrapf(err, "invalid object ref: %q", ref.ObjectId)
            }
            edges.Objects = append(edges.Objects, oid)
        }
    }

    return edges, nil
}
```

**Step 4: Verify it compiles**

Run: `cd go && go build ./internal/papa/store/` Expected: clean compile.

**Step 5: Commit**

Message: `feat(store): implement concrete EdgeExplorer with blob discovery`

--------------------------------------------------------------------------------

### Task 4: Add `AdditionalBlobs` to importer and wire caller

**Files:** - Modify: `go/internal/quebec/repo/importer.go` (add
`AdditionalBlobs` field to `ImporterOptions`) - Modify:
`go/internal/romeo/remote_transfer/import.go` (copy additional blobs after
object import loop) - Modify:
`go/internal/sierra/local_working_copy/local_op_pull.go` (construct explorer,
handle edges)

**Step 1: Add `AdditionalBlobs` field to `ImporterOptions`**

In `go/internal/quebec/repo/importer.go`, add after the existing fields:

``` go
AdditionalBlobs []markl.Id
```

This requires importing
`"code.linenisgreat.com/dodder/go/internal/bravo/markl"`.

**Step 2: Copy additional blobs in `ImportSeq`**

In `go/internal/romeo/remote_transfer/import.go`, after the main object import
loop completes (after all objects have been imported), add a loop that copies
each `AdditionalBlobs` entry from the remote blob store to the local blob store
using the existing `blob_transfers.CopyBlobIfNecessary` function (or the
importer's `blobImporter`). Only copy if: - `importer.remoteBlobStore != nil` -
The blob doesn't already exist locally

Check where `ImportBlobIfNecessary` calls into `blob_transfers` and follow the
same pattern. Skip blobs that aren't found on the remote (soft fail, same as
missing primary blobs).

**Step 3: Wire `pullQueryGroupFromWorkingCopy`**

In `go/internal/sierra/local_working_copy/local_op_pull.go`, update the
`pullQueryGroupFromWorkingCopy` function:

``` go
// Before: expandEdges(list, remote.GetObjectStore())
// After:
explorer := store.MakeEdgeExplorer(
    remote.GetObjectStore(),
    remote.GetBlobStore(),
    /* need typed blob store access from remote */
)

edges, err := expandEdges(list, remote.GetObjectStore(), explorer)
if err != nil {
    return errors.Wrap(err)
}

// Hard-fail on skipped edges for now
if len(edges.Skipped) > 0 {
    return errors.Errorf("edge traversal had %d failures: %s",
        len(edges.Skipped), edges.Skipped[0])
}

importerOptions.AdditionalBlobs = edges.Blobs
```

**Important:** Getting the `typed_blob_store.Stores` from the remote repo
requires checking whether `repo.Repo` exposes it. If not, you may need to add a
method to the `repo.Repo` interface or use a type assertion. Check what
`papa/store.Store` exposes and what `repo.Repo` provides. The
`GetTypedBlobStore()` method exists on `papa/store.Store` --- verify whether
it's accessible through the `repo.Repo` interface or needs to be added.

**Step 4: Verify it compiles**

Run: `cd go && go build ./...` Expected: clean compile.

**Step 5: Run existing tests**

Run: `just test-go` (from repo root) Expected: all unit tests pass. The
refactored `expandEdges` should produce identical behavior for the existing
object-only traversal.

**Step 6: Commit**

Message: `feat: wire EdgeExplorer into pull with AdditionalBlobs transfer`

--------------------------------------------------------------------------------

### Task 5: Run full integration tests

**Step 1: Build binaries**

Run: `just build` (from repo root)

**Step 2: Run full test suite**

Run: `just test` (from repo root) Expected: all tests pass. Existing blob
reference tests in `zz-tests_bats/current_version/show.bats` (lines 958-1002)
continue to work. Pull/push/clone tests pass unchanged.

If fixture-related failures occur, regenerate: `just test-bats-update-fixtures`

**Step 3: Commit fixture changes if any**

Message: `chore: regenerate v14 fixtures for edge explorer`

--------------------------------------------------------------------------------

### Task 6: Add BATS integration test for blob edge traversal during pull

**Files:** - Create or modify: `zz-tests_bats/current_version/pull.bats` (add
new test at end)

**Step 1: Write the test**

Add a new test function that verifies blob references are followed during a
filtered pull between two repos. The test should:

1.  Init repo A (parent) with a type that has `[references]` discovery
2.  Create a zettel in repo A whose blob content contains a blob reference
3.  Init repo B (workspace-repo or direct clone)
4.  Pull from A to B
5.  Verify the referenced blob exists in B's blob store

Pattern to follow: the existing `pull_direct_local_path_no_conflicts` test
(pull.bats line 552) for the two-repo setup, and
`show_zettel_with_discovered_blob_references` (show.bats line 958) for the type
with `[references]` config.

The test name should be `pull_direct_blob_references_transferred`.

**Step 2: Run the new test to verify it passes**

Run: `just test-bats-targets pull.bats` Expected: new test passes alongside
existing pull tests.

**Step 3: Commit**

Message: `test: add BATS test for blob reference traversal during pull`

--------------------------------------------------------------------------------

### Task 7: Update FDR-0001 implementation status

**Files:** - Modify: `docs/features/0001-object-locks.md`

**Step 1: Update the Phase 3 checklist**

In the "Implementation Status" section, check off the completed items:

- [x] Implement recursive traversal in `expandEdges` for typed blob refs

**Step 2: Evaluate promotion criteria**

The promotion criteria are: \> all three lock kinds (type, tag, referenced
object) and typed blob references \> round-trip through text, inventory list,
binary, and JSON formats; expandEdges \> follows typed blob refs recursively;
migration tests pass for stores created \> before referenced object locks
existed

Check each: - Three lock kinds round-trip: already done (Phase 1) - Typed blob
references round-trip: already done (Phase 3, previous work) - expandEdges
follows typed blob refs: done (this plan) - Migration tests: v14 not used in
wild, regenerated fixtures pass

If all criteria are met, update status from `experimental` to `accepted`.

**Step 3: Commit**

Message: `docs: update FDR-0001 status after edge explorer implementation`

--------------------------------------------------------------------------------

### Task 8: Final full test run

**Step 1: Build and test everything**

Run: `just build && just test` Expected: all unit tests and integration tests
pass.

**Step 2: Review the diff**

Run: `git log --oneline master..HEAD` to review all commits on the branch.
Verify each commit is independently meaningful and the full change set matches
the design doc.
