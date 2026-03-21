---
date: 2026-03-08
promotion-criteria: all three lock kinds (type, tag, referenced object) and
  typed blob references round-trip through text, inventory list, binary, and
  JSON formats; expandEdges follows typed blob refs recursively; migration tests
  pass for stores created before referenced object locks existed
status: testing
---

# Object Locks

## Problem Statement

When dodder commits an object, it pins the current signatures of the object's
type and tags into the object's metadata. This prevents silent schema drift: if
a type or tag definition changes after the object was committed, the lock
records exactly which version was in effect at commit time.

Today locks cover types and tags but not referenced objects. An object can
reference other objects (e.g. a zettel linking to another zettel), and those
references carry no version pin. If a referenced object changes, there is no
record of which version was in scope when the referencing object was committed.

A second gap: blob references carry no type information. An object can reference
a blob by digest, but the reference doesn't declare how to interpret that blob's
content. This prevents recursive graph traversal --- if a blob contains
references to other blobs (e.g. a git tree listing subtrees and file blobs),
there's no way to discover those edges without ad-hoc content parsing. Without
typed blob references, structures like git trees, archive manifests, and
hierarchical content require a third primitive type, expanding every layer of
dodder that assumes a two-kind world (objects and blobs).

Adding referenced-object locks and typed blob references closes both gaps.

## Lock Kinds

Each object's metadata carries:

- **Type lock** (`! type@signature`) --- pins the object's type to the signature
  of the type-object at commit time. `!` prefix inspired by shebangs.
- **Tag locks** (`tag@signature`) --- one per tag, pins each tag to its
  tag-object signature at commit time.
- **Referenced object locks** (`- ref@signature`) --- one per referenced object,
  pins each reference to the referenced object's signature at commit time.

Referenced objects are discovered by the object's type (which defines how to
extract references from blob content). The metadata stores the result: a map of
fully qualified object IDs to their pinned signatures.

### Lock data model

`markl.Lock[KEY, KEY_PTR]` pairs a key (type ID, tag ID, or object ID) with a
`markl.Id` value (the pinned signature):

    Lock { Key KEY, Value markl.Id }

Referenced object locks reuse the existing `containedObject` struct:

- `ContainedObjectType`: `containedObjectTypeReferencedObject`
- `Lock.Key`: fully qualified object ID (e.g. `one/uno`)
- `Lock.Value`: object signature at commit time

Stored in a `References ContainedObjects` field on metadata, separate from
`Tags`.

Lock values are required in all persistent formats (binary, inventory list) and
optional only in user-facing text input.

### Commit options

`LockfileOptions` controls failure tolerance:

- `AllowTypeFailure` --- if the type object can't be read, skip its lock
- `AllowTagFailure` --- same for tags
- `AllowReferencedObjectFailure` --- same for referenced objects

## Blob References

Blob references pin object-to-blob relationships: an object's content may embed
or refer to a specific blob by its `markl.Id` digest (e.g., an image, a code
snippet, an attachment).

### Data model

``` go
type blobReferenceEntry struct {
    Key      markl.Id    // blob digest
    Alias    string      // optional human-readable alias
    TypeLock markl.Lock  // type ID + pinned signature
}
```

Every blob reference carries a type lock declaring how to interpret the blob's
content. The type is on the reference, not baked into the blob's hash --- two
references to the same blob bytes can declare different types. Storage and
hashing are unchanged.

### Reference discovery

Type-driven: the type definition's `[references]` section configures a script
that runs against the blob. The script emits both object references and blob
references on stdout. Lines starting with `@` are parsed as blob references; all
others as object references.

Discovery script output format:

    one/uno                                # object ref
    alias = one/dos                        # object ref with alias
    @blake2b256-xxxx... !tree@sig          # blob ref with type lock
    alias = @blake2b256-xxxx... !type@sig  # blob ref with alias and type lock

### Type-driven recursive traversal

The type definition declares which fields within a blob's content are references
to other blobs. This lets the graph walker follow edges recursively without
understanding the blob's content directly. The contract is narrow: the type
tells dodder "here are the edges in this node." The core doesn't need to
understand the rest.

Blobs whose type declares no ref fields are leaf nodes --- the common case for
images, attachments, and similar content.

### Visibility and indexing

Typed blobs used as structural plumbing (e.g., tree blobs in a git workspace)
are reachable for traversal and GC but not indexed as queryable entities. Types
may expose blob tree references for external graph traversal without promoting
the blob to an indexed object.

## Serialization

### Hyphence format

References use `<` as an operator with space after it:

    # object ref:
    - < one/uno@sig
    - blog-template < one/uno@sig

    # blob ref:
    - < @blake2b256-abc... !tree@sig
    - hero-image < @blake2b256-def... !image-png@sig

The `@` prefix on the digest distinguishes blob refs from object refs. The type
lock (`!type@sig`) follows the digest, separated by space.

Aliases with unsafe characters are quoted:

    - "unsafe alias with spaces" < one/uno@sig

### Box format

Box format uses `<` prefix. Blob references with type locks are Go-quoted
because they contain spaces:

    # object ref:
    <one/uno@sig

    # blob ref (quoted because of internal space):
    "<@blake2b256-abc... !tree@sig"

Full box line example:

    [one/dos @digest !md@sig project@sig <one/uno@sig "<@blake2b256-abc... !tree@sig"]

### Binary index

Key byte: `BlobReferences = Binary('b')`.

Per blob reference entry:

1.  Blob ID bytes (uint16 length prefix + `markl.Id` binary encoding)
2.  Type lock bytes (uint16 length prefix + type ID + signature)
3.  Alias string (remaining bytes, implicit length)

### Inventory list

    [one/dos @digest !md@sig <one/uno@sig "<@blake2b256-abc... !tree@sig"]

### Format summary

  ------------------------------------------------------------------------------
  Format         Object ref                 Blob ref
  -------------- -------------------------- ------------------------------------
  Hyphence       `- alias < ref@sig`        `- alias < @digest !type@sig`

  Box            `<ref@sig`                 `"<@digest !type@sig"`

  Binary index   key + null + fmt + id      key + type lock + alias

  Inventory list `<ref@sig`                 `"<@digest !type@sig"`
  ------------------------------------------------------------------------------

### Sigil design etymology

- `!` (type) --- inspired by shebangs (`#!/bin/sh`)
- `#` (description) --- inspired by shebangs / comment syntax
- `-` (tag or reference) --- list item
- `<` (reference target) --- inspired by shell input redirection
- `<(...)` (grouped reference) --- inspired by bash process substitution

## Consumers

### Edge expansion in filtered pull (workspace-as-repo)

`expandEdges` (`sierra/local_working_copy/expand_edges.go`) walks all lock edge
kinds on pulled objects and transitively includes them in the transfer:

1.  **Type edges** --- `object.GetType()` -\> fetch the type object
2.  **Tag edges** --- `object.AllTags()` -\> fetch each tag object
3.  **Referenced object edges** ---
    `object.GetMetadata().AllReferencedObjects()` -\> fetch each referenced
    object
4.  **Blob reference edges** --- `object.GetMetadata().AllBlobReferences()` -\>
    for each typed blob ref, if the type declares further ref fields, follow
    recursively

Traversal runs up to 5 levels deep. Objects and blobs already in the transfer
set or missing from the remote are skipped.

### GC reachability

Blob references prevent garbage collection of referenced blobs. With typed blob
references, reachability extends transitively --- a tree blob references other
blobs, which must also be kept alive. The reachability walker uses the type's
ref-discovery declarations to find transitive edges.

## Application: Git Bridge Workspace

The git bridge workspace motivates typed blob references. A git repo is modeled
as:

- A **repo object** (regular dodder object) carrying branch, parents, tree ref,
  author, message
- **Tree-typed blobs** for each directory level, where entries contain name +
  type tag + ref to another tree blob or content blob
- **Content blobs** for file data, with dual digests (dodder-native + git OID)

A "blob tree" is a blob whose type declares that its content is a list of named,
typed references to other blobs. No new primitive needed --- the two-primitive
model (objects and blobs) is preserved.

On the write path, the workspace walks the tree-typed blob hierarchy back into
git tree/blob objects. The dual-digest model means only new or modified blobs
need to be hashed into git's scheme. The `message` field on the repo object is
empty until an explicit commit action --- mutations to the tree mark the
workspace as dirty.

## Implementation Status

### Phase 1: Object Reference Locks (Complete)

Type locks, tag locks, and referenced object locks are implemented across all
serialization formats. Reference discovery via type-configured scripts is
working with shell pipelines and pandoc Lua writers.

Key files:

- `delta/objects/interfaces.go` --- `AllReferencedObjects`, `AddBlobReference`
- `papa/store/reference_discovery.go` --- script-driven discovery
- `hotel/type_blobs/references_config.go` --- `[references]` TOML config
- `golf/sku/edge_explorer.go` --- `EdgeExplorer` interface and `Edges` struct
- `papa/store/edge_explorer.go` --- concrete `EdgeExplorer` with blob discovery
- `sierra/local_working_copy/expand_edges.go` --- graph walker using
  `EdgeExplorer` for filtered pull

### Phase 2: Untyped Blob References (Complete)

Blob references with `markl.Id` keys and optional aliases are implemented. The
`BlobReferences` collection is a peer of `ContainedObjects` on metadata.

Key files:

- `delta/objects/blob_reference.go` --- `BlobReferences` collection
- `india/stream_index/binary_encoder.go` --- `BlobReferences` key encoder
- `india/stream_index/binary_decoder.go` --- `BlobReferences` key decoder
- `foxtrot/object_metadata_fmt_hyphence/formatter_components.go` ---
  `writeBlobReferences`
- `foxtrot/object_metadata_fmt_hyphence/text_parser2.go` --- `readBlobReference`
- `echo/object_metadata_box_builder/main.go` --- `AddBlobReferences`

### Phase 3: Typed Blob References (Complete)

- [x] Add `TypeLock` field to `blobReferenceEntry`
- [x] Update hyphence parser/formatter for `< @digest !type@sig` syntax
- [x] Update box parser/formatter for `<(...)` grouping syntax (both object and
  blob refs)
- [x] Update binary encoder/decoder for type lock field
- [x] Update reference discovery output format to include type lock
- [x] Implement recursive traversal via `EdgeExplorer` interface

### Phase 4: Remaining Work

- [ ] Update GC reachability walker to follow typed blob reference edges
- [ ] Audit existing untyped blob references for type annotation

## Future Exploration

### Discovery result caching

`ExploreEdges` (the recursive blob content inspection) runs a discovery script
for every blob reference whose type declares ref fields. For large graphs with
repeated blob types, caching results keyed on `(type_id, type_sig, blob_digest)`
would avoid redundant script invocations. The cache key captures all inputs:
same type definition at the same version applied to the same blob content should
produce the same edges.

**Prerequisite:** Discovery scripts must be deterministic given the same type
definition and blob content. This is not guaranteed today --- scripts can read
external state, use timestamps, etc. Enforcing or verifying determinism is a
separate effort.

### Edges memory pooling

The `Edges` struct returned by `ExploreEdges` allocates fresh slices per call.
For large graphs (thousands of objects, each with a handful of edges), this
creates GC pressure. Options to explore:

- Pooled `Edges` via `sync.Pool` with `Reset()` method (walker returns after
  processing each batch)
- Arena allocator if Go adds arena support for slices
- Pre-allocated slice capacity hint based on type metadata

## Limitations

- **Blob edge traversal is one level deep.** `expandEdges` runs discovery
  scripts on blob references in object metadata, collecting nested blob and
  object refs. But newly discovered blobs are not themselves inspected for
  further references. Object edge traversal IS recursive (up to 5 levels), but
  blob→blob→blob chains are not followed. The git bridge workspace (tree blobs
  referencing subtree blobs) will need this extended to true transitive blob
  traversal.
- Builtin types are not locked (there is a TODO to address this).
- Lock values are not overwritten once set during a commit --- re-committing an
  object does not update its locks unless the lock is explicitly cleared first.
- Reference discovery is covered by a separate design:
  `docs/plans/2026-03-07-object-reference-discovery-design.md`. First
  implementation uses external commands, with Lua hooks as future work.

## Open Issues

- [#35](https://github.com/amarbel-llc/dodder/issues/35) --- hyphenated type
  names create phantom blobless type object during pull
- [#36](https://github.com/amarbel-llc/dodder/issues/36) --- moderate test
  coverage gaps (discovery errors, multi-ref, aliases, GC)

## More Information

- [FDR-0005: Workspace-as-Repo](0005-workspace-as-repo.md) --- workspace
  isolation, filtered pull uses `expandEdges`
- [FDR-0006: Two-Stage Commit](0006-two-stage-commit.md) --- plan-based batch
  commit
- [FDR-0007: Pluggable Checkout Stores](0007-checkout-bridges.md) --- typed
  blobs enable non-filesystem checkout decomposition
