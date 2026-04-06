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

`golf/type_blobs/main.go` gains a `DefaultTaskType()` function (and a sibling
`DefaultChoreType()` --- see §1a) returning a `TomlV2` with the following
content:

``` toml
file-extension = "toml"

[[fields]]
name = "status"
kind = "enum"
values = ["todo", "in-process", "done", "cancelled"]
default = "todo"

[[fields]]
name = "priority"
kind = "enum"
values = ["p0", "p1", "p2", "p3"]

[[fields]]
name = "due"
kind = "string"
```

The blob format for `!task` instances is TOML (file extension `toml`); the
fields are stored as TOML key/value pairs in the blob and as records in the
binary stream index (see §3 for how the haustoria sets them).

Status enum values match the field-mutation work in `19c6f63` ("use 'done'
instead of 'completed'"). The mapping to/from VTODO STATUS is the haustoria's
job (see §3 below).

`priority` is an enum with values `p0` / `p1` / `p2` / `p3` corresponding to the
tasks.org convention (none / `!` / `!!` / `!!!`). The numeric VTODO PRIORITY
mapping is the haustoria's job:

  VTODO PRIORITY   `!task` priority   tasks.org
  ---------------- ------------------ -----------
  0 (or absent)    `p0`               (none)
  9                `p1`               `!`
  5                `p2`               `!!`
  1                `p3`               `!!!`

Other PRIORITY values fall back to the closest bucket on read; on write the
haustoria emits the canonical numeric value above.

### 1a. `!chore` type blob

`!chore` ships as a separate built-in type with the **same field set** as
`!task`. There is no inheritance / mixin mechanism in dodder yet --- the field
list is duplicated in both `DefaultTaskType()` and `DefaultChoreType()`. Future
work: explore an `!actionable` abstract type that both `!task` and `!chore`
compose against, replacing the duplication.

The reason both ship together: the live workspace already has separate `tasks`
and `chores` calendars with semantically distinct meanings (one-shot work vs
recurring habits), and CalDAV doesn't carry that distinction in any field.
Calendar-to-type binding stays a workspace config concern.

### 2. Genesis: opt-in `!task` and `!chore` built-in types

`local_working_copy/genesis.go` gains a single helper next to
`prepareDefaultType`:

``` go
func (local *Repo) prepareBuiltinActionableTypes(
    bigBang env_repo.BigBang,
    builder *import_plan.Builder,
) (err error)
```

Behavior:

- For each of `!task` and `!chore`:
  - Allocates the `id.TypeStruct`.
  - Calls the corresponding `type_blobs.DefaultTaskType()` /
    `DefaultChoreType()` constructor.
  - Saves the blob via the typed blob store.
  - Constructs a `sku.Transacted` for the type object, sets its blob digest and
    type (`!toml-type-v2`), and adds it to the import plan builder.
- Does **not** return digests; the haustoria looks them up fresh on each commit
  cycle (see §3).

**Opt-in via a new `BigBang.IncludeBuiltinActionableTypes` flag.** Default is
`false` --- `dodder init` does NOT commit `!task` / `!chore` unless the user
passes the flag. Once the types stabilize and the haustoria flow is proven, the
flag can flip to opt-out (or merge into `ExcludeDefaultType`), but for the
bootstrap phase the existing default-type semantics stay unchanged.

Rationale for opt-in: the existing `dodder init` test fleet, fixture generation,
and downstream tooling all assume the only built-in type is `!md`. Opt-in keeps
this PR's blast radius minimal --- only fresh repos that explicitly request
actionable types pay the fixture-regeneration and behavior-change cost. The live
haustoria workspace will be one of the first opt-in users.

The flag is exposed on the genesis command (and madder's init for now). No
existing flag is reused.

### 3. Haustoria: VTODO ↔ `!task` field mapping

`mike/haustoria_caldav/main.go` changes:

**Drop `CalendarMapping.StatusTags`** and the
`[haustoria.calendars.*.status-tags]` config keys in
`echo/workspace_config_blobs`. Tommy codegen for the V2 workspace config is
regenerated. The status-tag tests are deleted (see §4).

**Look up the `!task` (or `!chore`) type blob digest fresh on every compile and
decompile cycle.** Do NOT cache the digest at `Initialize`. The haustoria runs
`s.supplies.ReadPrimitiveQuery` for `task:t` (or `chore:t`, keyed off
`cal.TypeId`) inside `queryCheckedOutForCalendar` and `CheckoutOne`, captures
the current type object's blob digest, and stamps it on each `fields.Field` it
constructs. Rationale: the user might re-commit the type object with new field
definitions; a cached digest would point at a stale blob and break field
round-tripping for newly-set fields.

If the type is missing (e.g. the workspace was init'd without
`-include-builtin-actionable-types` and no user-defined `!task` exists), the
haustoria fails the query with a clear error pointing the user at
`dodder init -include-builtin-actionable-types` or at creating the type
manually.

#### Field-writer / field-reader interaction

The fields plumbing in `papa/store/{field_writer,field_reader}.go` runs
script-based projection between the blob and the index. Both functions
early-return if the type blob has no `[fields-reader]` / `[fields-writer]`
script configured (verified at `field_writer.go:110-112` and
`field_reader.go:74-77`).

`!task` and `!chore` ship **without** reader/writer scripts. Consequence:

1.  The haustoria sets `daughter.Metadata.Index.Fields["status"]` (etc.) with
    `TypeBlobDigest` populated.
2.  The commit cycle calls `tryWriteFields` --- early-returns at
    `fieldsWriter == nil`. Daughter fields stay intact.
3.  The commit cycle calls `tryReadFields` --- early-returns at
    `fieldsReader == nil`. Daughter fields still intact.
4.  The binary stream-index encoder serializes `Metadata.Index.Fields` verbatim
    (this is the codec change from `19c6f63`).
5.  On read back, fields appear in the index without ever touching the blob.

The blob is therefore a **decorative artifact** for `!task` --- it can be empty
or hold a TOML mirror of the fields, but neither dodder nor the haustoria reads
it for field values. The canonical source is CalDAV (via the haustoria) and the
cached projection is the binary stream index. **Open verification task:**
confirm the binary codec actually persists `Metadata.Index.Fields` even when no
script ran (the codec was added in `19c6f63` and tests pass for organize-driven
mutation, so this should hold, but the haustoria path is a new entry point and
warrants a unit test before relying on it).

**Compile path (`queryCheckedOutForCalendar`)**: replace the `StatusTags` block
with direct field projection. The type blob digest is fetched fresh per calendar
(the result can be reused for all tasks in the same calendar because the type
object can't change mid-query).

``` go
typeBlobDigest, err := s.lookupTypeBlobDigest(cal.TypeId) // fresh per calendar
if err != nil {
    return errors.Wrapf(err, "lookup %s type blob", cal.TypeId)
}

// ... per task ...

metadata := external.GetMetadataMutable()
indexFields := metadata.GetIndexMutable().GetFieldsMutable()

indexFields.Add(fields.Field{
    Key:            "status",
    Value:          mapVTODOStatusToFieldValue(twm.Task.Status), // "todo" default
    TypeBlobDigest: typeBlobDigest,
})

indexFields.Add(fields.Field{
    Key:            "priority",
    Value:          mapVTODOPriorityToFieldValue(twm.Task.Priority), // "p0" default
    TypeBlobDigest: typeBlobDigest,
})

if twm.Task.Due != "" {
    indexFields.Add(fields.Field{
        Key:            "due",
        Value:          twm.Task.Due,
        TypeBlobDigest: typeBlobDigest,
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
mapped back through the §1 priority table (`p0`→0, `p1`→9, `p2`→5, `p3`→1).
`due` is passed through unchanged (CalDAV expects an iCal datetime string; we
don't reformat it).

### 4. Tests

**Unit tests (Go)** for the haustoria:

- `TestStatusValueRoundTrip` --- table-driven over the five VTODO STATUS values.
- `TestPriorityValueRoundTrip` --- table-driven `p0`/`p1`/`p2`/`p3` ↔
  `0`/`9`/`5`/`1`, plus out-of-band PRIORITY values (`2`, `3`, `7`) bucketed to
  nearest.
- `TestDuePassthrough` --- verbatim string passthrough.

**Critical-path Go test** (must run before BATS):

- `TestFieldsPersistThroughCommitWithoutScripts` --- in
  `papa/store/field_writer_test.go` (or a new `field_persistence_test.go`).
  Commits a daughter object with `Metadata.Index.Fields` populated but no
  `[fields-writer]` / `[fields-reader]` scripts on the type blob. Reads the
  object back from the stream index and asserts the fields survived. This
  validates the assumption documented in §3 "Field-writer / field-reader
  interaction" before the haustoria starts depending on it.

**BATS integration tests** in `zz-tests_bats/current_version/`:

- Update existing 10 `caldav_*` tests to assert field projection in
  `dodder status` output. Drop `caldav_status_tags_*` tests entirely (the
  feature is removed).
- Add `caldav_round_trip_status_field`: VTODO with `STATUS:COMPLETED` → checkin
  → `dodder show` displays `status=done` → `dodder organize` to set
  `status=todo` → checkout → CalDAV server has `STATUS:NEEDS-ACTION`.
- Add `caldav_round_trip_priority_field`: VTODO with `PRIORITY:1` → checkin →
  `priority=p3` → organize to `p1` → checkout → `PRIORITY:9`.
- Add `caldav_round_trip_due_field`: passthrough verification.
- Add `caldav_fresh_type_lookup`: re-commit the `!task` type with a new field
  definition between two `dodder status` runs; verify the haustoria picks up the
  new digest fresh on the second run.
- Add `genesis_opt_in_actionable_types`: fresh
  `dodder init -include-builtin-actionable-types` then `dodder show '!task:t'`
  and `dodder show '!chore:t'` both return non-empty results with the field
  definitions visible. Without the flag, both return empty.

**Fixture regeneration**: only the new opt-in genesis test creates a repo with
`!task`/`!chore` --- the existing fresh-store fixtures stay unchanged because
the default is opt-out. The opt-in path gets its own `.fixtures.env` entries.
The frozen v14 snapshot stays untouched.

### 5. Live workspace migration (`~/workspaces/dodder-haustoria-caldav`)

The existing workspace is at store V14 with no zettels checked in. The cleanest
path is to **re-init the repo** with the new opt-in flag rather than
retrofitting:

1.  Back up `workspace/.dodder-workspace` (the haustoria config block is the
    only thing worth keeping).
2.  Wipe `~/workspaces/dodder-haustoria-caldav/.dodder/`.
3.  Re-run `dodder init -include-builtin-actionable-types` against the parent
    dir.
4.  Restore the workspace config, removing the three
    `[haustoria.calendars.*.status-tags]` blocks (no replacement --- fields are
    projected automatically).
5.  `dodder status` should show field columns (`status=todo` / `status=done`,
    `priority=p0`...`p3`) on each untracked CalDAV resource.

The dormant filtering issue noted in the project memory becomes moot:
`dodder status '^status=done'` filters completed tasks at the doddish layer, no
need to reach into the dormant index from `mike/haustoria_caldav`.

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

## Decisions (resolved from review)

Resolved in PR #100 review comments:

1.  **Status enum values**: `todo` / `in-process` / `done` / `cancelled`. The
    haustoria maps these to/from CalDAV STATUS values (table in §3).

2.  **Priority**: enum with values `p0` / `p1` / `p2` / `p3` mapped to the
    tasks.org convention (none / `!` / `!!` / `!!!`) and VTODO PRIORITY integers
    `0` / `9` / `5` / `1` (table in §1).

3.  **`!chore`**: ships as a separate built-in type alongside `!task`, with the
    same field set. Future: explore an `!actionable` abstract type that both
    `!task` and `!chore` compose against (less inheritance, more composition).

4.  **Genesis flag**: new `BigBang.IncludeBuiltinActionableTypes`, default
    `false`. Opt-in for now; can flip to opt-out once the types stabilize. No
    reuse of `ExcludeDefaultType`.

5.  **`StatusTags` removal**: drop the field, the config keys, and the tests in
    one PR. The live workspace is the only consumer.

6.  **`TypeBlobDigest` discovery**: fresh lookup on every compile and decompile
    cycle (per calendar, not per task). No caching.

7.  **File extension on `!task` / `!chore`**: `toml`. Instances are stored as
    TOML blobs that mirror the field values. The blob is decorative for the
    haustoria flow (the canonical source is CalDAV); for direct
    `dodder new !task` use, the user edits a TOML file.

## Open verification before implementation

1.  **Field codec round-trip without scripts**: `!task` ships without
    `[fields-reader]` / `[fields-writer]` script configs. The plan assumes that
    fields set directly on `Metadata.Index.Fields` (with TypeBlobDigest
    populated) are persisted by the binary stream-index codec from `19c6f63` and
    read back intact on the next query. Confirmed by reading
    `field_writer.go:110-112` and `field_reader.go:74-77` (both early-return
    when no script is configured), but the assumption is gated on a Go unit test
    (`TestFieldsPersistThroughCommitWithoutScripts`, see §4) that runs BEFORE
    any haustoria changes.

2.  **`s.supplies.ReadPrimitiveQuery` for type lookup**: confirm the primitive
    query API can find a type object by `task:t` predicate from inside the
    haustoria. If not, exposing a small accessor on the supplies struct
    (e.g. `GetBuiltinTypeBlobDigest("!task")`).

## Implementation order

1.  Land `TestFieldsPersistThroughCommitWithoutScripts` first --- validates the
    core assumption before any haustoria work.
2.  Add `type_blobs.DefaultTaskType()` + `DefaultChoreType()` + unit tests.
3.  Wire `prepareBuiltinActionableTypes` into `local_working_copy/genesis.go`
    and add the `BigBang.IncludeBuiltinActionableTypes` flag.
4.  Add `genesis_opt_in_actionable_types` BATS test for the genesis path.
5.  Update `mike/haustoria_caldav`: fresh-lookup helper, drop `StatusTags`,
    compile path field projection. Drop `StatusTags` from
    `echo/workspace_config_blobs` config (regenerate tommy codegen).
6.  Update `mike/haustoria_caldav` decompile path: read fields from metadata,
    write to VTODO.
7.  Add Go round-trip tests + BATS round-trip tests.
8.  Drop existing `caldav_status_tags_*` tests.
9.  Update FDR-0007 status and add the supersede note to the !vtodo plan.
10. Test the live workspace, re-init with the new flag.

## Risks

- **Field codec assumption**: if `TestFieldsPersistThroughCommitWithoutScripts`
  fails, the whole approach needs rethinking --- either ship a `fields-writer`
  script for `!task` (a small TOML projector), or add a "field-only" commit path
  that bypasses the script gate. The test runs first to surface this fast.

- **Field digest drift**: even with fresh lookup, if the user re-commits the
  `!task` type with incompatible field defs (say removing the `status` field),
  existing index records will have a stale `TypeBlobDigest` and may not resolve
  correctly on read. Same risk class as user-defined types --- no new
  mitigation, but worth a note in the FDR.

- **Workspace status output churn**: every BATS test that does `dodder status`
  in a haustoria workspace will now show field columns. The two-pass WRONG
  assertion strategy in `CLAUDE.md` should be used to capture the new output.

- **Forward compat with future built-in types**: this PR introduces the pattern
  of "genesis commits more than just `!md`". Future built-in types (`!file`,
  `!url`, `!event`?) follow the same template. No abstraction is needed yet ---
  three call sites is the threshold for extracting a `prepareBuiltinTypes`
  helper.
