# Referenced Object Locks Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add referenced object locks to the object metadata system so objects
can pin the signatures of other objects they reference, with optional alias
mapping.

**Architecture:** Reuse the existing `containedObject` struct with a new
`ContainedObjectType` value. Add a `References ContainedObjects` field to
metadata. Extend all four serialization formats (triple-hyphen, inventory list,
binary index, JSON). Add `<` operator to the doddish tokenizer.

**Tech Stack:** Go, doddish tokenizer, bats integration tests

**Rollback:** Purely additive. Remove the `References` field and related methods.
No migration needed since empty references are the default.

---

### Task 1: Data Model — ContainedObjectType and metadata field

**Files:**
- Modify: `go/internal/delta/objects/contained_object_type.go:11-14`
- Modify: `go/internal/delta/objects/main.go:19-36`

**Step 1: Add containedObjectTypeReferencedObject**

In `contained_object_type.go`, add the new const:

```go
const (
	containedObjectTypeMetadataExplicit containedObjectType = iota
	containedObjectTypeBlobReferences
	containedObjectTypeReferencedObject
)
```

**Step 2: Add References field to metadata struct**

In `main.go`, add `References` after `Tags` (line 24):

```go
type metadata struct {
	Description descriptions.Description
	Tags        ContainedObjects
	References  ContainedObjects  // new
	Type        markl.Lock[Type, TypeMutable]
	// ... rest unchanged
}
```

**Step 3: Run unit tests**

Run: `just test-go`
Expected: PASS (no behavior change yet)

**Step 4: Commit**

```
feat: add containedObjectTypeReferencedObject and References field
```

---

### Task 2: Metadata interface and method implementations

**Files:**
- Modify: `go/internal/delta/objects/interfaces.go:29-76`
- Modify: `go/internal/delta/objects/main.go` (add methods after line 235)

**Step 1: Add interface methods**

In `interfaces.go`, add to `Metadata` interface (after `GetTagLock`):

```go
GetReferencedObjects() ContainedObjects
GetReferencedObjectLock(SeqId) IdLock
AllReferencedObjects() interfaces.Seq[SeqId]
```

Add to `MetadataMutable` interface (after `GetTagLockMutable`):

```go
GetReferencedObjectLockMutable(SeqId) IdLockMutable
```

**Step 2: Implement methods on metadata**

In `main.go`, add implementations mirroring the tag methods:

```go
func (metadata *metadata) GetReferencedObjects() ContainedObjects {
	return metadata.References
}

func (metadata *metadata) AllReferencedObjects() interfaces.Seq[SeqId] {
	return func(yield func(SeqId) bool) {
		for ref := range metadata.References.All() {
			if !yield(ref) {
				return
			}
		}
	}
}

func (metadata *metadata) GetReferencedObjectLock(ref SeqId) IdLock {
	lock, _ := metadata.References.getLock(ref.String())
	return lock
}

func (metadata *metadata) GetReferencedObjectLockMutable(ref SeqId) IdLockMutable {
	lock, _ := metadata.References.getLockMutable(ref.String())
	return lock
}
```

**Step 3: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 4: Commit**

```
feat: add referenced object methods to Metadata interfaces
```

---

### Task 3: Reset, ResetWith, and Equals for References

**Files:**
- Modify: `go/internal/delta/objects/resetter.go:7-44`
- Modify: `go/internal/delta/objects/cmp.go:22-100`

**Step 1: Update Reset**

In `resetter.go` `Reset()` method, add after `metadata.ResetTags()` (line 13):

```go
metadata.References.Reset()
```

**Step 2: Update ResetWithExceptFields**

In `resetter.go` `ResetWithExceptFields()`, add after `dst.SetTagsFast(...)` (line 30).
First check if a `SetReferencesFast` method exists on ContainedObjects, otherwise
use the same pattern as tags. Most likely:

```go
dst.References.ResetWith(src.References)
```

If `ContainedObjects` doesn't have `ResetWith`, use the same approach as
`SetTagsFast`:

```go
dst.References = src.References
```

**Step 3: Update Equals**

In `cmp.go`, add References comparison after the Tags comparison block (after
line 89). Mirror the Tags pattern but using `metadata.References` and comparing
by key:

```go
// Compare References
aRefs := a.(*metadata).References
bRefs := b.(*metadata).References

if aRefs.Len() != bRefs.Len() {
	return false
}

for ref := range aRefs.All() {
	if _, ok := bRefs.getLock(ref.String()); !ok {
		return false
	}
}
```

Adapt to the exact patterns used for tag comparison (the tag comparison uses
`ContainsKey` -- use the equivalent for references).

**Step 4: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
feat: add Reset, ResetWith, and Equals support for References
```

---

### Task 4: Doddish operator and token matchers

**Files:**
- Modify: `go/internal/_/doddish/op.go:44-62` (add OpReference)
- Modify: `go/internal/_/doddish/op.go:85-115` (add to operatorTypeMixedSeq)
- Modify: `go/internal/_/doddish/token_matcher.go` (add new matchers)

**Step 1: Add OpReference operator**

In `op.go`, add after `OpDescription` (line 62):

```go
OpReference = Op('<')
```

**Step 2: Register operator type**

In `op.go` `GetType()` method, add `OpReference` to the `operatorTypeMixedSeq`
case (lines 103-107) alongside `OpMarklId`, `OpPathSeparator`, `OpType`,
`OpVirtual`.

**Step 3: Add token matchers**

In `token_matcher.go`, add after the last matcher:

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

**Step 4: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
feat: add < (OpReference) operator and token matchers to doddish
```

---

### Task 5: Triple-hyphen parser — read references

**Files:**
- Modify: `go/internal/foxtrot/object_metadata_fmt_triple_hyphen/text_parser2.go:54-76`

**Step 1: Add OpReference case to parser switch**

In `text_parser2.go`, add a case in the operator switch (after line 72, the
`OpExact` case):

```go
case doddish.OpReference:
	err = parser.readReference(metadata, remainder)
```

**Step 2: Implement readReference**

Add method to the parser. It needs to handle two forms:
- `one/uno@blake2b256-abc...` (no alias)
- `blog-template = one/uno@blake2b256-abc...` (with alias)

Parse by splitting on ` = ` first (alias detection), then split the object-ref
on `@` for the lock value. Use `markl.SetMarklIdWithFormatBlech32` for the
signature, same as type lock parsing.

**Step 3: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 4: Commit**

```
feat: parse referenced object locks in triple-hyphen format
```

---

### Task 6: Triple-hyphen formatter — write references

**Files:**
- Modify: `go/internal/foxtrot/object_metadata_fmt_triple_hyphen/formatter_components.go` (add writeReferencedObjects)
- Modify: `go/internal/foxtrot/object_metadata_fmt_triple_hyphen/factory.go:55-79` (add to formatter chains)

**Step 1: Add writeReferencedObjects method**

In `formatter_components.go`, add after `writeTypeAndSig` (line 177). Follow the
same pattern:

```go
func (factory formatterComponents) writeReferencedObjects(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	metadata := formatterContext.GetMetadata()

	for ref := range metadata.AllReferencedObjects() {
		lock := metadata.GetReferencedObjectLock(ref)

		if lock == nil || lock.GetValue().IsEmpty() {
			continue
		}

		// Get alias from the containedObject if present
		// Format: "< alias = ref@sig" or "< ref@sig"
		line := fmt.Sprintf("< %s@%s", ref, lock.GetValue())
		// TODO: handle alias case

		var n1 int64
		if n1, err = ohio.WriteLine(writer, line); err != nil {
			return n, err
		}
		n += n1
	}

	return n, err
}
```

Adapt to handle the alias. The `containedObject.Alias` field will need to be
accessible -- check how `ContainedObjects` exposes it.

**Step 2: Add to formatter chains**

In `factory.go`, add `formatterComponents.writeReferencedObjects` to the
`MetadataBlobPath`, `MetadataOnly`, and `MetadataInlineBlob` formatters. Place
after `writeCommonMetadataFormat` and before `writeComments`.

**Step 3: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 4: Commit**

```
feat: write referenced object locks in triple-hyphen format
```

---

### Task 7: Box format — parse references

**Files:**
- Modify: `go/internal/hotel/box_format/read.go:185-300` (add cases to switch)

**Step 1: Add token matcher cases**

In `read.go`, add cases in the switch statement. Place the alias matcher first
(longer match) before the non-alias matcher to avoid ambiguity:

```go
// <alias=ref@sig (referenced object with alias)
case seq.MatchAll(doddish.TokenMatcherReferencedObjectAlias...):
	// seq tokens: [<, alias, =, ref, @, sig]
	// Parse alias from seq[0:2], ref from seq[2:4], sig from seq[4:]
	// Add to metadata.References with alias

// <ref@sig (referenced object without alias)
case seq.MatchAll(doddish.TokenMatcherReferencedObject...):
	// seq tokens: [<, ref, @, sig]
	// Parse ref from seq[0:2], sig from seq[2:]
	// Add to metadata.References without alias
```

Use `markl.SetMarklIdWithFormatBlech32` for the signature value, same as the
type lock case at lines 229-236.

**Step 2: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 3: Commit**

```
feat: parse referenced object locks in box format
```

---

### Task 8: Box builder — write references

**Files:**
- Modify: `go/internal/echo/object_metadata_box_builder/main.go` (add method after line 179)
- Modify: `go/internal/hotel/box_format/transacted.go:274` (call new method)

**Step 1: Add AddReferencedObjectsAndLocks method**

In `object_metadata_box_builder/main.go`, add after `AddTagsAndLocks`:

```go
func (builder *Builder) AddReferencedObjectsAndLocks(metadata objects.MetadataMutable) {
	for ref := range metadata.AllReferencedObjects() {
		lock := metadata.GetReferencedObjectLock(ref)

		if lock == nil {
			continue
		}

		// Format key: "<ref" or "<alias=ref"
		// TODO: handle alias prefix
		key := "<" + ref.String()
		value := lock.GetValue()

		if value.IsEmpty() {
			builder.Contents.Append(string_format_writer.Field{
				Value: key,
			})
		} else {
			builder.addMarklIdLockWithColorType(
				key,
				value,
				string_format_writer.ColorTypeId, // or appropriate color
			)
		}
	}
}
```

**Step 2: Wire into transacted encoder**

In `transacted.go`, add call after `builder.AddTags(metadata)` (line 274):

```go
builder.AddReferencedObjectsAndLocks(metadata)
```

**Step 3: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 4: Commit**

```
feat: write referenced object locks in box format
```

---

### Task 9: Binary index — key_bytes constant and encoder

**Files:**
- Modify: `go/internal/_/key_bytes/main.go:13-39` (add References constant)
- Modify: `go/internal/india/stream_index/binary_field.go:14-24` (add to binaryFieldOrder)
- Modify: `go/internal/india/stream_index/binary_encoder.go` (add case after Type)

**Step 1: Add key_bytes constant**

In `key_bytes/main.go`, add:

```go
References = Binary('R')
```

Choose a character not already used. Verify `'R'` is available by checking the
existing constants.

**Step 2: Add to binaryFieldOrder**

In `binary_field.go`, add `key_bytes.References` after `key_bytes.Type` (line 23):

```go
var binaryFieldOrder = []key_bytes.Binary{
	key_bytes.Sigil,
	key_bytes.ObjectId,
	key_bytes.Blob,
	key_bytes.RepoPubKey,
	key_bytes.RepoSig,
	key_bytes.Description,
	key_bytes.Tag,
	key_bytes.Tai,
	key_bytes.Type,
	key_bytes.References,  // new
	key_bytes.SigParentMetadataParentObjectId,
	// ...
}
```

**Step 3: Add encoder case**

In `binary_encoder.go`, add a case in `writeFieldKey()` after the `key_bytes.Type`
case. Follow the Tag encoding pattern (lines 159-178) since references are also
a collection of locks:

```go
case key_bytes.References:
	for ref := range metadata.AllReferencedObjects() {
		refLock := object.GetMetadataMutable().GetReferencedObjectLockMutable(ref)
		if refLock == nil || refLock.GetValue().IsNull() {
			continue
		}

		binaryMarshaler := markl.MakeMutableLockCoder(refLock, true)

		if n, err = encoder.writeFieldBinaryMarshaler(binaryMarshaler); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}
```

Adapt to handle the Alias field in the binary encoding. The alias needs to be
encoded alongside the lock key+value.

**Step 4: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
feat: add References to binary stream index encoder
```

---

### Task 10: Binary index — decoder

**Files:**
- Modify: `go/internal/india/stream_index/binary_decoder.go` (add case after Type)

**Step 1: Add decoder case**

In `binary_decoder.go`, add a case in `readFieldKey()` after the `key_bytes.Type`
case. Follow the Tag decoding pattern:

```go
case key_bytes.References:
	marshaler := markl.MakeMutableLockCoderValueRequired(
		metadata.GetReferencedObjectLockMutable(/* need to construct SeqId from decoded key */),
	)

	if err = marshaler.UnmarshalBinary(decoder.Content.Bytes()); err != nil {
		err = errors.Wrap(err)
		return err
	}
```

The tricky part: the decoder needs to first read the lock key (object ID) from
the binary data, then look up or create the corresponding lock entry in
`metadata.References`. Study how the Tag decoder creates tag entries during
decode to mirror this.

**Step 2: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 3: Commit**

```
feat: add References to binary stream index decoder
```

---

### Task 11: JSON format

**Files:**
- Modify: `go/internal/hotel/sku_json_fmt/lock.go:3-5`
- Modify: `go/internal/hotel/sku_json_fmt/main.go:79-81` (populate from metadata)
- Modify: `go/internal/hotel/sku_json_fmt/main.go:185-192` (populate metadata from JSON)

**Step 1: Extend Lock struct**

In `lock.go`:

```go
type Lock struct {
	Type       string            `json:"type,omitempty"`
	References map[string]string `json:"references,omitempty"`
}
```

**Step 2: Populate Lock.References from metadata**

In `main.go` `FromObjectIdStringAndMetadata()`, after the type lock population
(line 81), add reference population:

```go
json.Lock.References = make(map[string]string)
for ref := range metadata.AllReferencedObjects() {
	lock := metadata.GetReferencedObjectLock(ref)
	if lock != nil && !lock.GetValue().IsEmpty() {
		json.Lock.References[ref.String()] = lock.GetValue().String()
	}
}
if len(json.Lock.References) == 0 {
	json.Lock.References = nil
}
```

**Step 3: Read Lock.References back into metadata**

In `main.go` `ToTransacted()`, after the type lock reading (line 192), add:

```go
for refId, sig := range json.Lock.References {
	// Create SeqId from refId, set lock value to sig
	// Add to metadata.References
}
```

**Step 4: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
feat: add References to JSON lock format
```

---

### Task 12: Lock finalizer

**Files:**
- Modify: `go/internal/hotel/object_finalizer/lockfile.go` (add writeReferencedObjectLockIfNecessary)
- Modify: `go/internal/hotel/object_finalizer/main.go:141-194` (call in WriteLockfile)
- Modify: `go/internal/golf/sku/commit_options.go:37-40` (add AllowReferencedObjectFailure)

**Step 1: Add AllowReferencedObjectFailure**

In `commit_options.go`:

```go
type LockfileOptions struct {
	AllowTypeFailure              bool
	AllowTagFailure               bool
	AllowReferencedObjectFailure  bool
}
```

Update the store option getters that set `AllowTagFailure: true` to also set
`AllowReferencedObjectFailure: true` (lines 42-105).

**Step 2: Add writeReferencedObjectLockIfNecessary**

In `lockfile.go`, add after `writeTagLockIfNecessary` (line 73):

```go
func (finalizer finalizer) writeReferencedObjectLockIfNecessary(
	metadata objects.MetadataMutable,
	ref ids.SeqId,
	funcs ...sku.FuncReadOne,
) (err error) {
	if ref.IsEmpty() {
		err = ErrEmptyLockKey
		return err
	}

	refLock := metadata.GetReferencedObjectLockMutable(ref)

	if !refLock.GetValue().IsNull() {
		return err
	}

	refObject, repool := sku.GetTransactedPool().GetWithRepool()
	defer repool()

	if ok := sku.ReadOneObjectIdBespoke(ref, refObject, funcs...); ok {
		refLock.GetValueMutable().ResetWithMarklId(refObject.GetMetadataMutable().GetObjectSig())
	} else {
		err = ErrFailedToReadCurrentLockObject
		return err
	}

	return err
}
```

**Step 3: Call in WriteLockfile**

In `main.go`, add after the tag lock loop (after line 191):

```go
for ref := range metadata.AllReferencedObjects() {
	if err = finalizer.writeReferencedObjectLockIfNecessary(
		metadata,
		ref,
		funcs...,
	); err != nil {
		switch err {
		case ErrEmptyLockKey:
			err = nil

		case ErrFailedToReadCurrentLockObject:
			if options.AllowReferencedObjectFailure {
				err = nil
				break
			}

			fallthrough

		default:
			err = errors.Wrapf(err, "failed to write referenced object lock for: %q", ref)
			return err
		}
	}
}
```

**Step 4: Run unit tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
feat: add referenced object lock finalizer
```

---

### Task 13: Integration tests

**Files:**
- Modify: `zz-tests_bats/current_version/show.bats` (add reference lock assertions)

**Step 1: Write bats test for referenced object lock output**

Add a test that creates a zettel with a referenced object, commits it, and
verifies the lock appears in `show` output. The test must:

1. Initialize a store with `run_dodder_init_disable_age`
2. Create a referenced zettel (the target)
3. Create a zettel that references the target (with alias)
4. Run `dodder show` and assert the `<` reference lock appears in output

Since reference discovery is out of scope, this test will need to add references
through whatever metadata API is available (possibly through the triple-hyphen
format by writing a file with `< ref@sig` lines and checking it in).

**Step 2: Run the specific test**

Run: `just test-bats-targets show.bats`
Expected: New test passes

**Step 3: Commit**

```
test: add integration test for referenced object locks
```

---

### Task 14: Existing test verification

**Step 1: Run full test suite**

Run: `just test`
Expected: All existing tests pass. No regressions from the new References field
(it defaults to empty, so existing serialization should be unaffected).

**Step 2: Fix any regressions**

If binary format tests fail, the new `key_bytes.References` in
`binaryFieldOrder` may cause issues with existing fixtures. Verify that the
encoder/decoder skip empty references gracefully (no output for empty
collections).

**Step 3: Commit any fixes**

```
fix: ensure empty References field does not affect existing serialization
```
