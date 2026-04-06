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
- An `!actionable` abstract type or any composition / inheritance mechanism
  shared between `!task` and `!chore`. Both ship with duplicated field
  lists; the abstract type is captured as future work in §1a.
- Migration of existing repos. V14 → V15 was the only recent format bump and
  it's already shipped; this plan does not change persisted formats.
- Conflict resolution between dodder-side edits and CalDAV-side edits
  during sync. Last-write-wins for now; proper merge semantics block on
  #19.

## Design

### 1. `!task` type blob

`golf/type_blobs/main.go` gains a `DefaultTaskType()` function (and a sibling
`DefaultChoreType()` --- see §1a) returning a `TomlV2` with the following
content:

``` toml
file-extension = "toml"
vim-syntax-type = "toml"

[[fields]]
name = "status"
kind = "enum"
values = ["todo", "in-process", "done", "cancelled"]
default = "todo"

[[fields]]
name = "priority"
kind = "enum"
values = ["p0", "p1", "p2", "p3"]
default = "p0"

[[fields]]
name = "due"
kind = "string"

[fields-reader]
script = "yq -p toml -o json '{\"status\": .status, \"priority\": .priority, \"due\": .due}'"

[fields-writer]
script = "yq -p toml -o toml -i \".status = \\\"$DODDER_FIELD_status\\\" | .priority = \\\"$DODDER_FIELD_priority\\\" | .due = \\\"$DODDER_FIELD_due\\\"\" \"$DODDER_BLOB_PATH\""
```

The blob format for `!task` instances is TOML. Each instance's blob mirrors the
field values; the reader script projects them into `Metadata.Index.Fields` on
every commit, and the writer script projects edits back into the blob during
organize mutations.

Both scripts are mandatory. PR #100 probes proved that omitting either breaks
the round-trip: missing writer drops user edits and duplicates fields in
`dodder show` (issue #102), and missing reader+writer means fields don't persist
at all. See PR #100 commits `cfdb846` and `c5ebd34` for the full probe matrix.

`yq` is required at runtime (it's already a dependency for the existing fields
tests).

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

#### Approach: blob-canonical, reader/writer-driven

PR #100 probes (commits `cfdb846`, `c5ebd34`) established that the only
reliable way to round-trip fields through the commit cycle is via the
existing `[fields-reader]` / `[fields-writer]` script machinery declared on
the type blob. The haustoria does NOT set `Metadata.Index.Fields` directly.
Instead:

- **Compile path** (CalDAV → dodder): the haustoria builds a Go-side
  `{status, priority, due}` from the parsed VTODO, serializes it as a TOML
  blob, writes the blob to the blob store, and commits a `!task` object
  whose blob digest points at it. The reader script declared on `!task`
  (see §1) projects the fields out of the blob into
  `Metadata.Index.Fields` automatically during commit.

- **User edits** (`dodder organize`): standard fields-writer pipeline.
  Daughter has the edited field value, `tryWriteFields` runs the writer
  script, the script rewrites the TOML blob, the new blob digest replaces
  the old. `tryReadFields` re-projects the fields. Probe 12
  (`field_full_task_organize_mutate_one_of_three`) confirms this round-trip.

- **Decompile path** (dodder → CalDAV): the haustoria reads
  `Metadata.Index.Fields` (which the reader has populated from the blob)
  for status / priority / due, applies the inverse mappings (table below),
  and emits a VTODO via `CheckoutOne`.

#### What this avoids vs the original draft

- **No fresh `TypeBlobDigest` lookup needed.** The reader script handles
  digest stamping automatically when it runs during commit.
- **No new code paths in `papa/store`.** The existing reader/writer
  machinery does all the work.
- **No `s.supplies.ReadPrimitiveQuery` plumbing.** The haustoria doesn't
  need to know the type blob digest at all.
- **`dodder` is a source of truth between sync cycles.** User edits via
  organize survive haustoria re-syncs because the dodder blob has the
  canonical field values. Conflict resolution between dodder and CalDAV
  is out of scope for this PR (last-write-wins is acceptable until #19
  lands).

#### Compile path: VTODO → TOML blob

```go
// per task in queryCheckedOutForCalendar
blob := buildTaskTomlBlob(twm.Task)
if err := s.writeBlob(external, blob); err != nil {
    return errors.Wrapf(err, "write task blob for %s", twm.Task.UID)
}
// reader script runs during commit and projects fields into the index
```

`buildTaskTomlBlob` is a small Go function that emits TOML like:

```toml
status = "in-process"
priority = "p2"
due = "20260415T120000Z"
```

The status and priority values come from the mapping tables below.

#### Status mapping

| VTODO STATUS   | `!task` field value |
|----------------|---------------------|
| (absent)       | `todo`              |
| `NEEDS-ACTION` | `todo`              |
| `IN-PROCESS`   | `in-process`        |
| `COMPLETED`    | `done`              |
| `CANCELLED`    | `cancelled`         |

Unknown values fall back to `todo` and log a warning at debug level.

#### Decompile path: `Metadata.Index.Fields` → VTODO

`CheckoutOne` reads the three fields out of
`object.GetMetadata().GetIndex().GetFields()`, applies the inverse status
and priority mappings, and emits a VTODO. The current decompile hardcodes
`STATUS:NEEDS-ACTION` and ignores priority/due — that goes away.

The reverse status mapping is the inverse of the table above. Priority is
mapped back through the §1 priority table (`p0`→0, `p1`→9, `p2`→5,
`p3`→1). `due` is passed through unchanged (CalDAV expects an iCal
datetime string; the haustoria does not reformat it).

### 4. Tests

**Unit tests (Go)** for the haustoria:

- `TestStatusValueRoundTrip` --- table-driven over the five VTODO STATUS values.
- `TestPriorityValueRoundTrip` --- table-driven `p0`/`p1`/`p2`/`p3` ↔
  `0`/`9`/`5`/`1`, plus out-of-band PRIORITY values (`2`, `3`, `7`) bucketed to
  nearest.
- `TestDuePassthrough` --- verbatim string passthrough.

**Already-landed probe tests** (PR #100, `cfdb846` and `c5ebd34`,
`zz-tests_bats/current_version/fields.bats`):

- `field_persists_without_any_scripts` — confirms fields are dropped when
  the type has no scripts. Filed as issue #101 for follow-up.
- `field_persists_with_reader_only_no_writer` — confirms reader-only mode
  drops user edits and duplicates fields in `show` output. Filed as
  issue #102.
- `field_full_task_three_fields_from_blob` — confirms the haustoria
  compile path works: TOML blob with three fields baked in projects all
  three via the reader script.
- `field_full_task_organize_mutate_one_of_three` — confirms
  organize-driven field mutation round-trips through the writer script
  while preserving untouched fields.
- `field_full_task_organize_from_empty_blob` — documents the empty-blob
  failure mode (also issue #101).

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

## Resolved verification (PR #100 probes)

The original draft assumed fields could be set directly on
`Metadata.Index.Fields` and would persist via the binary stream-index codec
without scripts. **PR #100 probes proved that wrong** — see the five
`field_*` probe tests in `zz-tests_bats/current_version/fields.bats` landed
in commits `cfdb846` and `c5ebd34`.

Both `tryWriteFields` and `tryReadFields` early-return on missing scripts,
but the daughter fields do NOT survive the early-return path through to the
binary codec — they are silently dropped at commit time.

The plan now uses `[fields-reader]` + `[fields-writer]` scripts on `!task`
(see §1) so the haustoria can rely on the existing fields machinery instead
of bypassing it. The previously-required
`TestFieldsPersistThroughCommitWithoutScripts` is unnecessary; probes 11 and
12 already cover the supported flow.

Two follow-up bugs were filed against the fields infrastructure:

- **#101** — `fields-writer` should support blob-less first writes. The
  haustoria sidesteps this by always writing a non-empty TOML blob; the bug
  remains worth fixing for future programmatic field-set callers.
- **#102** — reader-only mode silently drops user edits AND duplicates fields
  in `show` output. Independent of this PR but exposed by the probes.

## Implementation order

1.  Add `type_blobs.DefaultTaskType()` + `DefaultChoreType()` (with the
    reader and writer scripts from §1) + unit tests.
2.  Wire `prepareBuiltinActionableTypes` into `local_working_copy/genesis.go`
    and add the `BigBang.IncludeBuiltinActionableTypes` flag.
3.  Add `genesis_opt_in_actionable_types` BATS test for the genesis path.
4.  Update `mike/haustoria_caldav` compile path: drop `StatusTags`, build
    TOML blob from VTODO via `buildTaskTomlBlob`, write to blob store, commit
    with the blob digest. Drop `StatusTags` from `echo/workspace_config_blobs`
    config (regenerate tommy codegen).
5.  Update `mike/haustoria_caldav` decompile path: read fields from
    `Metadata.Index.Fields`, apply inverse mappings, write to VTODO.
6.  Add Go round-trip tests + BATS round-trip tests.
7.  Drop existing `caldav_status_tags_*` tests.
8.  Update FDR-0007 status and add the supersede note to the !vtodo plan.
9.  Test the live workspace, re-init with the new flag.

## Risks

- **`yq` runtime dependency**: the reader/writer scripts shell out to `yq`
  on every commit. Already a dependency for the existing fields tests, so
  no new external requirement, but worth noting that `!task` won't work in
  environments without it. Three OS process invocations per object on
  commit (writer + reader, plus reader on subsequent reads). For 1100-task
  workspaces this is a few thousand `yq` invocations on a full sync —
  acceptable for an interactive command, possibly slow enough to want
  batching later.

- **Field digest drift**: even with the reader script auto-stamping the
  current type blob digest, if the user re-commits the
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
