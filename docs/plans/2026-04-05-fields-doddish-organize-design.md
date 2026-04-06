# Fields in Doddish and Organize Text Format

Date: 2026-04-05

## Implementation Status

**Implemented** (mild-elm branch, 2026-04-05):

- Data model: `fields.Kind` (WIT-aligned), `fields.Definition`, `TypeBlobDigest`
  on `Field`
- TomlV2 type blob with `[[fields]]`, `[fields-reader]`, `[fields-writer]`
- Store version V15 with binary codec for field key/value/TypeBlobDigest
- Doddish query: `status=todo` equality matching
- Field read pipeline: commit-time projection via script
- Field write pipeline: organize mutation with internal/external fork overlay
- Default output format changed from log to box (fields visible)
- Smart quoting: `status=todo` unquoted, descriptions quoted
- Enum validation on commit
- BATS tests: projection, query, enum validation, organize mutation

**Open issues:**

- #98 --- resolved: infix `status^=todo` doesn't work due to `^` being
  `operatorTypeSoloSeq`; use prefix `^status=todo` instead
- Organize internal/external fork resolution needs cleanup and more tests
- #101 --- `fields-writer` should support blob-less first writes. Currently
  `papa/store/field_writer.go:25-27` early-returns when the daughter has no blob
  digest, so the writer can't create a blob from scratch when fields are set
  programmatically. Blocks the haustoria use case in PR #100 and any future
  "construct object from fields" code path.
- #102 --- when `[fields-reader]` is configured but `[fields-writer]` is not,
  organize edits are silently dropped AND fields appear duplicated in
  `dodder show` output (`status=todo status=todo`). Two regressions in one
  output line, exposed by PR #100 probes.

## Problem

Dodder objects have metadata (tags, type, description) but no mechanism for
type-defined structured fields (status, priority, due date). Fields exist today
as presentation artifacts in the box format but are not persisted, not
queryable, and not connected to type definitions. CalDAV integration (#94),
field queries (#93), and organize mutation (#92) all require this
infrastructure.

## Approach

Bottom-up assembly: build the full stack (type blob declaration, binary codec,
query engine, organize round-trip) for one concrete field (`status` on `!task`),
with each layer designed generically from day one.

## Data Model

### `fields.Kind` --- WIT-Aligned Type Enum

Field kinds align with WebAssembly Interface Types (WIT) primitives, not
application-level semantics. All five are defined as constants; only
`KindString` and `KindEnum` are implemented in the reader/writer/query pipeline
for v1.

    type Kind byte

    const (
        KindString     Kind = iota // WIT: string
        KindEnum                   // WIT: enum
        KindBool                   // WIT: bool
        KindU32                    // WIT: u32
        KindS32                    // WIT: s32
        KindListString             // WIT: list<string>
    )

**Flag:** Semantic aliases (timestamp over string, priority over u32) are a
future extension. Integration with amarbel-llc/langlang for grammar-based type
constraints is planned --- langlang grammars could define the validation rules
for semantic aliases.

**Done:** FDR-0000 renamed "WATI" to "WIT" throughout, with note that WIT refers
to WebAssembly Interface Types (W3C standard). Semantic aliases and langlang
integration flagged in FDR.

### `fields.Definition` --- Field Schema

    type Definition struct {
        Name    string
        Kind    Kind
        Values  []string // populated for KindEnum
        Default string
    }

### `fields.Field` --- Concrete Value on an Object

    type Field struct {
        Type                       // byte enum for rendering (kept)
        Key, Value    string
        TypeBlobDigest markl.Id    // lookup hint for lazy definition resolution
    }

- `Type` byte is kept for rendering (TypeId, TypeHash, TypeUserData, etc.)
- `*Definition` is NOT embedded --- definitions are resolved lazily via
  `typeBlobStore.Get(digest)` then finding the `[[fields]]` entry by name
- `TypeBlobDigest` enables direct lookup without traversing object -\> type -\>
  blob
- Builtin definitions for native box fields (object-id, blob digest, type, tags)
  are package-level vars, not used for validation yet

**Flag:** `TypeBlobDigest` could become a `PageId + offset` into the stream
index once the index format stabilizes. This is a general pattern worth
exploring: lightweight intra-index references instead of full content-addressed
digests.

**Flag:** Field as universal index primitive --- if every piece of metadata in
the index were represented as a Field with a Definition that describes its codec
behavior, the binary encoder/decoder could be driven by field definitions
instead of a hardcoded switch statement. The `binaryFieldOrder` + per-key
encode/decode cases would collapse into a generic loop. Significant
architectural shift, aligns with self-describing type system vision.

## Type Blob TOML (`TomlV2`)

New type blob version. `[[fields]]` uses WIT-shaped array-of-tables syntax for
forward compatibility with WIT interface contracts.

    # !task type blob (toml-type-v2)
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
      tomlq '{status}' "$DODDER_BLOB_PATH"
    """

    [fields-writer]
    shell = ["bash", "-c"]
    script = """
      tomlq --in-place --toml-output \
        ".status = \"$DODDER_FIELD_status\"" \
        "$DODDER_BLOB_PATH"
    """

Go struct --- `TomlV2` extends `TomlV1` with:

- `Fields []FieldDefinition` --- TOML `[[fields]]` array
- `FieldsReader *ScriptConfig` --- blob -\> field values
- `FieldsWriter *ScriptConfig` --- field values -\> blob

Tommy codegen handles round-trip TOML parsing. `TomlV0` and `TomlV1` remain
decodable. Types without `[[fields]]` in `TomlV2` behave identically to
`TomlV1`.

**Flag:** Eventually per-field readers/writers may be supported alongside the
batch reader/writer, and the WASM model would use a single function for all
field reading.

## Binary Codec and Store Version

### Store Version Bump

Snapshot current tests into `previous_versions/v15/`, bump `VCurrent`.

### New Key Byte

Add `key_bytes.Field` (e.g., `Binary('F')`) for serializing type-defined field
key/value pairs. Placed after `CacheTags` in `binaryFieldOrder`.

### Encoder

For each field on the index with a non-empty `TypeBlobDigest` (type-defined):

    [key_bytes.Field] [key length u16] [key bytes] [value length u16] [value bytes]

Multiple fields produce multiple entries (same pattern as `Tag`).

### Decoder

Read key/value string pairs, create `Field{Key, Value, Type: TypeUserData}`.
`TypeBlobDigest` is set from the object's own type metadata after decode.
`Definition` is not serialized --- resolved lazily via the type blob store.

## Field Read/Write Pipeline

### On Commit (new object or blob change)

1.  Object's type is resolved, type blob parsed, check for `[[fields]]` +
    `[fields-reader]`
2.  Blob is written to a temp file, `fields-reader` script is executed with
    `DODDER_BLOB_PATH` env var
3.  Script outputs field values (JSON, one object with field names as keys)
4.  Field values are parsed, validated against `[[fields]]` definitions (enum
    membership, etc.)
5.  `Field` entries are set on the object's index with `TypeBlobDigest` hint
6.  Object is persisted with field values in the binary index

**Flag:** JSON output format for the reader script is provisional and may
change.

### On Organize Mutation

1.  Organize reader parses all field values from the box format line
2.  On checkin, run `[fields-writer]` script with ALL current field values
    (`DODDER_FIELD_<name>` env vars) + current blob -\> produces new blob
3.  Compute new blob digest
4.  If digest differs from original -\> new object version. If same -\> no
    change

Full compile, no diff detection. Fields are projections from the blob: the blob
is the source of truth, fields are a cached view.

### Field Semantics

Fields are NOT independent metadata. They are cached projections of values
extracted from the blob by the type system (via `fields-reader`). Since blobs
are content-addressed, the cached field values are deterministic given a blob
digest. When the blob changes, a new object version is created with new field
values --- old versions keep their old field values forever. No cache
invalidation.

**Flag:** Mutable/chaos fields (values depending on state outside the
content-addressed graph) are a future extension. The type definition would need
to distinguish projected fields from mutable fields. Not in v1 scope.

## Doddish Query Support

### Syntax

- `status=done` --- equality match on cached field value
- `^status=done` --- negated equality (prefix `^`, not infix `key^=value` ---
  the `^` operator is `operatorTypeSoloSeq` in doddish and breaks the token
  sequence when placed between key and `=`)

### Scanner

`TokenMatcherKeyValue` already exists. Add `TokenMatcherKeyValueNegated` for
`key^=value` pattern.

### Query Engine (`kilo/queries/`)

- `parseTokens()` gets a new case matching `key=value` and `key^=value` token
  sequences
- New expression type `expField` holding `Key string`, `Value string`,
  `Negated bool`
- `expField.ContainsSku()` checks the object's index fields for matching
  key/value pair

Queries scan cached field values in the stream index. No blob loading at query
time.

**Flag:** Full comparison operators (`<`, `>`, `<=`, `>=`) require `Kind`
information from the type blob definition. Blocked until definitions are
available at query time via the lazy lookup path or index offset references.

## Organize Format Integration

### Display and Parsing

Already works. Box format writes fields as `key=value` and reads them via
`TokenMatcherKeyValue`. No changes needed.

### Checkin

Organize reader parses the full object line including field values. Checkin
takes parsed field values and runs the full compile: `fields-writer` script -\>
new blob -\> new digest. New object version committed if digest changed.

### Validation

On checkin, field values are validated against the type's `[[fields]]`
definitions. For `KindEnum`, the value must be in the `values` list. Invalid
values produce an error before commit.

**Flag:** Future organize features once this stabilizes: field completion,
field-specific grouping/sorting, inline enum value cycling, field-aware diffing,
etc.

## Rollback and Migration

### Type Blob (`TomlV1` -\> `TomlV2`)

- `TomlV1` remains decodable
- Types without `[[fields]]` in `TomlV2` behave identically to `TomlV1`
- Rollback: change type blob header back to `toml-type-v1`, fields stop being
  projected, objects otherwise unaffected

### Store Version Bump

- Old versions don't have field entries --- migration populates fields by
  running `fields-reader` on existing blobs for types that declare fields
- Old store versions ignore unknown key bytes, field data silently skipped on
  rollback

### Dual-Architecture Period

- Types without `[[fields]]` continue working unchanged
- `!task` is the only type that gets a `TomlV2` blob with fields initially
- Promotion criteria: `!task` status round-trips correctly through organize,
  queries, and CalDAV bridge for 2 weeks

## Related Issues and FDRs

- #92 --- Field mutation via organize and checkin
- #93 --- Query by fields
- #94 --- Typed status field on `!task`
- #38 --- Binary codec field coverage
- FDR-0000 --- Type interface contracts (rename WATI -\> WIT)
- FDR-0007 --- Pluggable checkout stores (CalDAV field mapping)
- FDR-0009 --- External object index (user fields vs cache fields)
