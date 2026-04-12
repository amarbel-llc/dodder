# !vtodo Type with Langlang PEG Grammar and Interface Mapping

**Superseded** by
`docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md` (PR #100).
VTODO format is now a haustoria-side concern only --- the CalDAV haustoria
parses VTODOs natively and populates `!task` fields directly, without a
persisted `!vtodo` type or PEG grammar. See FDR-0007 for the updated
implementation status.

## Context

The field infrastructure (mild-elm branch) is complete: type blobs declare
`[[fields]]` with reader/writer scripts, fields are projected on commit,
persisted in the binary codec, queryable via doddish, and mutable via organize.

The haustoria CalDAV bridge currently handles STATUS as a lossy tag mapping
(`status-tags` config). The `!task` type has no iCalendar awareness --- blobs
are plain DESCRIPTION text, and STATUS is discarded on decompile (hardcoded to
NEEDS-ACTION).

This plan introduces `!vtodo` as a concrete type that: 1. Stores the full VTODO
iCalendar text as its blob 2. Uses a langlang PEG grammar to parse the blob and
extract fields 3. Implements `!task`'s actionable interface via a declarative
field-map 4. Replaces the lossy status-tags approach with proper field
round-tripping

## Design

### !vtodo Type Blob (toml-type-v2)

``` toml
file-extension = "ics"
vim-syntax-type = "icalendar"

[[fields]]
name = "status"
kind = "enum"
values = ["NEEDS-ACTION", "IN-PROCESS", "COMPLETED", "CANCELLED"]
default = "NEEDS-ACTION"

[[fields]]
name = "priority"
kind = "u32"

[[fields]]
name = "due"
kind = "string"

[fields-reader]
engine = "langlang-vm"
grammar = "!@blake2b256-<vtodo-grammar-digest>"

[fields-writer]
engine = "langlang-vm"
grammar = "!@blake2b256-<vtodo-grammar-digest>"

[implements.actionable]

[implements.actionable.field-map]
status = "STATUS"
priority = "PRIORITY"
due = "DUE"
```

### PEG Grammar for VTODO

Aim for a full VTODO parser. The grammar parses RFC 5545 iCalendar content
lines, extracting named properties as parse tree nodes. Key challenges:

- **Line folding**: iCalendar uses CRLF + whitespace for continuation lines
- **Nested components**: VALARM, VTIMEZONE within VTODO
- **Property parameters**: `DTSTART;VALUE=DATE:20260405`
- **Quoted strings**: `DESCRIPTION:line1\nline2` (escaped newlines)

If full parsing proves impractical with PEG, fall back to property-line
extraction (match `PROPERTY-NAME:value` patterns, skip nested components).

The grammar blob is stored as a `!`-typed object (null type = tool blob, per
FDR-0010). Referenced by the type blob via typed blob lock.

### Fields-Reader with Langlang Engine

Currently fields-reader uses `script` (shell command). This plan adds an
alternative `engine` mode using langlang:

1.  Type blob declares `engine = "langlang-vm"` + `grammar = "!@digest"`
2.  At commit time, dodder loads the grammar blob, compiles via
    `langlang.NewMatcher(grammarBytes)`
3.  Blob content is parsed, producing a `langlang.Tree`
4.  Field values are extracted from named nodes in the tree
5.  The `[implements.actionable.field-map]` maps grammar node names (e.g.,
    "STATUS") to interface field names (e.g., "status")

This is a `oneof` with the existing `script` field --- a type blob uses either
`engine` or `script`, not both.

### Fields-Writer with Langlang Engine

For mutation (organize changes status field), the writer needs to modify the
iCalendar blob. Two approaches:

- **PEG-guided replacement**: Use the parse tree to locate the STATUS property
  line, replace its value, reconstruct the blob from the modified tree
- **Line-based replacement**: Find `STATUS:...` line, replace with new value

The PEG-guided approach is cleaner but requires langlang tree-to-text
reconstruction. The line-based approach is simpler and works for property
replacement. Start with line-based, upgrade later.

### Interface Conformance: !vtodo implements !task

`!task` exports the `actionable` interface with fields (status, priority, due).
`!vtodo` implements it via `[implements.actionable.field-map]`.

At commit time, the field-map translates: - iCalendar `STATUS` property value →
dodder `status` field - iCalendar `PRIORITY` property value → dodder `priority`
field - iCalendar `DUE` property value → dodder `due` field

This enables querying `!vtodo` objects with `status=COMPLETED` and mutating
status via organize, just like TOML-blob tasks.

### Haustoria Integration

The CalDAV bridge changes: 1. `Compile()` stores the **full VTODO text** as the
blob (not just DESCRIPTION) 2. Object type is `!vtodo` (not `!task`) --- the
interface mapping provides actionable compatibility 3. `Decompile()` reads the
blob (full VTODO), the field system provides the current STATUS value for CalDAV
write-back 4. `status-tags` config becomes optional/deprecated --- field-based
status replaces it

### Migration Path

1.  New repos can use `!vtodo` directly
2.  Existing haustoria workspaces with `!task` objects need migration:
    - Change calendar config `type = "!vtodo"`
    - Re-checkin from CalDAV to populate blobs with full VTODO text
    - Fields are projected on next commit

## Implementation Order

### Phase 1: Langlang Dependency

- Add langlang to go.mod and flake inputs
- Verify `langlang.NewMatcher()` works with a simple grammar
- Store a test grammar as a `!`-typed blob

### Phase 2: Engine-Based Fields-Reader

- Add `engine` field to `ScriptConfig` (or new struct alongside it)
- Add `oneof` logic in `field_reader.go`: if engine, use langlang; if script,
  use shell
- Implement langlang tree → field value extraction

### Phase 3: VTODO PEG Grammar

- Write PEG grammar for iCalendar VTODO
- Test against real VTODO samples from Fastmail
- Store as `!`-typed blob

### Phase 4: !vtodo Type Blob

- Create `!vtodo` type blob with fields, grammar reference, and field-map
- Add `[implements.actionable]` section to TomlV2 struct
- Implement field-map resolution in the field reader

### Phase 5: Haustoria Bridge Update

- Change `Compile()` to store full VTODO as blob
- Change `Decompile()` to read STATUS from field (not hardcode)
- Update workspace config to use `!vtodo` type
- Deprecate `status-tags`

### Phase 6: BATS Tests

- VTODO field projection test
- VTODO organize mutation test
- CalDAV round-trip test (checkin → field change → checkout)

## Files to Modify

- `go/go.mod` --- add langlang dependency
- `go/flake.nix` or root `flake.nix` --- add langlang flake input
- `go/internal/golf/type_blobs/toml_v2.go` --- add Implements struct
- `go/internal/papa/store/field_reader.go` --- add langlang engine path
- `go/internal/papa/store/field_writer.go` --- add langlang engine path
- `go/internal/mike/haustoria_caldav/main.go` --- full VTODO blob storage
- `go/internal/hotel/caldav/parser.go` --- may be replaced by PEG grammar
- `go/internal/echo/workspace_config_blobs/v2.go` --- deprecate status-tags
- `docs/features/0000-from-chaos.md` --- update with vtodo conformance example

## Open Questions

- Can a PEG grammar handle iCalendar line folding (CRLF + WSP continuation)? If
  not, a pre-processing unfold step may be needed before grammar matching.
- Should the field-map support value transformation (e.g., CalDAV "COMPLETED" →
  dodder "done")? Or should dodder adopt iCalendar STATUS values directly?
- How does `verify_wit` validate that `!vtodo` correctly implements actionable?
  Static check of field-map keys against interface field declarations?

## Issues to Update

- #92 --- field mutation via organize: DONE, add note about vtodo
- #93 --- query by fields: DONE, add note about vtodo queries
- #94 --- typed status field: update with vtodo plan, link to this design
- FDR-0000 --- add vtodo as the concrete conformance example
- FDR-0007 --- update haustoria integration with vtodo approach
