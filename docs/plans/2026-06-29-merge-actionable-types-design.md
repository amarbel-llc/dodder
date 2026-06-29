# Merge actionable types (`!task` / `!chore` / `!habit`) — design

**Date:** 2026-06-29
**Status:** design approved; implementation plan pending
**Related:** FDR-0017 (type-defined field index), the blob-backed pandoc
internal-formatting plan (`docs/plans/2026-03-27-pandoc-internal-formatting-plan.md`),
and the forthcoming `dang` scoped-typefulness FDR (tracked separately).

## Problem

Two divergent definitions of the actionable types exist and need merging into a
single canonical set, "taking the best from both":

- **the personal repo set** (a real user repo, `!toml-type-v1`): pandoc/markdown actionable
  types with a shared `actionable-common` lua hook object (`cotton/horsea`),
  rich pandoc formatters on `!task`. State is modeled as **separate types**
  (`!task-done`, `!task-cancelled` = `archived=true`, `!task-in_progress`).
  Urgency, scheduling, and archival are modeled as **tags**:
  - urgency → `urgency-0_hour` … `urgency-6_year` (a 7-level time-pressure
    scale, each mapped to a seconds threshold in the hooks),
  - scheduling → `w-YYYY-MM-DD` date tags + `today`, normalized by lua hooks
    (collapse to the latest date, auto-stamp today on archive, strip stale
    `today`),
  - archival → `zz-archive*` tag.

- **default set** (the dodder builtin, `!toml-type-v2`): structured
  `status` / `priority` / `due` enum **fields** with yq `fields-reader` /
  `fields-writer` scripts. No formatters, no hooks. State is a **field**.
  `!habit` absent.

The two encode the *same concepts* with different machinery: the personal repo spreads
state/urgency/scheduling/archival across types + tags + lua hooks; v2 captures a
subset as structured fields. The merge reconciles these per-dimension.

`TomlV2` is a **superset** struct (`go/internal/alfa/type_blobs/toml_v2.go`): it
already carries `Fields` + `FieldsReader`/`FieldsWriter` *and*
`Formatters` + `Hooks` + `UTIGroups` + `References`. So the unified "best of
both" is expressible as a single `!toml-type-v2` per kind — no new schema.

## Goals

- One canonical `!task` / `!chore` / `!habit`, each `!toml-type-v2`, combining
  v2's structured fields with the personal repo's formatters + hooks + richer semantics.
- Target **both** dodder's Go builtins (`type_blobs.DefaultTaskType` /
  `DefaultChoreType` + a new `DefaultHabitType`) and the user's repo, with the
  eventual home being a **public dodder.net type repo** (so the types must be
  self-contained and portable).

## Non-goals (deferred)

- The `dang` scoped-typefulness mechanism itself (its own FDR — see Dependencies).
- Migrating the user's *existing* the personal repo task data (urgency/date tags + state
  types → fields). Existing v1-typed objects keep working; bulk migration is a
  later phase.
- dodder.net publishing.
- `!habit`-specific structure (streaks/consistency) — neither source set had it;
  adding it now would be new scope, not a merge.

## Design

### Field schema

All three types are `!toml-type-v2` with a structured field set. The
state/urgency/scheduling dimensions that the personal repo spread across types + tags + hooks
collapse into fields.

Shared by `!task` / `!chore` / `!habit`:

| field      | kind                         | values / shape                                                        | default | replaces (the personal repo)                         |
|------------|------------------------------|-----------------------------------------------------------------------|---------|-------------------------------------------|
| `status`   | enum                         | `todo`, `in_progress`, `done`, `cancelled`                            | `todo`  | the `!task-done/-cancelled/-in_progress` state types |
| `urgency`  | enum                         | 7-level time-pressure: `0_hour`, `1_day`, `2_week`, `3_month`, `4_quarter`, `5_episode`, `6_year` | unset   | the `urgency-N_*` tags                    |
| `priority` | enum                         | importance `p0`..`p3` (orthogonal 2nd axis — Eisenhower)              | unset   | v2's `priority`                           |
| `due`      | date                         | scheduled / next-due `YYYY-MM-DD`                                      | unset   | the `w-YYYY-MM-DD` date tags              |
| `body`     | string (multiline, `dang`-typed) | the prose, declaring its sub-type via a `dang` identifier (see below) | —       | the markdown blob                         |

`!chore` / `!habit` add:

| field        | kind          | shape                                              | notes                                                          |
|--------------|---------------|----------------------------------------------------|----------------------------------------------------------------|
| `recurrence` | string / enum | cadence (ISO-8601 duration `P1W`, or enum)         | drives the "advance `due`, reset `todo`" hook; `!task` omits it |

Two orthogonal priority axes are intentional: `urgency` = "how soon" (the personal repo's
time-pressure scale), `priority` = "how much it matters" (v2). A task can be
urgent-not-important or vice versa.

### Blob shape & the `dang`-typed body

The object blob is **TOML** (structured), with the prose in a multiline string
field. Example `!task` object:

```toml
status   = "in_progress"
urgency  = "2_week"
priority = "p1"
due      = "2026-07-15"

body = '''
#!dang md
# Ship the unified actionable types

Brainstorm done; now planning.
'''
```

- `fields-reader` (yq) extracts `status`/`urgency`/`priority`/`due`(+`recurrence`)
  → JSON, feeding the type-defined field index (FDR-0017) so field-filtered
  queries work.
- `fields-writer` (yq) writes `DODDER_FIELD_*` back into the TOML on checkin.
- `body` is `dang`-typed (see next).

### `dang` (scoped, pinned typefulness)

A multiline string field carries its own scoped typefulness via `dang`:

- The referenceable types are **hyphence-pinned objects** (content-addressed
  identity — not a mutable name lookup).
- The object **metadata carries a binding table**: a short **identifier** ↔ the
  pinned type object (type name + hyphence pin).
- The body's first line is a `dang` shebang that uses the **identifier**
  (`#!dang md`), resolved through the metadata binding to the specific pinned
  `!md`.

This makes the body's typefulness both *scoped* (the identifier means only what
the metadata binds) and *pinned* (stable/verifiable; survives renames). It is a
stronger form of an import table.

`dang` is its own mechanism and its own FDR. The type-merge `body` field
*depends* on it. Phasing: until `dang` lands, `body` ships either as a plain
string or with a minimal binding convention; the formatter (below) extracts the
body and strips the `dang` line.

### Formatters — blob-backed pandoc

The unified types use the **blob-backed** pandoc mechanism (the modern `!md`
type's approach), not inline host scripts — so a fresh clone or dodder.net pull
formats with no host setup:

- The type carries blob references to pandoc lua filters + defaults, materialized
  to a tmpdir (`DODDER_BLOB_TREE`) by the existing materializer.
- the personal repo's formatter set (`text`/`html`/`html-gdoc`/`pdf-beamer`) is preserved.
- Each formatter pipeline: extract `body` (yq) → strip the `dang` line → pandoc
  with the blob-backed defaults.

### Hooks — self-contained behavior engine

The `cotton/horsea` lua becomes a **self-contained, blob-backed
`actionable-common.lua` tool blob** (materialized like the pandoc filters), *not*
a reference to the user's private object. All three types carry a blob reference
to it and `require` the materialized copy.

Behavior re-expressed for the field model (the personal repo's date-tag collapse/today
machinery is replaced by explicit, status-driven transitions in `on_format` /
`on_pre_commit`):

- `status = cancelled` → archive (dormant), all three.
- `status = done`:
  - `!task` (once) → **auto-archive** (dormant).
  - `!chore` / `!habit` (recurring) → **advance `due` by `recurrence`, reset
    `status = todo`** (the recurrence engine).
- `status = todo` / `in_progress` → active.
- Field normalization: default `urgency`, guard `due` on recurring objects, etc.

Archival uses dodder's tag-driven dormancy (the hook applies an archive tag the
repo's dormant config recognizes — successor to the personal repo's `zz-archive`).

### Per-type differentiation

`!chore` and `!habit` are **structurally identical** (same fields + recurrence
engine); they differ only in default cadence and description. `!task` omits
`recurrence` and auto-archives on `done`. `!habit`-specific structure
(streaks/consistency) is deferred.

| | `!task` | `!chore` | `!habit` |
|---|---|---|---|
| meaning | "must be done once" | "accomplished periodically" | "should be done regularly" |
| `recurrence` | — | yes | yes |
| `done` → | auto-archive | advance `due`, reset `todo` | advance `due`, reset `todo` |
| default cadence | n/a | longer (e.g. monthly) | tighter (e.g. daily/weekly) |

## Targets & phasing

1. **dodder builtins:** `type_blobs.DefaultTaskType()` / `DefaultChoreType()` +
   new `DefaultHabitType()` return the unified TomlV2; genesis
   (`-include-builtin-actionable-types`) also writes the blob-backed
   `actionable-common.lua` + pandoc tool blobs and the type→tool blob references.
2. **User repo:** the unified v2 types land alongside the existing v1 ones
   (different type-blob versions coexist; v1 stays decodable).
3. **dodder.net:** canonical public hosting — later phase.

- **Phase 1 (this work):** unified types as builtins + genesis, blob-backed
  formatters + hooks, the field schema, `dang`-as-convention for `body`. Ships
  behind the existing opt-in genesis flag.
- **Phase 2 (deferred):** the `dang` FDR + first-class scoped-type integration.
- **Phase 3 (deferred):** dodder.net publishing + migrating existing the personal repo data.

## Rollback

Additive and low-risk:

- **Dual-architecture:** the unified types ship via the opt-in
  `-include-builtin-actionable-types` genesis flag; default genesis and all
  existing v1-typed objects are untouched. The v1 type-blob codec is retained
  (old objects still decode) — no wire-format break.
- **Promotion criterion:** the merged types drive the user's repo for a few weeks
  with no capability gap vs the v1 set before v1 deprecation / publishing.
- **Rollback procedure:** single revert of the `DefaultTaskType/Chore/Habit` Go
  change; existing objects unaffected.

## Tuning levers

- **Urgency granularity** (7 levels). Signal to change: levels going unused, or
  needing finer/coarser steps.
- **Recurrence representation** (ISO-8601 duration vs enum) + chore/habit default
  cadences. Signal: awkwardness expressing real cadences.
- **`done` → archive policy** (currently auto-archive on `done` for `!task`).
  Signal: wanting recent completions visible → revisit a grace-window variant.
- **`dang` convention vs first-class** (Phase 1 vs Phase 2). Signal: the `dang`
  FDR landing.
- **Two-axis urgency + priority.** Signal: one axis going consistently unused →
  collapse to one.

## Dependencies & risks

- **`dang` FDR** — the `body` field's scoped typefulness depends on the
  metadata-binding + pinned-identifier mechanism. Phase 1 uses a convention;
  the FDR formalizes it.
- **Type-defined field index (FDR-0017)** must support enum fields for
  `status`/`urgency`/`priority` querying (it already did for v2
  `status`/`priority`).
- **Existing-data migration** (urgency/date tags + state types → fields) is real
  but deferred to Phase 3; the user has live data tagged the old way.

## Open questions

- Exact `urgency` enum value spelling (`0_hour` vs `hour` vs `urgency-0_hour`) —
  pick during planning for sort-friendliness + readability.
- Whether the `body` `dang` binding lives per-object or is provided as a default
  by the type blob (so every actionable object inherits the `md` binding without
  repeating it). Likely the latter for ergonomics; confirm against the `dang`
  FDR.
- Recurrence advance semantics around missed periods (advance by one period vs
  catch-up to the next future period).
