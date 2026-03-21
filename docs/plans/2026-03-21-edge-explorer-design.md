# Edge Explorer: Unified Graph Traversal for expandEdges

**Date:** 2026-03-21 **FDR:** 0001 (Object Locks) --- advancing from
experimental to accepted **Scope:** Refactor `expandEdges` to support recursive
blob reference traversal via an `EdgeExplorer` interface, complete FDR-0001
promotion criteria

## Problem

`expandEdges` hardcodes object-edge extraction (types, tags, referenced objects)
and ignores blob references entirely. Blob references carry typed content that
may contain nested references (e.g., git tree blobs listing subtrees). Without
recursive traversal, filtered pulls and clones miss transitive blob
dependencies, and GC can collect reachable blobs.

The FDR-0001 promotion criteria require: "expandEdges follows typed blob refs
recursively."

## Design

### Edges struct (golf/sku)

``` go
type Edges struct {
    Objects []ids.ObjectId
    Blobs   []markl.Id
    Skipped []error
}
```

- `Objects`: object IDs discovered as edges (types, tags, referenced objects,
  objects discovered inside blob content)
- `Blobs`: blob digests discovered as edges (blob references on metadata, nested
  blob refs from content inspection)
- `Skipped`: edges that could not be traversed (blob not found, discovery script
  failed, type has no blob config). Soft failures that don't stop the walk.

### EdgeExplorer interface (golf/sku)

``` go
type EdgeExplorer interface {
    ExploreEdges(object *Transacted) (Edges, error)
}
```

Returns one level of edges for the given object. The walker calls this
repeatedly to traverse the graph. A non-nil error return is a hard failure that
stops the walk. Soft failures go into `Edges.Skipped`.

### expandEdges refactored (sierra/local_working_copy)

``` go
func expandEdges(
    list *sku.HeapTransacted,
    objectStore sku.RepoStore,
    explorer sku.EdgeExplorer,
) (sku.Edges, error)
```

Pure graph walker. Does not know what kinds of edges exist --- delegates
entirely to the explorer. Returns accumulated `Edges` across the full traversal.

Algorithm:

1.  Seed `seen` set from existing heap entries
2.  For each depth iteration (up to `maxEdgeExpansionDepth = 5`):
    a.  For each object in the current frontier, call `explorer.ExploreEdges`
    b.  Collect new object refs not in `seen` → fetch from `objectStore`, add to
        heap
    c.  Collect new blob refs not in `seen` → add to accumulated blob set
    d.  Accumulate `Skipped` errors
3.  Return accumulated `Edges` (all objects, all blobs, all skipped)

When `explorer` is nil, falls back to current behavior (no edge exploration, no
blob output) for backward compatibility.

### Concrete EdgeExplorer (papa/store)

Implementation composes cheap metadata reads with optional blob content
inspection:

1.  **Type edge** --- `object.GetType()` if non-empty and non-builtin
2.  **Tag edges** --- `object.AllTags()`
3.  **Referenced object edges** ---
    `object.GetMetadata().AllReferencedObjects()`
4.  **Blob reference edges** --- `object.GetMetadata().AllBlobReferences()`:
    - Add each blob digest to `Edges.Blobs`
    - For each typed blob ref, check if the type declares `[references]` in its
      blob config
    - If yes: read blob content from blob store, run discovery script via
      `script_config.MakeWriterToWithStdin`, parse output, add discovered object
      refs and blob refs to `Edges`
    - If blob not found or script fails: append to `Edges.Skipped`

This reuses the existing `discoverReferences` infrastructure --- same script
config, same output parser, same type blob store access.

### Caller integration (pullQueryGroupFromWorkingCopy)

``` go
explorer := store.MakeEdgeExplorer(
    remote.GetBlobStore(),
    remote.GetObjectStore(),
    /* typed blob store access */
)

edges, err := expandEdges(list, remote.GetObjectStore(), explorer)
if err != nil {
    return errors.Wrap(err)
}

// Hard-fail on skipped edges for now; refine during testing
if len(edges.Skipped) > 0 {
    return errors.Errorf("edge traversal had %d failures: %s",
        len(edges.Skipped), edges.Skipped[0])
}

importerOptions.AdditionalBlobs = edges.Blobs
```

The importer gains an `AdditionalBlobs` field. After importing all objects
(which transfers their primary blobs inline as today), it copies any additional
blob digests not already in the local store.

## Testing

- **Regenerate v14 fixtures** --- `just test-bats-update-fixtures`. v14 is not
  used in the wild; no backward compatibility concern.
- **Existing migration tests** --- `previous_versions/main.bats` validates old
  stores load and reindex. Objects without blob references produce empty
  `Edges`.
- **New BATS tests in current_version/** --- create objects with blob
  references, pull/clone between repos, verify blob refs are followed and blobs
  transferred.
- **Skipped edge behavior** --- hard fail initially; refine to soft fail for
  specific error types during testing.

## Rollback

No persistent format changes. The `EdgeExplorer` is injected --- passing `nil`
restores the previous behavior. If the feature causes issues, revert the caller
to pass `nil` and the walker falls back to metadata-only extraction.

## Not in scope

- Discovery result caching (see FDR-0001 Future Exploration)
- Edges memory pooling (see FDR-0001 Future Exploration)
- GC reachability walker update (separate effort, same interface)
- Store version bump (binary format already supports blob references)
