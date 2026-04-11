# Typed Blob References Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add required type locks to blob references across all serialization
formats, with hard failure at commit time when type information is missing.

**Architecture:** Extend `blobReferenceEntry` with a `TypeLock` field, add
doddish token matchers for typed blob reference grammars, update all 4
serialization paths (hyphence, box, binary, inventory list), and add commit-time
validation in the object finalizer.

**Tech Stack:** Go, doddish scanner/token matcher, markl.Lock, binary
encoder/decoder

**Rollback:** N/A --- hard cutover, no backward compatibility with untyped blob
references.

--------------------------------------------------------------------------------

### Task 1: Add doddish token matchers for typed blob references

**Files:**

- Modify: `go/internal/0/doddish/token_matcher.go:73-86`

**Step 1: Add typed blob reference matchers**

In `go/internal/0/doddish/token_matcher.go`, replace the existing untyped blob
reference matchers and add typed variants. The old `TokenMatcherBlobReference`
and `TokenMatcherBlobReferenceAlias` are replaced --- hard cutover, no untyped
blob refs survive.

``` go
// <@digest !type@sig
TokenMatcherTypedBlobReference = TokensMatcher{
    TokenMatcherOp('<'),
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
    TokenMatcherOp('!'),
    TokenTypeIdentifier,
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
}

// alias<@digest !type@sig
TokenMatcherTypedBlobReferenceAlias = TokensMatcher{
    TokenTypeIdentifier,
    TokenMatcherOp('<'),
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
    TokenMatcherOp('!'),
    TokenTypeIdentifier,
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
}

// <@digest !type (user input, no sig pinned yet)
TokenMatcherTypedBlobReferenceNoSig = TokensMatcher{
    TokenMatcherOp('<'),
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
    TokenMatcherOp('!'),
    TokenTypeIdentifier,
}

// alias<@digest !type (user input, no sig pinned yet)
TokenMatcherTypedBlobReferenceAliasNoSig = TokensMatcher{
    TokenTypeIdentifier,
    TokenMatcherOp('<'),
    TokenMatcherOp('@'),
    TokenTypeIdentifier,
    TokenMatcherOp('!'),
    TokenTypeIdentifier,
}
```

Keep the old `TokenMatcherBlobReference` and `TokenMatcherBlobReferenceAlias`
names but update their definitions to the typed versions (with sig). Add the
no-sig variants as new names.

**Step 2: Run unit tests**

Run: `cd go && go test -v -tags test,debug ./internal/0/doddish/` Expected:
existing tests pass (no existing tests use the blob reference matchers).

**Step 3: Commit**

    feat(doddish): add typed blob reference token matchers

--------------------------------------------------------------------------------

### Task 2: Add doddish tests for typed blob reference grammars

**Files:**

- Modify: `go/internal/0/doddish/seq_test.go`

**Step 1: Add test cases for typed blob reference matching**

Add to `getSeqTestCases()` in `go/internal/0/doddish/seq_test.go`:

``` go
// typed blob ref without alias: <@digest !type@sig
{
    input: "<@blake2b256-abc123 !tree@ed25519-sig456",
    expected: [][]TokenMatcher{
        {
            TokenMatcherOp('<'),
            TokenMatcherOp('@'),
            TokenTypeIdentifier,
            TokenMatcherOp('!'),
            TokenTypeIdentifier,
            TokenMatcherOp('@'),
            TokenTypeIdentifier,
        },
    },
},
// typed blob ref with alias: alias<@digest !type@sig
{
    input: "hero<@blake2b256-abc123 !image-png@ed25519-sig456",
    expected: [][]TokenMatcher{
        {
            TokenTypeIdentifier,
            TokenMatcherOp('<'),
            TokenMatcherOp('@'),
            TokenTypeIdentifier,
        },
        {
            TokenMatcherOp(' '),
        },
        {
            TokenMatcherOp('!'),
            TokenTypeIdentifier,
            TokenMatcherOp('@'),
            TokenTypeIdentifier,
        },
    },
},
// typed blob ref no sig: <@digest !type
{
    input: "<@blake2b256-abc123 !tree",
    expected: [][]TokenMatcher{
        {
            TokenMatcherOp('<'),
            TokenMatcherOp('@'),
            TokenTypeIdentifier,
        },
        {
            TokenMatcherOp(' '),
        },
        {
            TokenMatcherOp('!'),
            TokenTypeIdentifier,
        },
    },
},
```

Note: the scanner splits on spaces, so `<@digest !type@sig` produces multiple
seqs when scanned with `ScanDotAllowedInIdentifiers`. The box parser uses
`Scan()` which handles this differently. Study the scanner behavior carefully
when writing tests --- verify by running and adjusting.

Also add test cases to `getScannerTestCases()` in `scanner_test.go` to validate
token-level scanning of typed blob reference inputs.

**Step 2: Run tests**

Run: `cd go && go test -v -tags test,debug ./internal/0/doddish/` Expected: PASS

**Step 3: Commit**

    test(doddish): add typed blob reference grammar tests

--------------------------------------------------------------------------------

### Task 3: Add TypeLock field to blobReferenceEntry and update collection API

**Files:**

- Modify: `go/internal/delta/objects/blob_reference.go`
- Modify: `go/internal/delta/objects/interfaces.go:49-50,88-90`
- Modify: `go/internal/delta/objects/main.go:294-312`

**Step 1: Update the struct and collection**

In `go/internal/delta/objects/blob_reference.go`:

``` go
type blobReferenceEntry struct {
    Key      markl.Id
    TypeLock markl.Lock[ids.SeqId, *ids.SeqId]
    Alias    string
}
```

Update `Add()` signature --- the type lock is required:

``` go
func (refs *BlobReferences) Add(id markl.Id, typeLock markl.Lock[ids.SeqId, *ids.SeqId]) {
```

Add a method to get the type lock for a blob ref:

``` go
func (refs BlobReferences) GetTypeLock(id markl.Id) markl.Lock[ids.SeqId, *ids.SeqId] {
```

Add a mutable getter:

``` go
func (refs *BlobReferences) GetTypeLockMutable(id markl.Id) *markl.Lock[ids.SeqId, *ids.SeqId] {
```

Update `ResetWith` to clone the type lock:

``` go
clone.TypeLock.ResetWith(entry.TypeLock)
```

**Step 2: Update the Metadata interface**

In `go/internal/delta/objects/interfaces.go`, update `Metadata`:

``` go
AllBlobReferences() interfaces.Seq[markl.Id]
GetBlobReferenceAlias(markl.Id) string
GetBlobReferenceTypeLock(markl.Id) markl.Lock[ids.SeqId, *ids.SeqId]
```

Update `MetadataMutable`:

``` go
AddBlobReference(markl.Id, markl.Lock[ids.SeqId, *ids.SeqId])
SetBlobReferenceAlias(markl.Id, string) error
GetBlobReferenceTypeLockMutable(markl.Id) *markl.Lock[ids.SeqId, *ids.SeqId]
ResetBlobReferences()
```

**Step 3: Update metadata implementations**

In `go/internal/delta/objects/main.go`, update `AddBlobReference`,
`GetBlobReferenceTypeLock`, `GetBlobReferenceTypeLockMutable` to delegate to
`BlobRefs`.

**Step 4: Fix all callers of AddBlobReference**

These callers currently pass only `markl.Id` --- they all need the type lock
parameter. File list (from grep):

- `go/internal/hotel/box_format/read.go:275,291`
- `go/internal/india/stream_index/binary_decoder.go:408`
- `go/internal/papa/store/reference_discovery.go:154`
- `go/internal/foxtrot/object_metadata_fmt_hyphence/text_parser2.go:166`

For now, pass `markl.Lock[ids.SeqId, *ids.SeqId]{}` (empty lock) at each call
site as a placeholder --- Tasks 4-7 will fill in proper values. The important
thing is the code compiles.

**Step 5: Build**

Run: `cd go && go build ./...` Expected: compiles successfully

**Step 6: Run unit tests**

Run: `cd go && go test -v -tags test,debug ./internal/delta/objects/...`
Expected: PASS

**Step 7: Commit**

    feat(objects): add TypeLock field to blobReferenceEntry

--------------------------------------------------------------------------------

### Task 4: Update hyphence parser to use doddish for typed blob references

**Files:**

- Modify:
  `go/internal/foxtrot/object_metadata_fmt_hyphence/text_parser2.go:120-176`

**Step 1: Rewrite readBlobReference to use doddish scanner**

Replace the string-based `isBlobReference` and `readBlobReference` with
doddish-based parsing. The blob reference line (after stripping the `-` prefix)
looks like:

- `@digest !type@sig` (no alias)
- `@digest !type` (no alias, no sig --- user input)
- `alias < @digest !type@sig` (with alias)
- `alias < @digest !type` (with alias, no sig)

Use `doddish.ScanExactlyOneSeqWithDotAllowedInIdenfierFromString` or the scanner
to tokenize, then match against the typed blob reference token matchers.

For the type lock, use `markl.MakeMutableLockCoderValueNotRequired` (same
pattern as `readType` at line 191) so the parser accepts both `!type` and
`!type@sig`.

**Step 2: Build and test**

Run: `cd go && go build ./...` Run:
`cd go && go test -v -tags test,debug ./internal/foxtrot/object_metadata_fmt_hyphence/...`
Expected: PASS

**Step 3: Commit**

    feat(hyphence): parse typed blob references with doddish scanner

--------------------------------------------------------------------------------

### Task 5: Update hyphence formatter to write typed blob references

**Files:**

- Modify:
  `go/internal/foxtrot/object_metadata_fmt_hyphence/formatter_components.go:220-249`

**Step 1: Update writeBlobReferences**

Update the `writeBlobReferences` function to include the type lock after the
digest. The formatter reads the type lock from metadata and appends `!type@sig`
(or just `!type` if no sig is pinned yet).

Access the type lock via the new `GetBlobReferenceTypeLock(blobId)` method on
metadata.

Format pattern:

``` go
// with alias:
line = fmt.Sprintf("- %s < @%s %s", alias, blobId, typeLockStr)
// without alias:
line = fmt.Sprintf("- @%s %s", blobId, typeLockStr)
```

Where `typeLockStr` is constructed from the type lock's key and value (e.g.,
`!tree@sig` or `!tree`).

**Step 2: Build and test**

Run: `cd go && go build ./...` Expected: compiles

**Step 3: Commit**

    feat(hyphence): write typed blob references with type lock

--------------------------------------------------------------------------------

### Task 6: Update box parser for typed blob references

**Files:**

- Modify: `go/internal/hotel/box_format/read.go:267-291`

**Step 1: Update blob reference cases in readStringFormatBox**

Replace the existing `TokenMatcherBlobReferenceAlias` and
`TokenMatcherBlobReference` cases with the typed variants. The box scanner
produces seqs differently from the hyphence line scanner --- within `[...]` the
scanner splits on spaces, so `<@digest` and `!type@sig` may be separate seqs.

Study the existing pattern at lines 238-265 (referenced object with alias) to
understand how the box parser extracts lock values from seqs. The type lock
extraction follows the same `markl.SetMarklIdWithFormatBlech32` pattern used for
type locks at lines 219-236.

Handle both with-sig and without-sig variants:

- `<@digest !type@sig` --- full lock
- `<@digest !type` --- key only, no value
- `alias<@digest !type@sig` --- with alias and full lock
- `alias<@digest !type` --- with alias, key only

The box parser currently handles blob refs as single seqs (`<@digest` =
`[< @ ident]`). With type locks, the blob ref and type lock will be separate
seqs split by space. This means the parser needs to look ahead after matching a
blob reference seq to check for a following type lock seq.

Alternatively, use `MatchStart` to match the blob ref prefix, then consume
remaining tokens for the type lock.

**Step 2: Build and test**

Run: `cd go && go build ./...` Expected: compiles

**Step 3: Commit**

    feat(box): parse typed blob references with type lock

--------------------------------------------------------------------------------

### Task 7: Update box builder for typed blob references

**Files:**

- Modify: `go/internal/echo/object_metadata_box_builder/main.go:265-282`

**Step 1: Update AddBlobReferences**

The current `AddBlobReferences` method builds `<@digest` or `alias<@digest`
strings. Update it to include the type lock in the output using `<(...)`
grouping:

``` go
func (builder *Builder) AddBlobReferences(metadata objects.MetadataMutable) {
    for blobId := range metadata.AllBlobReferences() {
        alias := metadata.GetBlobReferenceAlias(blobId)
        typeLock := metadata.GetBlobReferenceTypeLock(blobId)

        var typeStr string
        if !typeLock.GetValue().IsEmpty() {
            typeStr = fmt.Sprintf(" %s@%s", typeLock.GetKey(), typeLock.GetValue())
        } else if !typeLock.GetKey().IsEmpty() {
            typeStr = fmt.Sprintf(" %s", typeLock.GetKey())
        }

        var value string
        if alias != "" {
            value = fmt.Sprintf("%s<(@%s%s)", alias, blobId, typeStr)
        } else {
            value = fmt.Sprintf("<(@%s%s)", blobId, typeStr)
        }

        builder.Contents.Append(string_format_writer.Field{
            Value:      value,
            ColorType:  string_format_writer.ColorTypeId,
            NoTruncate: true,
        })
    }
}
```

**Step 2: Build**

Run: `cd go && go build ./...` Expected: compiles

**Step 3: Commit**

    feat(box-builder): render typed blob references with type lock

--------------------------------------------------------------------------------

### Task 8: Update binary encoder for typed blob references

**Files:**

- Modify: `go/internal/india/stream_index/binary_encoder.go:231-268`

**Step 1: Update the BlobReferences case**

In the `BlobReferences` case of `writeFieldKey`, after writing the blob ID
bytes, write the type lock bytes with a uint16 length prefix before the alias:

1.  uint16 length + blob ID bytes (existing)
2.  uint16 length + type lock binary bytes (new)
3.  Alias string (existing, remaining bytes)

Use `markl.MakeMutableLockCoderValueRequired` to marshal the type lock to
binary, same pattern as referenced object locks at lines 220-228.

**Step 2: Build**

Run: `cd go && go build ./...` Expected: compiles

**Step 3: Commit**

    feat(binary): encode type lock in blob reference entries

--------------------------------------------------------------------------------

### Task 9: Update binary decoder for typed blob references

**Files:**

- Modify: `go/internal/india/stream_index/binary_decoder.go:371-418`

**Step 1: Update the BlobReferences case**

After reading the blob ID bytes, read the type lock bytes:

1.  Read uint16 + blob ID bytes (existing)
2.  Read uint16 + type lock bytes (new) --- unmarshal with
    `markl.MakeMutableLockCoderValueRequired`
3.  Read remaining bytes as alias (existing, adjusted offset)

Pass the type lock to `metadata.AddBlobReference(blobId, typeLock)`.

**Step 2: Build**

Run: `cd go && go build ./...` Expected: compiles

**Step 3: Commit**

    feat(binary): decode type lock from blob reference entries

--------------------------------------------------------------------------------

### Task 10: Add commit-time validation for blob reference type locks

**Files:**

- Modify: `go/internal/hotel/object_finalizer/errors.go:20-24`
- Modify: `go/internal/hotel/object_finalizer/lockfile.go`
- Modify: `go/internal/hotel/object_finalizer/main.go:125-219`

**Step 1: Add error sentinel**

In `go/internal/hotel/object_finalizer/errors.go`, add:

``` go
ErrBlobReferenceMissingType = newPkgError("blob reference missing type")
```

**Step 2: Add writeBlobReferenceTypeLockIfNecessary**

In `go/internal/hotel/object_finalizer/lockfile.go`, add a new method that
mirrors `writeTypeLockIfNecessary` but operates on a blob reference's type lock.
It reads the type object by the type lock's key and pins the signature:

``` go
func (finalizer finalizer) writeBlobReferenceTypeLockIfNecessary(
    metadata objects.MetadataMutable,
    blobId markl.Id,
    funcs ...sku.FuncReadOne,
) (err error) {
```

If `TypeLock.Key` is empty, return `ErrBlobReferenceMissingType`. If
`TypeLock.Value` is already set, return nil (already pinned). Otherwise, read
the type object and pin its signature.

**Step 3: Call from WriteLockfile**

In `go/internal/hotel/object_finalizer/main.go`, add a loop after the referenced
objects loop (around line 216):

``` go
for blobId := range metadata.AllBlobReferences() {
    if err = finalizer.writeBlobReferenceTypeLockIfNecessary(
        metadata,
        blobId,
        funcs...,
    ); err != nil {
        switch err {
        case ErrBlobReferenceMissingType:
            err = errors.Wrapf(err, "blob reference: %s", blobId)
            return err

        case ErrFailedToReadCurrentLockObject:
            err = errors.Wrapf(err, "failed to pin type lock for blob reference: %s", blobId)
            return err

        default:
            err = errors.Wrap(err)
            return err
        }
    }
}
```

Note: `ErrBlobReferenceMissingType` is a hard fail --- no option to allow it.

**Step 4: Build**

Run: `cd go && go build ./...` Expected: compiles

**Step 5: Commit**

    feat(finalizer): hard fail on blob references missing type lock

--------------------------------------------------------------------------------

### Task 11: Update reference discovery to pass type lock

**Files:**

- Modify: `go/internal/papa/store/reference_discovery.go:143-160`

**Step 1: Update the blob reference creation path**

The reference discovery script currently emits blob references. The FDR
specifies that lines starting with `@` followed by `!type@sig` are typed blob
references:

    @blake2b256-xxxx... !tree@sig
    alias = @blake2b256-xxxx... !type@sig

Update the parser in `reference_discovery.go` to extract the type lock from the
discovery output and pass it to `AddBlobReference(blobId, typeLock)`.

If the discovery script emits a blob ref without type info, this is an error in
the type definition's `[references]` config --- fail with a descriptive message.

**Step 2: Build and test**

Run: `cd go && go build ./...` Expected: compiles

**Step 3: Commit**

    feat(reference-discovery): pass type lock for discovered blob references

--------------------------------------------------------------------------------

### Task 12: Build, run full test suite, fix remaining issues

**Step 1: Build**

Run: `cd go && go build ./...` Expected: compiles

**Step 2: Run unit tests**

Run: `cd go && go test -v -tags test,debug ./...` Expected: PASS (or identify
failures to fix)

**Step 3: Run integration tests**

Run: `just test-bats` Expected: PASS (or identify fixture/assertion updates
needed)

**Step 4: Fix any failures**

If BATS tests fail due to changed output format (blob references now include
type locks), update assertions. Do NOT regenerate fixtures unless the store
version was bumped.

**Step 5: Commit**

    fix: resolve test failures from typed blob reference cutover
