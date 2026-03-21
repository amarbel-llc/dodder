# Typed Blob References --- Data Model + Serialization

**Date:** 2026-03-21 **FDR:** [FDR-0001: Object
Locks](../features/0001-object-locks.md), Phase 3 (partial) **Scope:** Data
model, token matchers, parsing, formatting, binary encoding/decoding, hard fail
on missing type. Excludes recursive `expandEdges`, GC reachability, and
reference discovery output format.

## Responsibility Split

- **User** provides type on blob references when editing (e.g.,
  `- < @blake2b256-abc... !tree`)
- **Commit/finalizer** pins the signature (e.g., `!tree` → `!tree@sig`)
- **Hard fail** at commit if any blob reference has no type (`TypeLock.Key` is
  empty)

## Data Model

`blobReferenceEntry` gains a required `TypeLock` field:

``` go
type blobReferenceEntry struct {
    Key      markl.Id
    TypeLock markl.Lock[ids.SeqId, *ids.SeqId]
    Alias    string
}
```

`Add()` requires the type lock. `All()` / new entry-level iterator exposes the
type lock. `ResetWith` clones the type lock field.

## Token Matchers

New matchers in `_/doddish/token_matcher.go`:

``` go
// <@digest !type@sig
TokenMatcherTypedBlobReference

// alias<@digest !type@sig
TokenMatcherTypedBlobReferenceAlias

// <@digest !type (user input, no sig)
TokenMatcherTypedBlobReferenceNoSig

// alias<@digest !type (user input, no sig)
TokenMatcherTypedBlobReferenceAliasNoSig
```

## Doddish Tests

Tests in `_/doddish/token_matcher_test.go` validate:

- Typed matchers match `<@blake2b256-abc... !tree@sig`
- Alias variants match `hero<@blake2b256-abc... !tree@sig`
- No-sig variants match `<@blake2b256-abc... !tree`
- Old untyped matchers don't match typed inputs
- Both hyphence (line-by-line) and box (within `[...]`) grammars produce correct
  token sequences

## Hyphence Format

**Text syntax:**

    - < @blake2b256-abc... !tree@sig
    - hero-image < @blake2b256-def... !image-png@sig

**Parser** (`foxtrot/object_metadata_fmt_hyphence/text_parser2.go`): Replace
string-based `isBlobReference` / `readBlobReference` with doddish scanner. Use
`markl.MakeMutableLockCoderValueNotRequired` for the type lock (accepts `!type`
without sig from user input).

**Formatter** (`foxtrot/object_metadata_fmt_hyphence/formatter_components.go`):
`writeBlobReferences` appends `!type@sig` after the digest.

## Box Format

**Text syntax:**

    <(@blake2b256-abc... !tree@sig) hero-image<(@blake2b256-def... !image-png@sig)

**Parser** (`hotel/box_format/read.go`): Replace `TokenMatcherBlobReference` /
`TokenMatcherBlobReferenceAlias` cases with typed variants. Extract type lock
from the seq.

**Builder** (`echo/object_metadata_box_builder/main.go`): Render as
`<(@digest !type@sig)` and `alias<(@digest !type@sig)`.

## Binary Index

Key byte: `BlobReferences = Binary('b')` (unchanged).

Per entry:

1.  uint16 length + blob ID bytes (markl.Id binary encoding)
2.  uint16 length + type lock bytes (type ID + signature)
3.  Alias (remaining bytes, implicit length)

## Hard Fail on Missing Type

New error type in `hotel/object_finalizer/` alongside `IsTypeLockError`. At
commit, iterate blob references; if any has empty `TypeLock.Key`, fail with
descriptive error identifying which blob reference is missing type information.

## Files Touched

1.  `_/doddish/token_matcher.go` --- 4 new matchers
2.  `_/doddish/token_matcher_test.go` --- grammar validation tests
3.  `delta/objects/blob_reference.go` --- TypeLock field, API changes
4.  `foxtrot/object_metadata_fmt_hyphence/text_parser2.go` --- doddish-based
    parsing
5.  `foxtrot/object_metadata_fmt_hyphence/formatter_components.go` --- write
    type lock
6.  `hotel/box_format/read.go` --- typed blob ref matching
7.  `echo/object_metadata_box_builder/main.go` --- typed blob ref rendering
8.  `india/stream_index/binary_encoder.go` --- encode type lock
9.  `india/stream_index/binary_decoder.go` --- decode type lock
10. `hotel/object_finalizer/` --- hard fail on missing type
