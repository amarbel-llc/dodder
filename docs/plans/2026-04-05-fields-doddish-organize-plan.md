# Fields in Doddish and Organize Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add type-defined fields to dodder: WIT-aligned field kinds, TomlV2
type blob with `[[fields]]` declarations, script-based field reader/writer,
binary codec persistence, doddish query syntax (`=` and `^=`), and organize
format integration. First concrete field: `status` on `!task`.

**Architecture:** Fields are cached projections of values extracted from blob
content by the type system. The type blob TOML declares field schemas
(`[[fields]]`) and scripts (`[fields-reader]`/`[fields-writer]`) that read/write
field values from/to blobs. Field values are cached in the binary stream index
for query performance. Organize displays and parses fields via existing box
format; mutation runs a full compile (fields -\> blob -\> new digest).

**Tech Stack:** Go, TOML (tommy codegen), BATS integration tests, doddish
scanner/query engine.

**Rollback:** Revert type blobs to `toml-type-v1` header; old store versions
ignore unknown `key_bytes.Field` entries. Types without `[[fields]]` are
unaffected.

**Design doc:** `docs/plans/2026-04-05-fields-doddish-organize-design.md`

--------------------------------------------------------------------------------

### Task 1: Add `fields.Kind` WIT-aligned type enum

Add WIT-aligned Kind constants to the fields package. All six are defined; only
KindString and KindEnum are used in v1.

**Promotion criteria:** N/A (new code)

**Files:**

- Modify: `go/internal/_/fields/main.go`

**Step 1: Add Kind type and constants**

Add after the existing `Type` definition:

``` go
type Kind byte

const (
    KindString     Kind = iota // WIT: string
    KindEnum                   // WIT: enum
    KindBool                   // WIT: bool
    KindU32                    // WIT: u32
    KindS32                    // WIT: s32
    KindListString             // WIT: list<string>
)
```

**Step 2: Run `just build` to verify compilation**

Run: `just build` Expected: compiles without errors

**Step 3: Commit**

Message: `feat(fields): add WIT-aligned Kind type enum`

--------------------------------------------------------------------------------

### Task 2: Add `fields.Definition` struct

Add the field schema struct to the fields package.

**Promotion criteria:** N/A (new code)

**Files:**

- Modify: `go/internal/_/fields/main.go`

**Step 1: Add Definition struct**

Add after the Kind constants:

``` go
type Definition struct {
    Name    string
    Kind    Kind
    Values  []string // populated for KindEnum
    Default string
}
```

**Step 2: Run `just build` to verify compilation**

Run: `just build` Expected: compiles without errors

**Step 3: Commit**

Message: `feat(fields): add Definition struct for type-defined field schemas`

--------------------------------------------------------------------------------

### Task 3: Add `TypeBlobDigest` to `fields.Field`

Add the lookup hint to Field so type-defined fields can lazily resolve their
Definition via the type blob store.

**Promotion criteria:** N/A (additive)

**Files:**

- Modify: `go/internal/_/fields/main.go`
- Modify: any files that construct `Field` literals (box_format, hyphence
  parsers) --- add zero-value `TypeBlobDigest` if needed by compiler

**Step 1: Add import and field**

Add `markl.Id` field to the Field struct. The import path for markl.Id needs to
be determined from the codebase --- search for `markl.Id` usage in delta/objects
to find the correct import.

``` go
type Field struct {
    Type
    Key, Value     string
    TypeBlobDigest markl.Id // lookup hint for lazy definition resolution
}
```

**Step 2: Fix any compilation errors from existing Field literal constructors**

Existing code constructs `Field{Key: ..., Value: ..., Type: ...}` --- these
should continue to compile since `TypeBlobDigest` has a zero value. If any
constructors use positional initialization, convert them to named fields.

**Step 3: Run `just build` to verify compilation**

Run: `just build` Expected: compiles without errors

**Step 4: Commit**

Message: `feat(fields): add TypeBlobDigest lookup hint to Field`

--------------------------------------------------------------------------------

### Task 4: Add `TomlV2` type blob with `[[fields]]` support

Create the new type blob version with field declarations and reader/writer
scripts.

**Promotion criteria:** N/A (TomlV0 and TomlV1 remain decodable)

**Files:**

- Create: `go/internal/golf/type_blobs/toml_v2.go`
- Create: `go/internal/golf/type_blobs/field_definition.go`
- Modify: `go/internal/golf/type_blobs/main.go` (update Blob interface if
  needed, add WithFields sub-interface)
- Modify: `go/internal/bravo/ids/types_builtin.go` (add TypeTomlTypeV2 constant)
- Modify: `go/internal/hotel/type_blobs/coding.go` (register TomlV2 coder)

**Step 1: Create `field_definition.go` with FieldDefinition struct**

This is the TOML-tagged struct for `[[fields]]` array entries. It mirrors
`fields.Definition` but with TOML tags and tommy codegen.

``` go
package type_blobs

import "code.linenisgreat.com/dodder/go/internal/_/fields"

//go:generate tommy generate
type FieldDefinition struct {
    Name    string   `toml:"name"`
    Kind    string   `toml:"kind"`
    Values  []string `toml:"values,omitempty"`
    Default string   `toml:"default,omitempty"`
}

func (fd *FieldDefinition) ToDefinition() fields.Definition {
    return fields.Definition{
        Name:    fd.Name,
        Kind:    fields.KindFromString(fd.Kind),
        Values:  fd.Values,
        Default: fd.Default,
    }
}
```

This requires adding a `KindFromString` function to `_/fields/main.go`:

``` go
func KindFromString(s string) Kind {
    switch s {
    case "string":
        return KindString
    case "enum":
        return KindEnum
    case "bool":
        return KindBool
    case "u32":
        return KindU32
    case "s32":
        return KindS32
    case "list<string>":
        return KindListString
    default:
        return KindString
    }
}
```

**Step 2: Create `toml_v2.go`**

``` go
package type_blobs

import (
    "code.linenisgreat.com/dodder/go/lib/_/reset"
    "code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

//go:generate tommy generate
type TomlV2 struct {
    Binary        bool                                      `toml:"binary,omitempty"`
    FileExtension string                                    `toml:"file-extension,omitempty"`
    MimeType      string                                    `toml:"mime-type,omitempty"`
    ExecCommand   *script_config.ScriptConfig               `toml:"exec-command,omitempty"`
    VimSyntaxType string                                    `toml:"vim-syntax-type"`
    UTIGroups     map[string]UTIGroup                       `toml:"uti-groups"`
    Formatters    map[string]script_config.WithOutputFormat `toml:"formatters,omitempty"`

    Hooks      string            `toml:"hooks"`
    References *ReferencesConfig `toml:"references,omitempty"`

    Fields       []FieldDefinition          `toml:"fields,omitempty"`
    FieldsReader *script_config.ScriptConfig `toml:"fields-reader,omitempty"`
    FieldsWriter *script_config.ScriptConfig `toml:"fields-writer,omitempty"`
}
```

Add all the same getter methods as TomlV1 (GetBinary, GetFileExtension, etc.)
plus new methods:

``` go
func (blob *TomlV2) GetFieldDefinitions() []FieldDefinition {
    return blob.Fields
}

func (blob *TomlV2) GetFieldsReader() *script_config.ScriptConfig {
    return blob.FieldsReader
}

func (blob *TomlV2) GetFieldsWriter() *script_config.ScriptConfig {
    return blob.FieldsWriter
}
```

**Step 3: Add `WithFields` interface to `main.go`**

``` go
type WithFields interface {
    GetFieldDefinitions() []FieldDefinition
    GetFieldsReader() *script_config.ScriptConfig
    GetFieldsWriter() *script_config.ScriptConfig
}
```

**Step 4: Add TypeTomlTypeV2 constant**

In `go/internal/bravo/ids/types_builtin.go`, add:

``` go
TypeTomlTypeV2       = "!toml-type-v2"
TypeTomlTypeVCurrent = TypeTomlTypeV2  // update from V1
```

Register in `init()` alongside the others.

**Step 5: Register TomlV2 coder**

In `go/internal/hotel/type_blobs/coding.go`, add a third entry to the
`CoderToTypedBlob` map for `ids.TypeTomlTypeV2` following the same pattern as
the V1 entry, using `golf_tb.DecodeTomlV2`.

**Step 6: Run `just build-go-generate` then `just build`**

Run: `just build-go-generate && just build` Expected: tommy generates
`toml_v2_tommy.go` and `field_definition_tommy.go`, project compiles.

**Step 7: Commit**

Message:
`feat(type_blobs): add TomlV2 with [[fields]] and reader/writer scripts`

--------------------------------------------------------------------------------

### Task 5: Store version bump (V14 -\> V15)

Snapshot current test suite and bump VCurrent.

**Promotion criteria:** N/A (forward-only migration)

**Files:**

- Modify: `go/internal/alfa/store_version/main.go`

**Step 1: Snapshot current tests**

Run: `just test-bats-snapshot-version`

This freezes the current test suite into `zz-tests_bats/previous_versions/v14/`.

**Step 2: Bump VCurrent**

In `go/internal/alfa/store_version/main.go`, add V15 and update VCurrent:

``` go
V15 = Version(values.Int(15))

VCurrent = V15
VNext    = V15
```

**Step 3: Regenerate fixtures**

Run: `just test-bats-update-fixtures`

**Step 4: Run tests**

Run: `just test` Expected: all tests pass with new version

**Step 5: Commit**

Message: `chore: bump store version to V15 for field codec support`

Commit the snapshot, fixtures, and version bump together.

--------------------------------------------------------------------------------

### Task 6: Add `key_bytes.Field` and binary codec support

Add the new key byte and encoder/decoder cases for field key/value pairs.

**Promotion criteria:** N/A (new codec field)

**Files:**

- Modify: `go/internal/_/key_bytes/main.go` (add Field constant)
- Modify: `go/internal/india/stream_index/binary_field.go` (add to
  binaryFieldOrder)
- Modify: `go/internal/india/stream_index/binary_encoder.go` (add encoding case)
- Modify: `go/internal/india/stream_index/binary_decoder.go` (add decoding case)

**Step 1: Write failing unit test**

In `go/internal/india/stream_index/binary_test.go`, add a test that encodes an
object with a Field entry and decodes it, asserting the field survives the
round-trip. Look at existing tests in that file for the pattern.

**Step 2: Run test to verify it fails**

Run: `just test-go-pkg ./internal/india/stream_index/` Expected: FAIL (Field key
byte doesn't exist yet)

**Step 3: Add `key_bytes.Field`**

In `go/internal/_/key_bytes/main.go`, add:

``` go
Field = Binary('F')
```

Run `just build-go-generate` to regenerate `binary_string.go`.

**Step 4: Add to `binaryFieldOrder`**

In `binary_field.go`, append `key_bytes.Field` after `key_bytes.CacheTags`:

``` go
var binaryFieldOrder = []key_bytes.Binary{
    // ... existing entries ...
    key_bytes.CacheTags,
    key_bytes.Field,
}
```

**Step 5: Add encoder case**

In `binary_encoder.go`, add a case in `writeFieldKey` for `key_bytes.Field`.
Follow the `Tag` pattern (loop over collection, one entry per field):

``` go
case key_bytes.Field:
    for field := range metadata.GetIndex().GetFields() {
        if field.TypeBlobDigest.IsEmpty() {
            continue // skip non-type-defined fields
        }
        encoder.writeFieldStringer(/* key */)
        // Write key and value as two successive string fields
        // Pattern: write key string, then value string
    }
```

The exact wire format: for each type-defined field, write one binaryField entry
containing `[key\x00value]` (null-separated key and value). This keeps each
field as a single codec entry.

**Step 6: Add decoder case**

In `binary_decoder.go`, add the matching case in `readFieldKey`:

``` go
case key_bytes.Field:
    // Read null-separated key\x00value from content
    // Create Field{Key, Value, Type: TypeUserData}
    // Append to metadata.GetIndexMutable().GetFieldsMutable()
```

**Step 7: Run test to verify it passes**

Run: `just test-go-pkg ./internal/india/stream_index/` Expected: PASS

**Step 8: Commit**

Message: `feat(stream_index): add binary codec for type-defined fields`

--------------------------------------------------------------------------------

### Task 7: Add `TokenMatcherKeyValueNegated` for doddish queries

Add the `key^=value` token matcher for negated field queries.

**Promotion criteria:** N/A (new code)

**Files:**

- Modify: `go/internal/_/doddish/token_matcher.go`

**Step 1: Write failing test**

In `go/internal/_/doddish/scanner_test.go` or `seq_test.go`, add a test that
scans `status^=cancelled` and matches against the new
`TokenMatcherKeyValueNegated` pattern.

**Step 2: Run test to verify it fails**

Run: `just test-go-pkg ./internal/_/doddish/` Expected: FAIL

**Step 3: Add TokenMatcherKeyValueNegated**

In `token_matcher.go`, add:

``` go
// key^=value
TokenMatcherKeyValueNegated = TokensMatcher{
    TokenTypeIdentifier,
    TokenMatcherOp(OpNegation),
    TokenMatcherOp(OpExact),
}

// key^="value"
TokenMatcherKeyValueNegatedLiteral = TokensMatcher{
    TokenTypeIdentifier,
    TokenMatcherOp(OpNegation),
    TokenMatcherOp(OpExact),
    TokenTypeLiteral,
}
```

**Step 4: Run test to verify it passes**

Run: `just test-go-pkg ./internal/_/doddish/` Expected: PASS

**Step 5: Commit**

Message: `feat(doddish): add TokenMatcherKeyValueNegated for field queries`

--------------------------------------------------------------------------------

### Task 8: Wire field queries into query engine

Add `expField` expression type and wire `key=value` / `key^=value` parsing into
`parseTokens()`.

**Promotion criteria:** N/A (new code)

**Files:**

- Create: `go/internal/kilo/queries/exp_field.go`
- Modify: `go/internal/kilo/queries/build_state.go` (add case in `parseTokens`)
- Test: `go/internal/kilo/queries/builder_test.go`

**Step 1: Write failing test**

In `builder_test.go`, add a test that builds a query from `status=completed` and
verifies it matches an object with that field value but not one without. Follow
existing test patterns in that file.

**Step 2: Run test to verify it fails**

Run: `just test-go-pkg ./internal/kilo/queries/` Expected: FAIL

**Step 3: Create `exp_field.go`**

``` go
package queries

type expField struct {
    Key     string
    Value   string
    Negated bool
}

func (exp *expField) ContainsSku(getter ObjectGetter) bool {
    for field := range getter.GetMetadata().GetIndex().GetFields() {
        if field.Key == exp.Key {
            if exp.Negated {
                return field.Value != exp.Value
            }
            return field.Value == exp.Value
        }
    }
    // Field not present: negated match succeeds, positive match fails
    return exp.Negated
}
```

The exact interface for `ContainsSku` needs to match the existing expression
pattern used by `expSigilAndGenre` and others. Check how `ObjectId.ContainsSku`
is called and match that signature.

**Step 4: Add case in `parseTokens()`**

In `build_state.go`, add a case in the token sequence matching loop that detects
`TokenMatcherKeyValue` and `TokenMatcherKeyValueNegated` patterns, creates an
`expField`, and adds it to the query.

**Step 5: Run test to verify it passes**

Run: `just test-go-pkg ./internal/kilo/queries/` Expected: PASS

**Step 6: Commit**

Message: `feat(queries): add field equality and negation query predicates`

--------------------------------------------------------------------------------

### Task 9: Field read pipeline (commit-time projection)

Wire the fields-reader script execution into the commit path so field values are
projected from blobs and cached in the index.

**Promotion criteria:** N/A (new pipeline)

**Files:**

- Identify the commit pipeline entry point where type blob is already resolved
  and blob is being written. This is likely in `tango/repo_actions/` or
  `mike/store/`. The exact location needs to be found by tracing how objects are
  committed with their type blob.
- Create or modify the appropriate file to:
  1.  Check if the resolved type blob implements `WithFields`
  2.  If `GetFieldsReader()` is non-nil, write blob to temp file
  3.  Execute the reader script with `DODDER_BLOB_PATH` env var
  4.  Parse JSON output into field key/value pairs
  5.  Validate against `GetFieldDefinitions()` (enum membership)
  6.  Set fields on the object's index

**Step 1: Write a BATS integration test**

Create a test in `zz-tests_bats/current_version/fields.bats` that:

1.  Inits a repo
2.  Creates a `!task` type with `[[fields]]` and a `[fields-reader]` script
3.  Creates a zettel of type `!task` with a TOML blob containing
    `status = "todo"`
4.  Runs `dodder show :` and asserts the output includes `status=todo` in the
    box format

**Step 2: Run the test to verify it fails**

Run: `just test-bats-targets fields.bats` Expected: FAIL (field projection not
implemented yet)

**Step 3: Implement the projection pipeline**

Trace the commit path to find where to hook in. The fields-reader script
execution should happen after the blob is written but before the object is
indexed.

**Step 4: Run the test to verify it passes**

Run: `just test-bats-targets fields.bats` Expected: PASS

**Step 5: Commit**

Message: `feat: add field projection pipeline (fields-reader script on commit)`

--------------------------------------------------------------------------------

### Task 10: Field write pipeline (organize mutation)

Wire the fields-writer script execution into the organize checkin path so edited
field values are compiled back into the blob.

**Promotion criteria:** N/A (new pipeline)

**Files:**

- The organize checkin path is in `tango/repo_actions/` (organize options and
  commit flow). Find where changed objects are committed after organize.
- Modify to: detect field changes, run `[fields-writer]` script with
  `DODDER_FIELD_<name>` env vars, replace blob, compute new digest.

**Step 1: Write a BATS integration test**

Extend `zz-tests_bats/current_version/fields.bats`:

1.  Init repo, create `!task` type with fields + reader + writer scripts
2.  Create a zettel with `status = "todo"`
3.  Run organize, change `status=todo` to `status=completed` in the organize
    text
4.  Run `dodder show :` and assert the output shows `status=completed`
5.  Verify the blob content also changed (the TOML now has
    `status = "completed"`)

**Step 2: Run test to verify it fails**

Run: `just test-bats-targets fields.bats` Expected: FAIL

**Step 3: Implement the write pipeline**

On organize checkin: collect all field values from the parsed object, run
`[fields-writer]` with env vars + blob path, recompute blob digest. If digest
changed, commit new version.

**Step 4: Run test to verify it passes**

Run: `just test-bats-targets fields.bats` Expected: PASS

**Step 5: Commit**

Message:
`feat: add field mutation pipeline (fields-writer script on organize checkin)`

--------------------------------------------------------------------------------

### Task 11: Field query BATS integration test

End-to-end test for querying by field values.

**Promotion criteria:** N/A (test)

**Files:**

- Modify: `zz-tests_bats/current_version/fields.bats`

**Step 1: Add query integration test**

Extend `fields.bats`:

1.  Init repo, create `!task` type with fields + reader
2.  Create two zettels: one with `status = "todo"`, one with
    `status = "completed"`
3.  Run `dodder show status=todo` and assert only the first zettel appears
4.  Run `dodder show status^=todo` and assert only the second zettel appears

**Step 2: Run test**

Run: `just test-bats-targets fields.bats` Expected: PASS (if tasks 8-9 are
implemented correctly)

**Step 3: Commit**

Message: `test: add field query integration tests`

--------------------------------------------------------------------------------

### Task 12: Validation on checkin

Add enum value validation when fields are set via organize or commit.

**Promotion criteria:** N/A (new code)

**Files:**

- Modify: the field projection pipeline (from Task 9)
- Modify: the field mutation pipeline (from Task 10)

**Step 1: Write a BATS test for invalid enum value**

In `fields.bats`, add a test that:

1.  Creates a `!task` type with `status` enum field
2.  Tries to organize-edit `status=invalid_value`
3.  Asserts the checkin fails with a validation error

**Step 2: Run test to verify it fails**

Expected: currently accepts invalid values

**Step 3: Add validation**

After field values are parsed (from reader script output or organize edit),
check each field against its Definition. For KindEnum, verify the value is in
the `Values` list. Return an error if not.

**Step 4: Run test to verify it passes**

Run: `just test-bats-targets fields.bats` Expected: PASS

**Step 5: Commit**

Message: `feat: validate field values against type definitions on commit`

--------------------------------------------------------------------------------

### Task 13: Create `!task` type blob and run full test suite

Create the actual `!task` type definition with `status` as first field and run
the full suite.

**Promotion criteria:** `!task` status round-trips correctly through organize,
queries, and show for 2 weeks.

**Files:**

- Test fixtures for `!task` type blob (may be in BATS test setup)
- Modify: `zz-tests_bats/current_version/fields.bats` (finalize)

**Step 1: Finalize `!task` type blob TOML**

``` toml
file-extension = "toml"
vim-syntax-type = "toml"

[[fields]]
name = "status"
kind = "enum"
values = ["todo", "in-progress", "blocked", "completed", "cancelled"]
default = "todo"

[fields-reader]
shell = ["bash", "-c"]
script = """
  tomlq -r '{status: .status}' "$DODDER_BLOB_PATH"
"""

[fields-writer]
shell = ["bash", "-c"]
script = """
  tomlq --in-place --toml-output \
    ".status = \"$DODDER_FIELD_status\"" \
    "$DODDER_BLOB_PATH"
"""
```

**Step 2: Run full test suite**

Run: `just test` Expected: all tests pass

**Step 3: Commit**

Message: `feat: add !task type blob with status field definition`

--------------------------------------------------------------------------------

### Task 14: Update FDR-0000 (rename WATI -\> WIT)

Update FDR-0000 to correct WATI references to WIT and flag semantic aliases +
langlang integration.

**Promotion criteria:** N/A (docs)

**Files:**

- Modify: `docs/features/0000-from-chaos.md`

**Step 1: Rename WATI -\> WIT throughout**

Search-and-replace "WATI" with "WIT" in the FDR. Add a note explaining that WIT
refers to WebAssembly Interface Types, the W3C standard.

**Step 2: Add semantic aliases flag**

Add a section noting that semantic aliases (timestamp over string, priority over
u32) are planned for future work, with langlang grammar integration for
validation rules.

**Step 3: Commit**

Message: `docs(FDR-0000): rename WATI to WIT, flag semantic aliases + langlang`

--------------------------------------------------------------------------------

### Task 15: Regenerate fixtures and final test run

Regenerate fixtures after all changes and run the full suite.

**Files:**

- Fixture files in `zz-tests_bats/previous_versions/`

**Step 1: Regenerate fixtures**

Run: `just test-bats-update-fixtures`

**Step 2: Review fixture diff**

Run: `git diff -- zz-tests_bats/previous_versions/`

**Step 3: Run full test suite**

Run: `just test` Expected: all tests pass

**Step 4: Commit fixtures**

Message: `chore: regenerate fixtures for V15 with field codec support`
