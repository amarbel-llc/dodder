# Referenced Object Locks Design

## Summary

Extend the object lock system to pin referenced objects alongside types and tags.
When an object references other objects (e.g. a markdown zettel linking to a
template), the lock captures which version of each referenced object was in scope
at commit time. References can optionally carry aliases that map blob-local names
to fully qualified object IDs.

## Motivation

Types and tags are already locked. Referenced objects are the remaining gap:
if `one/dos` references `one/uno` and `one/uno` changes, there is no record of
which version was in effect when `one/dos` was committed. This makes it
impossible to reconstruct the exact dependency graph at a point in time.

## Data Model

### containedObject reuse

Referenced objects reuse the existing `containedObject` struct in
`delta/objects/`:

```go
containedObject struct {
    ContainedObjectType ContainedObjectType  // containedObjectTypeReferencedObject
    Alias               SeqId                // optional blob-local alias
    Lock                markl.Lock[SeqId, *SeqId]  // Key=object ID, Value=signature
}
```

New `ContainedObjectType` value:

```go
const (
    containedObjectTypeMetadataExplicit = iota
    containedObjectTypeBlobReferences
    containedObjectTypeReferencedObject  // new
)
```

### metadata struct

New field alongside `Tags`:

```go
type metadata struct {
    Tags       ContainedObjects  // existing
    References ContainedObjects  // new
    Type       markl.Lock[Type, TypeMutable]
    // ...
}
```

### Metadata interface additions

```go
Metadata interface {
    // existing: GetTypeLock, GetTagLock, ...
    GetReferencedObjects() ContainedObjects
    GetReferencedObjectLock(SeqId) IdLock
    AllReferencedObjects() /* iterator */
}

MetadataMutable interface {
    // existing: GetTypeLockMutable, GetTagLockMutable, ...
    GetReferencedObjectLockMutable(SeqId) IdLockMutable
    AddReferencedObject(alias SeqId, objectId SeqId) // or similar
}
```

## Serialization

### Triple-hyphen format

Line prefix: `<` (inspired by shell input redirection).

Without alias:

    < one/uno@blake2b256-abc...

With alias:

    < blog-template = one/uno@blake2b256-abc...

With quoted alias (unsafe characters):

    < "alias with spaces" = one/uno@blake2b256-abc...

Lock value (`@signature`) is required in persistent output, optional in user
input (same rule as type locks).

### Inventory list box format

Inline with `<` sigil, no space after `<`:

    [one/dos @digest !md <one/uno@sig <blog-template=one/uno@sig]

With quoted alias:

    [one/dos @digest !md <"unsafe"=one/uno@sig]

### Binary index

Same key+null+format+id encoding as type and tag locks. The
`ContainedObjectType` byte distinguishes referenced objects from tags. The
`Alias` field is encoded when non-empty.

### JSON

Extend the `Lock` struct:

```go
type Lock struct {
    Type       string            // existing
    References map[string]string // objectId -> signature (or alias -> signature)
}
```

## Doddish Tokenizer

### New operator

Register `<` as `OpReference` in `doddish/op.go` with operator type
`operatorTypeMixedSeq` (like `!`, `@`, `%`).

### New token matchers

```go
// <ref@sig
TokenMatcherReferencedObject = TokensMatcher{
    TokenMatcherOp('<'),
    TokenTypeIdentifier,
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
}

// <alias=ref@sig
TokenMatcherReferencedObjectAlias = TokensMatcher{
    TokenMatcherOp('<'),
    TokenTypeIdentifier,
    TokenMatcherOp('='),
    TokenTypeIdentifier,
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
}
```

### Box format parser

Extend `readStringFormatBox()` switch in `hotel/box_format/read.go` with cases
for the new matchers, populating `metadata.References`.

### Box builder

Add `AddReferencedObjectsAndLocks()` to `echo/object_metadata_box_builder/` --
writes `<alias=ref@sig` fields for each referenced object.

## Lock Mechanics

### Finalizer

Add `writeReferencedObjectLockIfNecessary()` to
`hotel/object_finalizer/lockfile.go`, following the existing pattern:

1. For each referenced object in `metadata.References`, get its `IdLockMutable`
2. Skip if lock value is already set
3. Read the referenced object from the store by its fully qualified ID
4. Pin `lock.Value` to the referenced object's signature

### Commit options

Add `AllowReferencedObjectFailure bool` to `LockfileOptions` in
`golf/sku/commit_options.go`.

## Reference Discovery (Out of Scope)

Types define how to discover object references in blob content. The discovery
mechanism is not part of this design. Types are dynamic and user-defined, so the
implementation must afford flexibility. Possible approaches:

- **WASM guest modules** -- type provides a WASM binary that parses blob content
- **Shell out** -- type specifies an external program to extract references
- **Regexes** -- type declares patterns to match in blob content
- **Lua scripts** -- embedded scripting for reference extraction
- **Builtin parsers** -- "free" parsers for structured formats (JSON, TOML, etc.)

A separate FDR should cover the discovery interface.

## Sigil Design Etymology

| Sigil | Meaning | Inspiration |
|-------|---------|-------------|
| `!`   | Type    | Shebangs (`#!/bin/sh`) |
| `#`   | Description | Shebangs / comment syntax |
| `<`   | Referenced object | Shell input redirection (`< file`) |

## Rollback Strategy

This is additive. The `References` field on metadata defaults to empty.
Existing objects have no referenced object locks and behave identically.
If the feature proves wrong:

1. Stop populating `References` in the finalizer
2. Existing locked references remain inert in metadata (no behavioral impact)
3. A migration could strip them if needed, but is not required

No dual-architecture period is needed because the feature does not replace
existing infrastructure.

## Future Steps

1. Implement metadata storage and interface methods
2. Implement serialization across all four formats
3. Implement lock finalizer for referenced objects
4. Design and implement reference discovery interface (separate FDR)
5. Migration tests for stores created before referenced object locks
