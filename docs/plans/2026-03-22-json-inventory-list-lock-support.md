# JSON Inventory List Lock Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add tag locks, blob references (with type locks and aliases), and
reference aliases to the JSON inventory list format so all lock kinds round-trip
through JSON.

**Architecture:** Extend the `Lock` struct in `sku_json_fmt/lock.go` with tag
lock and blob reference fields. Update `FromObjectIdStringAndMetadata` (encode)
and `ToTransacted` (decode) in `sku_json_fmt/main.go` to serialize/deserialize
the new fields. Test with a unit-level JSON round-trip that populates all lock
kinds on a `sku.Transacted`, encodes to JSON, decodes back, and asserts
equality.

**Tech Stack:** Go `testing`, `encoding/json`, `sku_json_fmt`, `sku.Transacted`

**Rollback:** N/A --- purely additive.

--------------------------------------------------------------------------------

### Task 1: Round-Trip Test for All Lock Kinds

Write a comprehensive round-trip test that exercises every lock field. This test
will fail initially (tag locks, blob references, and reference aliases are not
serialized), then guide the implementation in Tasks 2-4.

**Promotion criteria:** N/A

**Files:** - Create: `go/internal/hotel/sku_json_fmt/main_test.go`

**Step 1: Write the failing test**

``` go
//go:build test

package sku_json_fmt

import (
    "encoding/json"
    "testing"

    "code.linenisgreat.com/dodder/go/internal/bravo/ids"
    "code.linenisgreat.com/dodder/go/internal/bravo/markl"
    "code.linenisgreat.com/dodder/go/internal/golf/sku"
    "code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestTransactedRoundTripAllLockKinds(t1 *testing.T) {
    t := ui.T{T: t1}

    // Build an object with every lock kind populated
    original := sku.GetTransactedPool().Get()
    defer sku.GetTransactedPool().Put(original)

    metadata := original.GetMetadataMutable()

    // Object ID
    if err := original.GetObjectIdMutable().Set("one/uno"); err != nil {
        t.Fatal(err)
    }

    // Type + type lock
    if err := metadata.GetTypeMutable().SetType("ref-blob"); err != nil {
        t.Fatal(err)
    }
    if err := metadata.GetTypeLockMutable().GetValueMutable().Set(
        "ed25519_sig-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaas",
    ); err != nil {
        t.Fatal(err)
    }

    // Tags + tag locks
    if err := metadata.AddTagString("project-alpha"); err != nil {
        t.Fatal(err)
    }
    {
        var tag ids.TagStruct
        if err := tag.Set("project-alpha"); err != nil {
            t.Fatal(err)
        }
        tagLock := metadata.GetTagLockMutable(tag)
        if err := tagLock.GetValueMutable().Set(
            "ed25519_sig-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbs",
        ); err != nil {
            t.Fatal(err)
        }
    }

    // Referenced object + lock + alias
    {
        var refId ids.SeqId
        if err := refId.Set("two/dos"); err != nil {
            t.Fatal(err)
        }
        if err := metadata.AddReference(refId); err != nil {
            t.Fatal(err)
        }
        refLock := metadata.GetReferencedObjectLockMutable(refId)
        if err := refLock.GetValueMutable().Set(
            "ed25519_sig-ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccs",
        ); err != nil {
            t.Fatal(err)
        }
        if err := metadata.SetReferenceAlias(refId, "blog-template"); err != nil {
            t.Fatal(err)
        }
    }

    // Blob reference + type lock + alias
    {
        var blobId markl.Id
        if err := blobId.Set(
            "blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd",
        ); err != nil {
            t.Fatal(err)
        }

        var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
        if err := typeLock.GetKeyMutable().SetType("img"); err != nil {
            t.Fatal(err)
        }
        if err := typeLock.GetValueMutable().Set(
            "ed25519_sig-ddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddds",
        ); err != nil {
            t.Fatal(err)
        }

        metadata.AddBlobReference(blobId, typeLock)
        if err := metadata.SetBlobReferenceAlias(blobId, "hero-image"); err != nil {
            t.Fatal(err)
        }
    }

    // Encode
    var jsonObj Transacted
    if err := jsonObj.FromTransacted(original, nil); err != nil {
        t.Fatal(err)
    }

    // Verify JSON fields are populated
    encoded, err := json.MarshalIndent(jsonObj, "", "  ")
    if err != nil {
        t.Fatal(err)
    }
    t.Logf("JSON:\n%s", encoded)

    // Decode into fresh object
    decoded := sku.GetTransactedPool().Get()
    defer sku.GetTransactedPool().Put(decoded)

    if err := jsonObj.ToTransacted(decoded, nil); err != nil {
        t.Fatal(err)
    }

    decodedMeta := decoded.GetMetadataMutable()

    // Verify type lock
    t.AssertEqualStrings(
        metadata.GetTypeLock().GetValue().String(),
        decodedMeta.GetTypeLock().GetValue().String(),
    )

    // Verify tag lock
    {
        var tag ids.TagStruct
        if err := tag.Set("project-alpha"); err != nil {
            t.Fatal(err)
        }
        originalLock := metadata.GetTagLock(tag)
        decodedLock := decodedMeta.GetTagLock(tag)
        if originalLock == nil || decodedLock == nil {
            t.Fatal("tag lock missing after round-trip")
        }
        t.AssertEqualStrings(
            originalLock.GetValue().String(),
            decodedLock.GetValue().String(),
        )
    }

    // Verify referenced object lock + alias
    {
        var refId ids.SeqId
        if err := refId.Set("two/dos"); err != nil {
            t.Fatal(err)
        }
        originalLock := metadata.GetReferencedObjectLock(refId)
        decodedLock := decodedMeta.GetReferencedObjectLock(refId)
        if originalLock == nil || decodedLock == nil {
            t.Fatal("referenced object lock missing after round-trip")
        }
        t.AssertEqualStrings(
            originalLock.GetValue().String(),
            decodedLock.GetValue().String(),
        )
        t.AssertEqualStrings(
            metadata.GetReferenceAlias(refId),
            decodedMeta.GetReferenceAlias(refId),
        )
    }

    // Verify blob reference + type lock + alias
    {
        var blobId markl.Id
        if err := blobId.Set(
            "blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd",
        ); err != nil {
            t.Fatal(err)
        }

        originalTypeLock := metadata.GetBlobReferenceTypeLock(blobId)
        decodedTypeLock := decodedMeta.GetBlobReferenceTypeLock(blobId)
        if originalTypeLock.IsEmpty() || decodedTypeLock.IsEmpty() {
            t.Fatal("blob reference type lock missing after round-trip")
        }
        t.AssertEqualStrings(
            originalTypeLock.GetKey().String(),
            decodedTypeLock.GetKey().String(),
        )
        t.AssertEqualStrings(
            originalTypeLock.GetValue().String(),
            decodedTypeLock.GetValue().String(),
        )
        t.AssertEqualStrings(
            metadata.GetBlobReferenceAlias(blobId),
            decodedMeta.GetBlobReferenceAlias(blobId),
        )
    }
}
```

**Step 2: Run test to verify it fails**

Run:
`cd go && go test -v -tags test,debug -run TestTransactedRoundTripAllLockKinds ./internal/hotel/sku_json_fmt/`

Expected: FAIL --- tag lock, blob reference, and reference alias assertions fail
because those fields are not serialized.

**Step 3: Commit**

    test: add JSON round-trip test for all lock kinds (#44)

--------------------------------------------------------------------------------

### Task 2: Add Tag Locks to JSON Format

**Promotion criteria:** N/A

**Files:** - Modify: `go/internal/hotel/sku_json_fmt/lock.go` - Modify:
`go/internal/hotel/sku_json_fmt/main.go:79-91` (encode) - Modify:
`go/internal/hotel/sku_json_fmt/main.go:195-221` (decode)

**Step 1: Add `Tags` field to `Lock` struct**

In `lock.go`:

``` go
type Lock struct {
    Type       string            `json:"type,omitempty"`
    Tags       map[string]string `json:"tags,omitempty"`
    References map[string]string `json:"references,omitempty"`
}
```

**Step 2: Encode tag locks in `FromObjectIdStringAndMetadata`**

In `main.go`, after the existing `Lock` initialization (around line 79-91), add
tag lock encoding. Insert after the referenced objects loop:

``` go
for tag := range metadata.AllTags() {
    tagLock := metadata.GetTagLock(tag)
    if tagLock != nil && !tagLock.GetValue().IsEmpty() {
        if json.Lock.Tags == nil {
            json.Lock.Tags = make(map[string]string)
        }
        json.Lock.Tags[tag.String()] = tagLock.GetValue().String()
    }
}
```

**Step 3: Decode tag locks in `ToTransacted`**

In `main.go`, after the existing type lock decode block (around line 195-202),
add tag lock decoding:

``` go
for tagStr, sigStr := range json.Lock.Tags {
    var tag ids.TagStruct
    if err = tag.Set(tagStr); err != nil {
        err = errors.Wrap(err)
        return err
    }

    tagLock := metadata.GetTagLockMutable(tag)
    if err = tagLock.GetValueMutable().Set(sigStr); err != nil {
        err = errors.Wrap(err)
        return err
    }
}
```

**Step 4: Run test to verify tag lock assertions pass**

Run:
`cd go && go test -v -tags test,debug -run TestTransactedRoundTripAllLockKinds ./internal/hotel/sku_json_fmt/`

Expected: Tag lock assertions pass. Blob reference and reference alias
assertions still fail.

**Step 5: Commit**

    feat: add tag lock serialization to JSON inventory list format (#44)

--------------------------------------------------------------------------------

### Task 3: Add Blob References to JSON Format

**Promotion criteria:** N/A

**Files:** - Modify: `go/internal/hotel/sku_json_fmt/main.go` (Transacted
struct, encode, decode)

**Step 1: Add `BlobReference` type and field to `Transacted`**

Add a new struct and field. In `main.go`, add after the `Transacted` struct
definition:

``` go
type BlobReference struct {
    Digest   string `json:"digest"`
    Type     string `json:"type,omitempty"`
    TypeLock string `json:"type-lock,omitempty"`
    Alias    string `json:"alias,omitempty"`
}
```

Add to the `Transacted` struct:

``` go
BlobReferences []BlobReference `json:"blob-references,omitempty"`
```

**Step 2: Encode blob references in `FromObjectIdStringAndMetadata`**

After the tag locks loop, add:

``` go
for blobId := range metadata.AllBlobReferences() {
    ref := BlobReference{
        Digest: blobId.String(),
        Alias:  metadata.GetBlobReferenceAlias(blobId),
    }

    typeLock := metadata.GetBlobReferenceTypeLock(blobId)
    if !typeLock.IsEmpty() {
        ref.Type = typeLock.GetKey().String()
        if !typeLock.GetValue().IsEmpty() {
            ref.TypeLock = typeLock.GetValue().String()
        }
    }

    json.BlobReferences = append(json.BlobReferences, ref)
}
```

**Step 3: Decode blob references in `ToTransacted`**

After the referenced objects decode loop, add:

``` go
for _, ref := range json.BlobReferences {
    var blobId markl.Id
    if err = blobId.Set(ref.Digest); err != nil {
        err = errors.Wrapf(err, "invalid blob reference digest: %q", ref.Digest)
        return err
    }

    var typeLock markl.Lock[ids.SeqId, *ids.SeqId]

    if ref.Type != "" {
        if err = typeLock.GetKeyMutable().SetType(ref.Type); err != nil {
            err = errors.Wrapf(err, "invalid blob reference type: %q", ref.Type)
            return err
        }
    }

    if ref.TypeLock != "" {
        if err = typeLock.GetValueMutable().Set(ref.TypeLock); err != nil {
            err = errors.Wrapf(err, "invalid blob reference type lock: %q", ref.TypeLock)
            return err
        }
    }

    metadata.AddBlobReference(blobId, typeLock)

    if ref.Alias != "" {
        if err = metadata.SetBlobReferenceAlias(blobId, ref.Alias); err != nil {
            err = errors.Wrap(err)
            return err
        }
    }
}
```

**Step 4: Run test to verify blob reference assertions pass**

Run:
`cd go && go test -v -tags test,debug -run TestTransactedRoundTripAllLockKinds ./internal/hotel/sku_json_fmt/`

Expected: Blob reference assertions pass. Reference alias assertion still fails.

**Step 5: Commit**

    feat: add blob reference serialization to JSON inventory list format (#44)

--------------------------------------------------------------------------------

### Task 4: Add Reference Aliases to JSON Format

**Promotion criteria:** N/A

**Files:** - Modify: `go/internal/hotel/sku_json_fmt/lock.go` - Modify:
`go/internal/hotel/sku_json_fmt/main.go` (encode + decode)

**Step 1: Add `ReferenceAliases` field to `Lock` struct**

In `lock.go`:

``` go
type Lock struct {
    Type             string            `json:"type,omitempty"`
    Tags             map[string]string `json:"tags,omitempty"`
    References       map[string]string `json:"references,omitempty"`
    ReferenceAliases map[string]string `json:"reference-aliases,omitempty"`
}
```

**Step 2: Encode reference aliases in `FromObjectIdStringAndMetadata`**

In the existing referenced objects loop (around line 83-91), add alias
collection:

``` go
for ref := range metadata.AllReferencedObjects() {
    lock := metadata.GetReferencedObjectLock(ref)
    if lock != nil && !lock.GetValue().IsEmpty() {
        if json.Lock.References == nil {
            json.Lock.References = make(map[string]string)
        }
        json.Lock.References[ref.String()] = lock.GetValue().String()
    }

    alias := metadata.GetReferenceAlias(ref)
    if alias != "" {
        if json.Lock.ReferenceAliases == nil {
            json.Lock.ReferenceAliases = make(map[string]string)
        }
        json.Lock.ReferenceAliases[ref.String()] = alias
    }
}
```

**Step 3: Decode reference aliases in `ToTransacted`**

After the existing referenced objects decode loop, add:

``` go
for refIdStr, alias := range json.Lock.ReferenceAliases {
    var refId ids.SeqId
    if err = refId.Set(refIdStr); err != nil {
        err = errors.Wrap(err)
        return err
    }

    if err = metadata.SetReferenceAlias(refId, alias); err != nil {
        err = errors.Wrap(err)
        return err
    }
}
```

**Step 4: Run test --- all assertions pass**

Run:
`cd go && go test -v -tags test,debug -run TestTransactedRoundTripAllLockKinds ./internal/hotel/sku_json_fmt/`

Expected: PASS --- all lock kinds round-trip through JSON.

**Step 5: Commit**

    feat: add reference alias serialization to JSON inventory list format (#44)

--------------------------------------------------------------------------------

### Task 5: Integration Test --- JSON Inventory List Round-Trip via BATS

Verify the full encode→decode pipeline through the `inventory_list_coders`
`jsonV0` coder, not just the `sku_json_fmt` struct.

**Promotion criteria:** N/A

**Files:** - Modify: `zz-tests_bats/current_version/show.bats`

**Step 1: Write the BATS test**

Add a test that creates an object with all lock kinds, exports to JSON inventory
list format, then imports and verifies all fields survive:

``` bash
# bats test_tags=user_story:referenced_objects
function json_inventory_list_preserves_all_lock_kinds { # @test
    run_dodder init-workspace
    assert_success

    # Create a type for blob references
    cat >img.type <<-'TYPEFILE'
        ---
        ! toml-type-v1
        ---

        file-extension = 'png'
    TYPEFILE

    run_dodder checkin -delete img.type
    assert_success

    # Create a zettel with blob reference
    run_dodder new -edit=false - <<-'EOM'
        ---
        # json lock test
        - hero-image < @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !img
        ! md
        ---

        content
    EOM
    assert_success

    # Show in JSON format — verify blob reference fields present
    run_dodder show -format json two/uno:
    assert_success
    assert_output --partial '"blob-references"'
    assert_output --partial '"hero-image"'
    assert_output --partial '"digest"'
}
```

**Step 2: Run test**

Run: `just test-bats-targets current_version/show.bats`

**Step 3: Commit**

    test: add BATS test for JSON inventory list lock round-trip (#44)

--------------------------------------------------------------------------------

### Task 6: Run Full Test Suite and Close Issue

**Step 1: Run full suite**

Run: `just test`

Expected: all tests pass.

**Step 2: Close #44**

Comment noting all lock kinds now round-trip through JSON: type locks, tag
locks, referenced object locks + aliases, blob references + type locks +
aliases.

**Step 3: Commit FDR update**

Move #44 to closed in `docs/features/0001-object-locks.md`.

    docs: close #44 in FDR-0001 — JSON inventory list lock support complete
