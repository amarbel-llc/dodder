# Extract `alfa/fields` Package from `string_format_writer.Field`

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Separate the semantic field type (`Type`, `Key`, `Value`) from
presentation concerns (`Separator`, `DisableValueQuotes`, `NoTruncate`,
`NeedsNewline`) by extracting a new `alfa/fields` package.

**Architecture:** Create `alfa/fields` with a domain `Field` struct and `Type`
(renamed from `ColorType`). Rename `string_format_writer.Field` to
`FormattedField`, embedding `fields.Field`. Update `delta/objects` alias to
point at `fields.Field`. Update all callers.

**Tech Stack:** Go, no new dependencies.

**Rollback:** `git revert` the commit(s). Purely internal refactor, no external
API or serialization changes.

--------------------------------------------------------------------------------

## Naming Changes

  ----------------------------------------------------------------------------------
  Old                                        New
  ------------------------------------------ ---------------------------------------
  `string_format_writer.ColorType` (type)    `fields.Type`

  `string_format_writer.ColorTypeNormal`     `fields.TypeNormal`

  `string_format_writer.ColorTypeId`         `fields.TypeId`

  `string_format_writer.ColorTypeHash`       `fields.TypeHash`

  `string_format_writer.ColorTypeError`      `fields.TypeError`

  `string_format_writer.ColorTypeType`       `fields.TypeType`

  `string_format_writer.ColorTypeUserData`   `fields.TypeUserData`

  `string_format_writer.ColorTypeHeading`    `fields.TypeHeading`

  `string_format_writer.Field` (struct)      `string_format_writer.FormattedField`

  `objects.Field` (alias)                    `= fields.Field` (was
                                             `= string_format_writer.Field`)
  ----------------------------------------------------------------------------------

## Bridge Sites

Three call sites append `fields.Field` (from metadata `GetFields()`) into
`Box.Contents` (which becomes `Slice[FormattedField]`). These need a conversion
loop using `FormattedField{Field: f}`:

1.  `hotel/box_format/checked_out.go:263` --- `quiter.AppendSeq`
2.  `hotel/box_format/transacted.go:284` --- `quiter.AppendSeq`
3.  `india/sku_fmt/deleted.go:70` --- `slices.Collect`

--------------------------------------------------------------------------------

### Task 1: Create `alfa/fields` package

**Files:** - Create: `go/internal/alfa/fields/main.go`

**Step 1: Create the package**

``` go
package fields

type Type string

const (
    TypeNormal   Type = ""
    TypeId       Type = "\u001b[34m"
    TypeHash     Type = "\u001b[3m"
    TypeError    Type = "\u001b[31m"
    TypeType     Type = "\u001b[33m"
    TypeUserData Type = "\u001b[36m"
    TypeHeading  Type = "\u001b[31m"
)

type Field struct {
    Type
    Key, Value string
}
```

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/loud-plum/go && go build ./internal/alfa/fields/`

**Step 3: Commit**

    feat: add alfa/fields package with domain Field type

--------------------------------------------------------------------------------

### Task 2: Update `string_format_writer` to use `fields`

**Files:** - Modify: `go/internal/alfa/string_format_writer/options.go` -
Modify: `go/internal/alfa/string_format_writer/main.go` - Modify:
`go/internal/alfa/string_format_writer/fields_writer.go` - Modify:
`go/internal/alfa/string_format_writer/color.go`

**Step 1: Update `options.go`**

Replace `ColorType string` with alias `ColorType = fields.Type`. Keep
`ColorOptions` and `OutputOptions` unchanged.

**Step 2: Update `main.go`**

Remove `ColorType*` constants. Add aliases:

``` go
const (
    ColorTypeNormal   = fields.TypeNormal
    ColorTypeId       = fields.TypeId
    ColorTypeHash     = fields.TypeHash
    ColorTypeError    = fields.TypeError
    ColorTypeType     = fields.TypeType
    ColorTypeUserData = fields.TypeUserData
    ColorTypeHeading  = fields.TypeHeading
)
```

Keep unexported ANSI constants (`colorReset`, etc.) --- they are still used by
`boxStringEncoder.writeStringFormatField` and `color.go`. Remove the ones that
are only used by the now-removed `ColorType*` constants (they are duplicated in
`fields`).

**Step 3: Rename `Field` to `FormattedField` in `fields_writer.go`**

- Rename `type Field struct` → `type FormattedField struct`
- Replace `ColorType` embedded field with `fields.Field` embed
- Keep presentation fields: `Separator`, `DisableValueQuotes`, `NoTruncate`,
  `NeedsNewline`
- Move the `// TODO switch to using io.StringWriter` comment to `fields.Field`
- Update `Box` to use `FormattedField`
- Update `boxStringEncoder.EncodeStringTo` internal `Field{...}` literals →
  `FormattedField{...}`
- Update `writeStringFormatField` parameter type
- Update `writeStringFormatField` body: `field.ColorType` → `field.Type`,
  `field.Key`/`field.Value` still work via embedding

**Step 4: Update `color.go`**

`ColorType` is now an alias to `fields.Type`, so `color.color` field type and
`MakeColor` param type still compile. No changes needed unless `ColorType` was
used as a concrete type (it's not --- it's only used in type position).

**Step 5: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/loud-plum/go && go build ./internal/alfa/string_format_writer/`

**Step 6: Commit**

    refactor: rename Field to FormattedField, embed fields.Field

--------------------------------------------------------------------------------

### Task 3: Update `delta/objects` alias

**Files:** - Modify: `go/internal/delta/objects/index.go`

**Step 1: Change the alias**

``` go
import (
    "code.linenisgreat.com/dodder/go/internal/alfa/fields"
    // remove string_format_writer import if no longer needed
)

type (
    Field = fields.Field
    // ...
)
```

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/loud-plum/go && go build ./internal/delta/objects/`

**Step 3: Commit**

    refactor: point objects.Field alias at fields.Field

--------------------------------------------------------------------------------

### Task 4: Update metadata-populating callers (domain side)

These sites construct `Field` for
`metadata.GetIndexMutable().GetFieldsMutable()`. After the alias change they
must use `fields.Field` (or `objects.Field` which is the same). Replace
`string_format_writer.ColorType*` with `fields.Type*`.

**Files:** - Modify: `go/internal/mike/store_fs/main.go:502-504` - Modify:
`go/internal/sierra/store_browser/item.go:131-146` - Modify:
`go/internal/foxtrot/object_metadata_fmt_hyphence/text_parser.go:76-80` -
Modify: `go/internal/hotel/box_format/read.go:337-342`

**store_fs/main.go** (line 502):

``` go
// Before:
field := objects.Field{
    Value:     fdee.GetPath(),
    ColorType: string_format_writer.ColorTypeId,
}

// After:
field := objects.Field{
    Value: fdee.GetPath(),
    Type:  fields.TypeId,
}
```

Remove `string_format_writer` import, add `fields` import.

**store_browser/item.go** (lines 131-146):

``` go
// Before:
string_format_writer.Field{
    Key:       "title",
    Value:     item.Title,
    ColorType: string_format_writer.ColorTypeUserData,
}

// After (uses objects.Field or fields.Field directly):
fields.Field{
    Key:   "title",
    Value: item.Title,
    Type:  fields.TypeUserData,
}
```

Same pattern for the `url` field. Replace `string_format_writer` import with
`fields`.

**text_parser.go** (line 76):

``` go
// Before:
string_format_writer.Field{
    Key:       "blob",
    Value:     parser2.Blob.GetPath(),
    ColorType: string_format_writer.ColorTypeId,
}

// After:
fields.Field{
    Key:   "blob",
    Value: parser2.Blob.GetPath(),
    Type:  fields.TypeId,
}
```

Check if `string_format_writer` is still imported elsewhere in this file. If
not, remove the import.

**box_format/read.go** (line 337):

``` go
// Before:
field := string_format_writer.Field{
    Key:   string(seq.At(0).Contents),
    Value: value.String(),
}
field.ColorType = string_format_writer.ColorTypeUserData

// After:
field := fields.Field{
    Key:   string(seq.At(0).Contents),
    Value: value.String(),
    Type:  fields.TypeUserData,
}
```

**Step: Verify it compiles**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/loud-plum/go && go build ./...`

**Commit:**

    refactor: update metadata field producers to use fields.Field

--------------------------------------------------------------------------------

### Task 5: Update box-building callers (presentation side)

These sites construct `FormattedField` for `Box.Contents` / `Box.Trailer`.
Replace `string_format_writer.Field{...}` with
`string_format_writer.FormattedField{Field: fields.Field{...}, ...}`.

**Files:** - Modify: `go/internal/echo/object_metadata_fmt/fields.go` - Modify:
`go/internal/echo/object_metadata_fmt/main.go` - Modify:
`go/internal/echo/object_metadata_box_builder/main.go` - Modify:
`go/internal/hotel/box_format/transacted.go` - Modify:
`go/internal/hotel/box_format/checked_out.go`

**Pattern for simple fields (no presentation flags):**

``` go
// Before:
string_format_writer.Field{
    Value:     metadata.GetTai().String(),
    ColorType: string_format_writer.ColorTypeHash,
}

// After:
string_format_writer.FormattedField{
    Field: fields.Field{
        Value: metadata.GetTai().String(),
        Type:  fields.TypeHash,
    },
}
```

**Pattern for fields with presentation flags:**

``` go
// Before:
string_format_writer.Field{
    Key:        "error",
    Value:      err.Error(),
    ColorType:  string_format_writer.ColorTypeUserData,
    NoTruncate: true,
}

// After:
string_format_writer.FormattedField{
    Field: fields.Field{
        Key:   "error",
        Value: err.Error(),
        Type:  fields.TypeUserData,
    },
    NoTruncate: true,
}
```

**Pattern for fields with Separator:**

``` go
// Before:
string_format_writer.Field{
    Key:        id.GetPurposeId(),
    Separator:  '@',
    Value:      id.String(),
    NoTruncate: true,
    ColorType:  string_format_writer.ColorTypeHash,
}

// After:
string_format_writer.FormattedField{
    Field: fields.Field{
        Key:   id.GetPurposeId(),
        Value: id.String(),
        Type:  fields.TypeHash,
    },
    Separator:  '@',
    NoTruncate: true,
}
```

**`object_metadata_fmt/fields.go` changes:**

- Return types: `[]string_format_writer.Field` →
  `[]string_format_writer.FormattedField`
- Single returns: `string_format_writer.Field` →
  `string_format_writer.FormattedField`
- All struct literals updated per patterns above
- `MetadataFieldTags` returns `[]FormattedField` --- sort lambda accesses
  `.Value` which still works via embed promotion

**`object_metadata_fmt/main.go` changes:**

- `collections_slice.Slice[string_format_writer.Field]` →
  `collections_slice.Slice[string_format_writer.FormattedField]`
- All struct literals updated per patterns above
- `makeMarklIdField` return type changes
- Parameter `colorType string_format_writer.ColorType` → `colorType fields.Type`
  (if used here; check --- it's in box_builder)

**`object_metadata_box_builder/main.go` changes:**

- `type Builder string_format_writer.Box` --- unchanged (Box now uses
  `FormattedField`, so Builder follows)
- All `string_format_writer.Field{...}` →
  `string_format_writer.FormattedField{...}`
- `colorType string_format_writer.ColorType` params → `colorType fields.Type`
- `builder.Contents.Append(string_format_writer.Field{...})` calls all updated
- `sort.Slice` lambda accessing `.Value` still works via promotion

**`hotel/box_format/transacted.go` changes:**

- `var external string_format_writer.Field` →
  `var external string_format_writer.FormattedField`
- `var internal string_format_writer.Field` →
  `var internal string_format_writer.FormattedField`
- All `string_format_writer.Field{...}` literals → `FormattedField{...}`
- Function return types:
  - `makeFieldExternalObjectIdsIfNecessary` → returns `FormattedField`
  - `makeFieldObjectId` → returns `FormattedField`

**`hotel/box_format/checked_out.go` changes:**

- Same pattern as transacted.go for the shared methods
- `var id string_format_writer.Field` →
  `var id string_format_writer.FormattedField`
- `id.ColorType = ...` → `id.Type = ...` (field promotion)
- Trailer append updated

**Verify:**
`cd /home/sasha/eng/repos/dodder/.worktrees/loud-plum/go && go build ./...`

**Commit:**

    refactor: update box-building callers to use FormattedField

--------------------------------------------------------------------------------

### Task 6: Update bridge sites and remaining callers

**Files:** - Modify: `go/internal/hotel/box_format/checked_out.go:263` - Modify:
`go/internal/hotel/box_format/transacted.go:284` - Modify:
`go/internal/india/sku_fmt/deleted.go:70` - Modify:
`go/internal/delta/id_fmts/cli_format_fd.go:22` - Modify:
`go/internal/bravo/descriptions/format_cli_generic.go:29` - Modify:
`go/internal/sierra/local_working_copy/printers.go:73`

**Bridge sites** --- where `fields.Field` from metadata is appended into
`Slice[FormattedField]`:

``` go
// Before (checked_out.go:263, transacted.go:284):
quiter.AppendSeq(&builder.Contents, metadata.GetIndex().GetFields())

// After:
for field := range metadata.GetIndex().GetFields() {
    builder.Contents.Append(string_format_writer.FormattedField{Field: field})
}
```

``` go
// Before (deleted.go:70):
Contents: slices.Collect(object.GetMetadata().GetIndex().GetFields()),

// After:
Contents: quiter.CollectAs(
    object.GetMetadata().GetIndex().GetFields(),
    func(f fields.Field) string_format_writer.FormattedField {
        return string_format_writer.FormattedField{Field: f}
    },
),
```

If `quiter.CollectAs` doesn't exist, use an inline loop:

``` go
var contents []string_format_writer.FormattedField
for field := range object.GetMetadata().GetIndex().GetFields() {
    contents = append(contents, string_format_writer.FormattedField{Field: field})
}
// ... Box{Contents: contents, ...}
```

**Remaining callers** --- these use `ColorType` constants only (not `Field`
struct), so just update the constant references:

- `id_fmts/cli_format_fd.go:22`: `string_format_writer.ColorTypeId` --- no
  change needed (alias still works)
- `descriptions/format_cli_generic.go:29`:
  `string_format_writer.ColorTypeUserData` --- no change needed (alias still
  works)
- `local_working_copy/printers.go:73`: `string_format_writer.ColorTypeHeading`
  --- no change needed (alias still works)

These callers pass `ColorType` to `MakeColor()`, which accepts
`string_format_writer.ColorType` (alias for `fields.Type`). The backward-compat
aliases in `string_format_writer/main.go` mean these compile without changes.

**Verify:**
`cd /home/sasha/eng/repos/dodder/.worktrees/loud-plum/go && go build ./...`

**Commit:**

    refactor: convert fields.Field to FormattedField at bridge sites

--------------------------------------------------------------------------------

### Task 7: Build and test

**Step 1:** `cd /home/sasha/eng/repos/dodder/.worktrees/loud-plum && just build`

**Step 2:** `just test-go` (unit tests)

**Step 3:** `just test` (full suite including BATS)

Fix any failures. This is a purely internal refactor --- no serialization or
output format changes --- so tests should pass unchanged.

**Commit (if any fixes):**

    fix: address test failures from fields extraction

--------------------------------------------------------------------------------

### Task 8: Clean up backward-compat aliases (optional)

Once everything passes, optionally migrate callers of the backward-compat
aliases (`string_format_writer.ColorType*`) to use `fields.Type*` directly. This
removes the transitive dependency on `string_format_writer` for callers that
only need the type constants.

Files to update: - `go/internal/delta/id_fmts/cli_format_fd.go` -
`go/internal/bravo/descriptions/format_cli_generic.go` -
`go/internal/sierra/local_working_copy/printers.go` -
`go/internal/india/sku_fmt/deleted.go`

Then remove the aliases from `string_format_writer/main.go` and the `ColorType`
alias from `options.go`.
