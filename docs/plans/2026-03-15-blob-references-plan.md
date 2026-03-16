# Blob References Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add blob reference support to dodder metadata, unifying object and blob reference discovery under a single `[references]` type TOML section.

**Architecture:** New `BlobReferences` collection on metadata with `markl.Id` keys (separate from `ContainedObjects` which uses `SeqId` keys). Reference discovery parser dispatches on `@` prefix to distinguish blob refs from object refs. Box format alias syntax changes from `<alias=ref@sig` to `alias<ref@sig` for both object and blob references.

**Tech Stack:** Go, doddish tokenizer, existing `markl.Id` types.

**Rollback:** Purely additive — revert the commits. `[object-references]` has not shipped.

---

### Task 1: BlobReferences data model on metadata

**Promotion criteria:** N/A

**Files:**
- Create: `go/internal/delta/objects/blob_reference.go`
- Modify: `go/internal/delta/objects/main.go`
- Modify: `go/internal/delta/objects/interfaces.go`
- Modify: `go/internal/delta/objects/contained_object_type.go`

**Step 1: Create blob reference entry type**

Create `go/internal/delta/objects/blob_reference.go`:

```go
package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
)

type blobReferenceEntry struct {
	Key   markl.Id
	Alias string
}

type BlobReferences collections_slice.Slice[blobReferenceEntry]

func (refs BlobReferences) GetSlice() collections_slice.Slice[blobReferenceEntry] {
	return collections_slice.Slice[blobReferenceEntry](refs)
}

func (refs *BlobReferences) GetSliceMutable() *collections_slice.Slice[blobReferenceEntry] {
	return (*collections_slice.Slice[blobReferenceEntry])(refs)
}

func (refs BlobReferences) All() interfaces.Seq[markl.Id] {
	return func(yield func(markl.Id) bool) {
		for entry := range refs.GetSlice().All() {
			if !yield(entry.Key) {
				return
			}
		}
	}
}

func (refs *BlobReferences) Add(id markl.Id) error {
	for _, entry := range refs.GetSlice().All() {
		if markl.EqualIds(entry.Key, id) {
			return nil
		}
	}

	refs.GetSliceMutable().Append(blobReferenceEntry{Key: id})

	refs.GetSliceMutable().SortWithComparer(
		func(left, right blobReferenceEntry) cmp.Result {
			return cmp.CompareUTF8(
				cmp.ComparableString(left.Key.String()),
				cmp.ComparableString(right.Key.String()),
				false,
			)
		},
	)

	return nil
}

func (refs *BlobReferences) SetAlias(id markl.Id, alias string) error {
	for i := range *refs {
		if markl.EqualIds((*refs)[i].Key, id) {
			(*refs)[i].Alias = alias
			return nil
		}
	}

	return nil
}

func (refs BlobReferences) GetAlias(id markl.Id) string {
	for _, entry := range refs {
		if markl.EqualIds(entry.Key, id) {
			return entry.Alias
		}
	}

	return ""
}

func (refs *BlobReferences) Reset() {
	*refs = (*refs)[:0]
}
```

Note: Check if `markl.EqualIds` exists. If not, compare via
`left.String() == right.String()` or `bytes.Equal(left.GetBytes(), right.GetBytes())`.
Adjust as needed based on what `markl.Id` exposes.

**Step 2: Add BlobReferences field to metadata**

In `go/internal/delta/objects/main.go`, add a `BlobRefs BlobReferences` field to
the `metadata` struct (after `Contents ContainedObjects`).

Add methods on `metadata`:

```go
func (metadata *metadata) AllBlobReferences() interfaces.Seq[markl.Id] {
	return metadata.BlobRefs.All()
}

func (metadata *metadata) AddBlobReference(id markl.Id) error {
	return metadata.BlobRefs.Add(id)
}

func (metadata *metadata) SetBlobReferenceAlias(id markl.Id, alias string) error {
	return metadata.BlobRefs.SetAlias(id, alias)
}

func (metadata *metadata) GetBlobReferenceAlias(id markl.Id) string {
	return metadata.BlobRefs.GetAlias(id)
}
```

**Step 3: Add to interfaces**

In `go/internal/delta/objects/interfaces.go`, add to `Metadata`:

```go
AllBlobReferences() interfaces.Seq[markl.Id]
GetBlobReferenceAlias(markl.Id) string
```

Add to `MetadataMutable`:

```go
AddBlobReference(markl.Id) error
SetBlobReferenceAlias(markl.Id, string) error
```

Add import for `markl` package if not already present.

**Step 4: Add IsBlobReference to contained_object_type**

In `go/internal/delta/objects/contained_object_type.go`, add:

```go
func (t containedObjectType) IsBlobReference() bool { return t == containedObjectTypeBlobReferences }
```

**Step 5: Build and verify**

Run: `just build`
Expected: Compiles cleanly.

**Step 6: Commit**

```
feat: add BlobReferences data model to metadata

New BlobReferences collection with markl.Id keys, separate from
ContainedObjects. Methods: AllBlobReferences, AddBlobReference,
SetBlobReferenceAlias, GetBlobReferenceAlias.
```

---

### Task 2: Rename [object-references] to [references] in type TOML

**Promotion criteria:** N/A — `[object-references]` has not shipped to consumers.

**Files:**
- Modify: `go/internal/hotel/type_blobs/toml_v1.go`
- Modify: `go/internal/hotel/type_blobs/main.go`
- Modify: `go/internal/hotel/type_blobs/object_references_config.go` (rename file)

**Step 1: Rename TOML tag and field**

In `go/internal/hotel/type_blobs/toml_v1.go`, change the field name and TOML tag:

```go
// Before:
ObjectReferences *ObjectReferencesConfig `toml:"object-references,omitempty"`

// After:
References *ObjectReferencesConfig `toml:"references,omitempty"`
```

Update the getter method:

```go
// Before:
func (blob TomlV1) GetObjectReferences() *ObjectReferencesConfig {
	return blob.ObjectReferences
}

// After:
func (blob TomlV1) GetReferences() *ObjectReferencesConfig {
	return blob.References
}
```

**Step 2: Rename interface**

In `go/internal/hotel/type_blobs/main.go`, rename the interface:

```go
// Before:
type WithObjectReferences interface {
	GetObjectReferences() *ObjectReferencesConfig
}

// After:
type WithReferences interface {
	GetReferences() *ObjectReferencesConfig
}
```

Update `Blob` interface to embed `WithReferences` instead of `WithObjectReferences`.

**Step 3: Update TomlV0**

In `go/internal/hotel/type_blobs/toml_v0.go`, rename the method:

```go
func (blob TomlV0) GetReferences() *ObjectReferencesConfig {
	return nil
}
```

**Step 4: Optionally rename config file**

Rename `object_references_config.go` to `references_config.go` (the type name
`ObjectReferencesConfig` can stay or be renamed to `ReferencesConfig` — follow
the codebase convention for renaming).

**Step 5: Update callers**

Search for all callers of `GetObjectReferences()` and update to
`GetReferences()`. Key locations:

- `go/internal/papa/store/reference_discovery.go` (line ~87)
- Any other callers found via grep

**Step 6: Update integration test type TOML**

In `zz-tests_bats/current_version/show.bats`, update all `[object-references]`
sections in type TOML heredocs to `[references]`.

**Step 7: Build and run tests**

Run: `just build && just test`
Expected: All tests pass.

**Step 8: Commit**

```
refactor: rename [object-references] to [references] in type TOML

Unified section name for both object and blob reference discovery.
[object-references] was never shipped to consumers, no migration needed.
```

---

### Task 3: Update parseReferenceOutput for blob references

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/papa/store/reference_discovery.go`

**Step 1: Update discoveredReference struct**

Add a `IsBlobRef bool` field to `discoveredReference`:

```go
type discoveredReference struct {
	Alias    string
	ObjectId string
	BlobId   string  // populated when line value starts with @
}
```

**Step 2: Update parseReferenceOutput**

After extracting the value (either the full line or after `" = "`), check for
`@` prefix:

```go
if strings.HasPrefix(value, "@") {
	ref.BlobId = value[1:] // strip @
} else {
	ref.ObjectId = value
}
```

**Step 3: Update discoverReferences to handle blob refs**

After parsing, dispatch on the ref type. Where the existing code does
`metadata.AddReference(refId)`, add an else branch:

```go
if ref.BlobId != "" {
	var blobId markl.Id
	if err = blobId.Set(ref.BlobId); err != nil {
		// handle error
		continue
	}
	metadata.AddBlobReference(blobId)
	if ref.Alias != "" {
		metadata.SetBlobReferenceAlias(blobId, ref.Alias)
	}
} else {
	// existing object reference logic
}
```

**Step 4: Build**

Run: `just build`
Expected: Compiles cleanly.

**Step 5: Commit**

```
feat: extend reference discovery parser for blob references

Lines starting with @ (or with @ after alias =) are parsed as blob
references. Dispatches to AddBlobReference on metadata.
```

---

### Task 4: Change box format alias syntax to alias<ref@sig

This task changes the alias format for BOTH object references (existing) and
blob references (new).

**Promotion criteria:** N/A — existing format has not shipped.

**Files:**
- Modify: `go/internal/_/doddish/token_matcher.go`
- Modify: `go/internal/hotel/box_format/read.go`
- Modify: `go/internal/echo/object_metadata_box_builder/main.go`

**Step 1: Update token matchers**

In `go/internal/_/doddish/token_matcher.go`, change the alias matcher and add
blob ref matchers:

```go
// alias<ref@sig (referenced object with alias)
TokenMatcherReferencedObjectAlias = TokensMatcher{
	TokenTypeIdentifier,
	TokenMatcherOp('<'),
	TokenTypeIdentifier,
	TokenMatcherOp('@'),
	TokenTypeIdentifier,
}

// <@digest (blob reference without alias)
TokenMatcherBlobReference = TokensMatcher{
	TokenMatcherOp('<'),
	TokenMatcherOp('@'),
	TokenTypeIdentifier,
}

// alias<@digest (blob reference with alias)
TokenMatcherBlobReferenceAlias = TokensMatcher{
	TokenTypeIdentifier,
	TokenMatcherOp('<'),
	TokenMatcherOp('@'),
	TokenTypeIdentifier,
}
```

Also update the existing `TokenMatcherReferencedObject` (no alias) — it stays
the same: `<`, identifier, `@`, identifier.

**Step 2: Update box format reader**

In `go/internal/hotel/box_format/read.go`, update the case for
`TokenMatcherReferencedObjectAlias` to use the new token positions:

- Old: `<`, alias(1), `=`, ref(3), `@`, sig(5) → `seq.At(1)`, `seq.At(3)`
- New: alias(0), `<`, ref(2), `@`, sig(4) → `seq.At(0)`, `seq.At(2)`

Add new cases for `TokenMatcherBlobReference` and `TokenMatcherBlobReferenceAlias`:

For `TokenMatcherBlobReferenceAlias` (alias<@digest):
- alias = `seq.At(0)`, digest = `seq.At(3)` (after `<`, `@`)
- Call `metadata.AddBlobReference(blobId)` and `SetBlobReferenceAlias`

For `TokenMatcherBlobReference` (<@digest):
- digest = `seq.At(2)` (after `<`, `@`)
- Call `metadata.AddBlobReference(blobId)`

**Important:** Order the cases from longest match first:
1. `TokenMatcherBlobReferenceAlias` (4 tokens)
2. `TokenMatcherReferencedObjectAlias` (5 tokens)
3. `TokenMatcherBlobReference` (3 tokens)
4. `TokenMatcherReferencedObject` (4 tokens)

Actually, check whether `MatchAll` is greedy or exact. If exact, order doesn't
matter. If prefix-matching, put longer matchers first. Verify behavior.

**Step 3: Update box format writer**

In `go/internal/echo/object_metadata_box_builder/main.go`, update
`AddReferencedObjectsAndLocks` to use the new alias format:

```go
// Before:
key = "<" + alias + "=" + ref.String()

// After:
key = alias + "<" + ref.String()
```

Add a new method `AddBlobReferences`:

```go
func (builder *Builder) AddBlobReferences(metadata objects.MetadataMutable) {
	for blobId := range metadata.AllBlobReferences() {
		alias := metadata.GetBlobReferenceAlias(blobId)

		var key string
		if alias != "" {
			key = alias + "<@" + blobId.String()
		} else {
			key = "<@" + blobId.String()
		}

		builder.Contents.Append(string_format_writer.Field{
			Value:     key,
			ColorType: string_format_writer.ColorTypeId,
		})
	}
}
```

**Step 4: Wire AddBlobReferences in the box transacted formatter**

Find where `AddReferencedObjectsAndLocks` is called (in
`go/internal/hotel/box_format/transacted.go` around line 280) and add a call to
`AddBlobReferences` after it.

**Step 5: Build and run tests**

Run: `just build && just test`
Expected: Tests may fail if existing tests assert the old `<alias=ref` format.
Fix assertions to use `alias<ref` format.

**Step 6: Commit**

```
feat: change box alias format to alias<ref, add blob reference display

Replace <alias=ref@sig with alias<ref@sig for object references.
Add <@digest and alias<@digest for blob references.
```

---

### Task 5: Add blob references to hyphence format

**Files:**
- Modify: `go/internal/foxtrot/object_metadata_fmt_hyphence/formatter_components.go`

**Step 1: Add writeBlobReferences method**

In `formatter_components.go`, add after `writeReferencedObjects`:

```go
func (factory formatterComponents) writeBlobReferences(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	metadata := formatterContext.GetMetadata()

	for blobId := range metadata.AllBlobReferences() {
		var line string

		alias := metadata.GetBlobReferenceAlias(blobId)

		if alias != "" {
			if strings.ContainsAny(alias, " \t\"") {
				alias = fmt.Sprintf("%q", alias)
			}

			line = fmt.Sprintf("- %s < @%s", alias, blobId)
		} else {
			line = fmt.Sprintf("- @%s", blobId)
		}

		var n1 int64
		if n1, err = ohio.WriteLine(writer, line); err != nil {
			return n, err
		}
		n += n1
	}

	return n, err
}
```

**Step 2: Wire into the format pipeline**

Find where `writeReferencedObjects` is called in the formatter and add
`writeBlobReferences` after it. Look in the same file or in
`formatter_family.go`/`formatter.go` for the call site.

**Step 3: Add hyphence parsing for blob references**

Find where `- ref@sig` lines are parsed in the hyphence parser. Add handling for
`- @digest` and `- alias < @digest` patterns. When the reference value starts
with `@`, call `AddBlobReference` instead of `AddReference`.

**Step 4: Build and test**

Run: `just build && just test`
Expected: All tests pass.

**Step 5: Commit**

```
feat: add blob reference serialization to hyphence format

Writes - @digest and - alias < @digest for blob references.
Parses @ prefix to distinguish blob from object references.
```

---

### Task 6: Integration tests

**Files:**
- Modify: `zz-tests_bats/current_version/show.bats`

**Step 1: Add test for blob reference discovery**

Add a test that creates a type with `[references]` that discovers both object
and blob references, creates a zettel with content containing both patterns,
and verifies the output includes both object and blob references.

Example test:

```bash
function show_zettel_with_blob_references { # @test
  # Create a type that discovers blob references
  cat >ref-blob.type <<-'TYPEFILE'
	[references]
	shell = ['bash', '-c']
	script = "grep -oP '@[a-z0-9-]+'"
  TYPEFILE

  run_dodder add -delete ref-blob.type
  assert_success

  run_dodder new -edit=false - <<-EOM
	---
	# zettel with blob ref
	! ref-blob
	---
	See blob @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd for details.
  EOM
  assert_success

  run_dodder show -format hyphence :z
  assert_success
  assert_output --partial "@blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd"
}
```

Adjust the grep pattern and assertions based on how dodder actually renders blob
references. The key property: the blob digest appears in the metadata output
prefixed with `@`.

**Step 2: Add test for alias<ref format change**

Add or update a test verifying the new `alias<ref@sig` format for object
references in box output.

**Step 3: Run full test suite**

Run: `just test`
Expected: All tests pass.

**Step 4: Commit**

```
test: add integration tests for blob references and alias format
```

---

### Task 7: Update FDR-0001

**Files:**
- Modify: `docs/features/0001-object-locks.md`

**Step 1: Move blob references from future to implemented**

Update the "Future: blob references" section to reflect implementation status.
Update the serialization table with the new alias format.

**Step 2: Commit**

```
docs: update FDR-0001 with blob references implementation status
```
