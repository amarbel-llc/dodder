# `!task` type at genesis + haustoria-side VTODO field mapping

## Context

The fields infrastructure (mild-elm branch) is complete: type blobs declare
`[[fields]]` entries, fields are projected on commit, persisted in the binary
codec, queryable via doddish, and mutable via organize. The CalDAV haustoria
currently handles VTODO `STATUS` as a lossy `status-tags` config mapping that
projects status into a *tag*; `PRIORITY` and `DUE` are dropped entirely.

The previous design (`docs/plans/2026-04-05-vtodo-langlang-design.md`) proposed
adding `!vtodo` as a persisted type with a langlang PEG grammar to extract
fields from a raw VTODO blob. **That plan is dropped.** VTODO format is a
haustoria-side concern, not a repo-side concern. The CalDAV haustoria already
parses VTODOs natively (`hotel/caldav/parser.go`); it can populate `!task`
fields directly without round-tripping through a PEG grammar or persisting any
iCal-shaped type.

This plan does two things:

1.  Formally commit a `!task` type into the bigbang/genesis process so every
    fresh repo has it as a built-in.
2.  Hardcode the VTODO ↔ `!task` field mapping inside `mike/haustoria_caldav` so
    `status`, `priority`, and `due` round-trip as typed fields instead of as a
    lossy `zz-archive-task-done` tag.

## Out of Scope

- `!vtodo` as a persisted type (dropped --- haustoria-side only).
- Langlang PEG grammars for any iCal property.
- Generic interface mapping (`[implements.actionable.field-map]`).
- `!chore` as a separate type. The existing `chores` calendar in
  `~/workspaces/dodder-haustoria-caldav` will reuse `!task`.
- Migration of existing repos. V14 → V15 was the only recent format bump and
  it's already shipped; this plan does not change persisted formats.

## Design

### 1. `!task` type blob

`golf/type_blobs/main.go` gains a `DefaultTaskType()` function returning a
`TomlV2` with the following content:

``` toml
file-extension = "task"
vim-syntax-type = "taskpaper"

[[fields]]
name = "status"
kind = "enum"
values = ["todo", "in-process", "done", "cancelled"]
default = "todo"

[[fields]]
name = "priority"
kind = "u32"

[[fields]]
name = "due"
kind = "string"
```

Status enum values match the field-mutation work in `19c6f63` ("use 'done'
instead of 'completed' in task status enum values"). The mapping to/from VTODO
STATUS is the haustoria's job (see §3 below).

Open question --- see "Decisions to confirm" --- whether the file extension
should be `task`, `taskpaper`, or omitted.

### 2. Genesis: commit `!task` alongside `!md`

`local_working_copy/genesis.go` gains a sibling helper next to
`prepareDefaultType`:

``` go
func (local *Repo) prepareTaskType(
    bigBang env_repo.BigBang,
    builder *import_plan.Builder,
) (objectIdType ids.TypeStruct, blobDigest domain_interfaces.MarklId, err error)
```

Behavior:

- Allocates an `id.TypeStruct` for `task`.
- Calls `type_blobs.DefaultTaskType()` to get the blob.
- Saves the blob via the typed blob store, capturing the digest.
- Constructs a `sku.Transacted` for the type object, sets its blob digest and
  type (`!toml-type-v2`), and adds it to the import plan builder.
- Returns the captured blob digest so the haustoria can stamp it on `Field`
  records (see §3).

`initDefaultTypeAndConfig` calls both `prepareDefaultType` and
`prepareTaskType`. The `!task` blob digest is stored on the `Repo` (or fetched
on demand via a query) so the haustoria initialization step can find it.

The new step is gated on the existing `BigBang.ExcludeDefaultType` flag --- same
flag that already gates `!md`. This means `init-workspace` and `clone` both skip
it, which matches existing behavior for default types.

No new `BigBang` field is added.

### 3. Haustoria: VTODO ↔ `!task` field mapping

`mike/haustoria_caldav/main.go` changes:

**Drop `CalendarMapping.StatusTags`** and the
`[haustoria.calendars.*.status-tags]` config keys in
`echo/workspace_config_blobs`. Tommy codegen for the V2 workspace config is
regenerated. The status-tag tests are deleted (see §4).

**Add a `taskTypeBlobDigest` lookup on `Initialize`.** When `s.supplies` is set,
the haustoria does a one-shot
`s.supplies.ReadPrimitiveQuery(taskTypeQuery, ...)` to find the committed
`!task` type object and caches its blob digest. If the type is missing
(workspace, cloned repo with `ExcludeDefaultType`), the haustoria returns an
init error pointing the user at the missing builtin --- the workspace mode isn't
expected to host a CalDAV haustoria anyway.

**Compile path (`queryCheckedOutForCalendar`)**: replace the `StatusTags` block
with direct field projection.

``` go
metadata := external.GetMetadataMutable()
indexFields := metadata.GetIndexMutable().GetFieldsMutable()

statusValue := mapVTODOStatusToFieldValue(twm.Task.Status) // "todo" default
indexFields.Add(fields.Field{
    Key:            "status",
    Value:          statusValue,
    TypeBlobDigest: s.taskTypeBlobDigest,
})

if twm.Task.Priority != 0 {
    indexFields.Add(fields.Field{
        Key:            "priority",
        Value:          strconv.Itoa(twm.Task.Priority),
        TypeBlobDigest: s.taskTypeBlobDigest,
    })
}

if twm.Task.Due != "" {
    indexFields.Add(fields.Field{
        Key:            "due",
        Value:          twm.Task.Due,
        TypeBlobDigest: s.taskTypeBlobDigest,
    })
}
```

The status value mapping:

  VTODO STATUS     `!task` field value
  ---------------- ---------------------
  (absent)         `todo`
  `NEEDS-ACTION`   `todo`
  `IN-PROCESS`     `in-process`
  `COMPLETED`      `done`
  `CANCELLED`      `cancelled`

Unknown values fall back to `todo` and log a warning at debug level.

**Decompile path (`CheckoutOne`)**: pull the same fields out of
`object.GetMetadata().GetIndex().GetFields()` and write them onto the
`caldav.Task` before PUT. Currently `CheckoutOne` doesn't read fields at all and
the existing decompile hardcodes `STATUS:NEEDS-ACTION`.

The reverse status mapping is the inverse of the table above. `priority` is
parsed back to `int`. `due` is passed through unchanged (CalDAV expects an iCal
datetime string; we don't reformat it).

### 4. Tests

**Unit tests (Go)** for the haustoria:

- `TestStatusValueRoundTrip` --- table-driven over the five VTODO STATUS values.
- `TestPriorityRoundTrip` --- `0`, `1`, `5`, `9`.
- `TestDuePassthrough` --- verbatim string passthrough.

**BATS integration tests** in `zz-tests_bats/current_version/`:

- Update existing 10 `caldav_*` tests to assert field projection in
  `dodder status` output. Drop `caldav_status_tags_*` tests entirely (the
  feature is removed).
- Add `caldav_round_trip_status_field`: VTODO with `STATUS:COMPLETED` → checkin
  → `dodder show` displays `status=done` → `dodder organize` to set
  `status=todo` → checkout → CalDAV server has `STATUS:NEEDS-ACTION`.
- Add `caldav_priority_and_due_round_trip`: same but for the other two fields.
- Add `genesis_includes_task_type`: fresh `dodder init` then
  `dodder show '!task:t'` produces a non-empty result with the field definitions
  visible.

**Fixture regeneration**: `dodder show :t` on a fresh-store BATS test will now
include `!task` alongside `!md`. Run `just test-bats-update-fixtures` and commit
the new `current_version` fixtures + `.fixtures.env`. The frozen v14 snapshot
stays untouched.

### 5. Live workspace migration (`~/workspaces/dodder-haustoria-caldav`)

After the PR merges, the live workspace needs:

1.  Edit `workspace/.dodder-workspace`: remove the three
    `[haustoria.calendars.*.status-tags]` blocks. (No replacement --- fields are
    projected unconditionally now.)
2.  The repo is at store V14, code at V15. No format migration is needed; V14
    reads fine on V15. Existing zettels (zero, in this case) keep working.
3.  `dodder status` should now show field columns (`status=todo` /
    `status=done`) on each untracked CalDAV resource.
4.  The dormant filtering issue noted in the project memory becomes moot:
    `dodder status '^status=done'` filters completed tasks at the doddish layer,
    no need to reach into the dormant index from `mike/haustoria_caldav`.

### 6. Docs

- Update `docs/features/0007-checkout-bridges.md` status: `exploring` →
  `experimental`. The promotion criteria ("CheckoutStore interface defined with
  Compile/Decompile methods; at least one concrete store passes a round-trip
  BATS test") are met by the existing CalDAV haustoria; this PR strengthens that
  with field round-tripping.
- Add a "superseded" addendum to
  `docs/plans/2026-04-05-vtodo-langlang-design.md` pointing at this plan.
- No new FDR for the `!task` type itself --- it's a built-in, not a feature.
  Mention it in the FDR-0007 update instead.

## Decisions to confirm

These are open until you comment on this file:

1.  **Status enum values**: `todo` / `in-process` / `done` / `cancelled`
    (matches the field-mutation work in `19c6f63`). Alternative: keep VTODO
    casing (`needs-action` / `in-process` / `completed` / `cancelled`).

2.  **`!chore` handling**: drop `!chore` entirely and have both `tasks` and
    `chores` calendars use `!task`. Alternative: ship `!task` and `!chore` as
    two separate built-in types with the same field set.

3.  **`ExcludeDefaultType` semantics**: gate `!task` under the existing flag
    alongside `!md`. Alternative: separate `ExcludeDefaultTaskType` flag.

4.  **`StatusTags` removal**: drop the field, the config key, and the tests in
    one PR. Alternative: leave as a deprecated no-op for one release.

5.  **`TypeBlobDigest` discovery**: one-shot `s.supplies.ReadPrimitiveQuery`
    lookup on `Initialize`. Alternative: explicit accessor on the
    `store_workspace.Supplies` struct
    (e.g. `GetBuiltinTypeBlobDigest("!task")`).

6.  **File extension on `!task`**: `task`, `taskpaper`, or omitted. The blob
    isn't used for anything (the haustoria reads VTODOs from CalDAV, not from
    files), so the extension only affects edit/checkout fallback paths.

## Implementation order

1.  Add `type_blobs.DefaultTaskType()` + unit test.
2.  Wire `prepareTaskType` into `local_working_copy/genesis.go`.
3.  Run `just test-bats-update-fixtures`, review and commit fixture diff.
4.  Update `mike/haustoria_caldav` compile path: drop `StatusTags`, add field
    projection. Drop `StatusTags` from `echo/workspace_config_blobs` config.
5.  Update `mike/haustoria_caldav` decompile path: read fields from metadata,
    write to VTODO.
6.  Update / add BATS tests in `zz-tests_bats/current_version/`.
7.  Update FDR-0007 status and add the supersede note to the !vtodo plan.
8.  Test the live workspace, update its `.dodder-workspace`.

## Risks

- **Field digest binding**: if the genesis-committed `!task` type blob digest
  drifts (e.g. someone changes `DefaultTaskType()` after the first commit on a
  given repo), existing field records on already-checked-in objects will have a
  stale `TypeBlobDigest`. This is the same risk that already exists for
  user-defined types and the field round-trip via organize, so no new mitigation
  is needed --- but worth noting in the FDR.

- **Workspace status output churn**: every BATS test that does `dodder status`
  in a haustoria workspace will now show field columns. The two-pass WRONG
  assertion strategy in `CLAUDE.md` should be used to capture the new output.

- **Forward compat with future built-in types**: this PR introduces the pattern
  of "genesis commits more than just `!md`". Future built-in types (`!file`,
  `!url`, `!event`?) follow the same template. No abstraction is needed yet ---
  three call sites is the threshold for extracting a `prepareBuiltinTypes`
  helper.
