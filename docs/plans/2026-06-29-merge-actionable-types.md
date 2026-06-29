# Merge actionable types (`!task` / `!chore` / `!habit`) — Phase-1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use eng:subagent-driven-development to
> implement this plan task-by-task.

**Goal:** Extend the existing built-in `!task` / `!chore` v2 types (and add
`!habit`) with the personal repo's richer semantics — a `urgency` field, a recurrence
field on the recurring kinds, a `dang`-typed markdown `body` rendered by
blob-backed pandoc, and self-contained per-state/recurrence hooks — taking the
best of both source sets.

**Architecture:** All three are `!toml-type-v2` (the superset struct already
carries `Fields` + `FieldsReader`/`FieldsWriter` *and* `Formatters` + `Hooks` +
`References`). We extend `type_blobs` (`go/internal/alfa/type_blobs/main.go`,
re-exported via `golf/type_blobs`) and `prepareBuiltinActionableTypes`
(`go/internal/romeo/local_working_copy/genesis.go`), mirroring the already-built
blob-backed pandoc machinery (`genesis_pandoc_tools.go`,
`blob_tree_materializer.go`) that `!md` uses. Everything stays behind the
existing opt-in `-include-builtin-actionable-types` flag.

**Tech Stack:** Go (tier alfa/golf/romeo/foxtrot/tango), `yq` (fields
reader/writer), pandoc (blob-backed formatter), gopher-lua (hooks), BATS.

**Rollback:** Purely additive. New fields/formatters/hooks ship only on the
opt-in built-in types; `!md` and existing repos are untouched; the v1 type-blob
codec is retained. Single-revert of the `type_blobs` + genesis changes.

**Design ref:** `docs/plans/2026-06-29-merge-actionable-types-design.md`.
**Prior art that landed the current types:**
`docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md`.

---

## Scope

**In (Phase 1):**

- Part A — structured fields + blob-backed pandoc formatter on the `body`
  (well-understood, low-risk).
- Part B — per-state + recurrence hooks (investigation-gated: the lua hook API's
  access to `Index.Fields` + archival/blob mutation is unverified).

**Deferred (not this plan):**

- `dang` formalization (its own FDR — task #19). Phase 1 treats `dang` as a
  first-line convention the formatter strips.
- Richer formatter outputs (`html` / `html-gdoc` / `pdf-beamer`) — additive,
  add later without rework.
- Migrating existing the personal repo data (urgency/date tags + state types → fields).
- Haustoria (`mike/haustoria_caldav`) emitting `urgency`/`recurrence` — the
  reader tolerates their absence, so the CalDAV path keeps working unchanged;
  mapping VTODO → the new fields is a follow-up.
- dodder.net publishing.

## Key facts (verified in the worktree)

- `type_blobs.DefaultTaskType()` / `DefaultChoreType()` already return
  `!toml-type-v2` with `actionableFields()` = `status`/`priority`/`due` +
  `actionableFieldsReader()`/`actionableFieldsWriter()` (yq). No urgency, body,
  recurrence, formatters, or hooks.
  (`go/internal/alfa/type_blobs/main.go:44-117`.)
- Field kinds (`go/internal/0/fields/main.go:20-43`): `string`, `enum`, `bool`,
  `u32`, `s32`, `list<string>`. **No `date` kind** — `due`/`recurrence` are
  `string`. `urgency`/`priority`/`status` are `enum`.
- `!md` gets its pandoc formatter (`DefaultWithPandocFormatter()`) + three tool
  blob references (`filters/dodder-common.lua`, `filters/dodder-edit.lua`,
  `defaults/dodder-edit.yaml`) in `prepareDefaultType` behind
  `bigBang.IncludeDefaultPandocTools`
  (`go/internal/romeo/local_working_copy/genesis.go:176-244`). The tool blobs
  are written by `prepareToolBlobs()` (returns `toolBlobDigests`) and attached
  via `addToolBlobReference()` (`genesis_pandoc_tools.go`). `toolBlobDigests`
  is created early in the genesis flow and passed into `prepareDefaultType`.
- `prepareBuiltinActionableTypes` (`genesis.go:251-303`) currently commits only
  `!task` + `!chore`, takes **no** `toolBlobDigests`, and attaches **no**
  formatter refs.
- Flag wiring: `BigBang.IncludeBuiltinActionableTypes`
  (`go/internal/foxtrot/env_repo/big_bang.go:41`), registered as
  `include-builtin-actionable-types`
  (`go/internal/tango/command_components_dodder/genesis.go:93`).
- The materializer exposes the blob tree as `$DODDER_BLOB_TREE`; formatters
  reference `--data-dir="$DODDER_BLOB_TREE"`.

---

# Part A — Fields + blob-backed formatter

### Task A1: Add the `urgency` field to the shared actionable field set

**Promotion criteria:** N/A (additive).

**Files:**
- Modify: `go/internal/alfa/type_blobs/main.go:49-68` (`actionableFields`)
- Modify: `go/internal/alfa/type_blobs/main.go:73-87` (reader/writer scripts)
- Test: `go/internal/alfa/type_blobs/main_test.go`

**Step 1: Write the failing test.** In `main_test.go`, extend the
`DefaultTaskType` assertions to require an `urgency` enum field with the
7-level scale and to require the reader/writer scripts mention `urgency`:

```go
func TestDefaultTaskTypeHasUrgency(t1 *testing.T) {
	t := ui.MakeT(t1)
	blob := DefaultTaskType()

	var urgency *FieldDefinition
	for i := range blob.Fields {
		if blob.Fields[i].Name == "urgency" {
			urgency = &blob.Fields[i]
		}
	}
	if urgency == nil {
		t.Fatalf("expected urgency field, got %#v", blob.Fields)
	}
	if urgency.Kind != "enum" {
		t.Errorf("urgency kind = %q, want enum", urgency.Kind)
	}
	want := []string{"0_hour", "1_day", "2_week", "3_month", "4_quarter", "5_episode", "6_year"}
	if !slices.Equal(urgency.Values, want) {
		t.Errorf("urgency values = %v, want %v", urgency.Values, want)
	}
}
```

**Step 2: Run it; expect FAIL.** `just test-go-pkg ./internal/alfa/type_blobs/`

**Step 3: Implement.** Add to `actionableFields()` (after `status`, before
`priority` — keep enum fields grouped):

```go
{
	Name:    "urgency",
	Kind:    "enum",
	Values:  []string{"0_hour", "1_day", "2_week", "3_month", "4_quarter", "5_episode", "6_year"},
	// no Default — urgency is unset until triaged. (Tuning lever: a default
	// could be "6_year"; left unset to match the personal repo, where an untagged item
	// has no urgency.)
},
```

Extend `actionableFieldsReader()` JSON projection and
`actionableFieldsWriter()` to include `urgency`:

```go
// reader
`yq -p toml -o json '{"status": .status, "urgency": .urgency, "priority": .priority, "due": .due}'`
// writer (append the urgency clause)
`... | .urgency = \"$DODDER_FIELD_urgency\" | ...`
```

**Step 4: Run tests; expect PASS.** `just test-go-pkg ./internal/alfa/type_blobs/`

**Step 5: Commit.**
`feat(type_blobs): add urgency field to actionable types`

> **Decision to confirm during execution:** exact `urgency` value spelling
> (`0_hour` vs `hour` vs `urgency-0_hour`). `0_hour` chosen for lexical
> sortability + brevity; revisit if it reads poorly in `dodder show`.

### Task A2: Split task vs recurring field sets; add `recurrence`

**Promotion criteria:** N/A.

**Files:**
- Modify: `go/internal/alfa/type_blobs/main.go`
- Test: `go/internal/alfa/type_blobs/main_test.go`

**Step 1: Write the failing test.** Assert `DefaultChoreType()` has a
`recurrence` string field and `DefaultTaskType()` does **not**:

```go
func TestRecurrenceOnlyOnRecurring(t1 *testing.T) {
	t := ui.MakeT(t1)
	hasField := func(b TomlV2, name string) bool {
		for _, f := range b.Fields { if f.Name == name { return true } }
		return false
	}
	if hasField(DefaultTaskType(), "recurrence") {
		t.Error("!task must not have recurrence")
	}
	if !hasField(DefaultChoreType(), "recurrence") {
		t.Error("!chore must have recurrence")
	}
}
```

**Step 2: Run; expect FAIL.**

**Step 3: Implement.** Introduce a second constructor for the recurring field
set, keeping `actionableFields()` for `!task`:

```go
// recurringFields is actionableFields plus a recurrence cadence, shared by
// !chore and !habit.
func recurringFields() []FieldDefinition {
	return append(actionableFields(), FieldDefinition{
		Name: "recurrence",
		Kind: "string", // ISO-8601 duration, e.g. "P1W"; no date kind exists
	})
}

func recurringFieldsReader() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o json '{"status": .status, "urgency": .urgency, "priority": .priority, "due": .due, "recurrence": .recurrence}'`,
	}
}

func recurringFieldsWriter() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o toml -i ".status = \"$DODDER_FIELD_status\" | .urgency = \"$DODDER_FIELD_urgency\" | .priority = \"$DODDER_FIELD_priority\" | .due = \"$DODDER_FIELD_due\" | .recurrence = \"$DODDER_FIELD_recurrence\"" "$DODDER_BLOB_PATH"`,
	}
}
```

Point `DefaultChoreType()` at `recurringFields()` + the recurring reader/writer.

**Step 4: Run; expect PASS.**

**Step 5: Commit.** `feat(type_blobs): add recurrence field to recurring actionable types`

### Task A3: Add `DefaultHabitType()`

**Promotion criteria:** N/A.

**Files:**
- Modify: `go/internal/alfa/type_blobs/main.go`
- Modify: `go/internal/golf/type_blobs/main.go` (re-export, alongside
  `DefaultTaskType`/`DefaultChoreType` at lines 27-28)
- Test: `go/internal/alfa/type_blobs/main_test.go`

**Step 1: Write the failing test** asserting `DefaultHabitType()` exists,
is `toml`/`toml`, and has `recurringFields()` (status/urgency/priority/due/recurrence).

**Step 2: Run; expect FAIL** (undefined).

**Step 3: Implement.** Mirror `DefaultChoreType()`; `!habit` is structurally
identical (differs only by default cadence + description, which live in the hook
/ docs, not the field schema):

```go
// DefaultHabitType returns the built-in !habit type blob. Structurally
// identical to !chore (same recurring field set); the distinction is
// semantic (a consistency practice vs a periodic obligation) and surfaces as
// a tighter default cadence in the hooks. See the design doc.
func DefaultHabitType() TomlV2 {
	return TomlV2{
		FileExtension: "toml",
		VimSyntaxType: "toml",
		Fields:        recurringFields(),
		FieldsReader:  recurringFieldsReader(),
		FieldsWriter:  recurringFieldsWriter(),
	}
}
```

Re-export in `golf/type_blobs/main.go`: `DefaultHabitType = golf_tb.DefaultHabitType`.

**Step 4: Run; expect PASS.**

**Step 5: Commit.** `feat(type_blobs): add DefaultHabitType built-in`

### Task A4: Genesis commits `!habit`

**Promotion criteria:** N/A.

**Files:**
- Modify: `go/internal/romeo/local_working_copy/genesis.go:259-271` (the
  builtin slice in `prepareBuiltinActionableTypes`)
- Test: a BATS test (Task A7)

**Step 1:** Add a third entry to the builtin slice:

```go
{
	objectIdString: "habit",
	blob:           type_blobs.DefaultHabitType(),
},
```

**Step 2: Build.** `just build` (nix). Expect success.

**Step 3: Commit.** `feat(genesis): commit !habit built-in actionable type`

### Task A5: Blob-backed pandoc formatter on the actionable `body`

**Promotion criteria:** N/A.

**Files:**
- Modify: `go/internal/alfa/type_blobs/main.go` (a formatter constructor for
  actionable types)
- Modify: `go/internal/romeo/local_working_copy/genesis.go` —
  `prepareBuiltinActionableTypes` signature + the genesis caller at line 97
  (thread `toolBlobDigests`); attach the three tool blob references per type
- Test: `zz-tests_bats/current_version/` (Task A7) + unit test for the formatter
  string

**Context:** `!md`'s formatter renders the *whole* blob (it IS markdown). The
actionable blob is TOML with markdown in the `body` key, so the formatter must
extract `body`, strip the `dang` first line (the Phase-1 convention), then
pandoc. Reuse the existing `dodder-edit` defaults + filters materialized to
`$DODDER_BLOB_TREE`.

**Step 1: Write the failing unit test** asserting `DefaultTaskType()` (when
built for the pandoc-tools path) carries a `text` formatter whose script
extracts `.body` and invokes pandoc with `--data-dir="$DODDER_BLOB_TREE"`.

> **Note:** decide during execution whether the formatter is unconditional on
> the type blob (simplest) or gated like `!md` is by `IncludeDefaultPandocTools`.
> Recommended: put the formatter on the type blob unconditionally (it only runs
> when the user formats), and gate only the *tool blob references* on the
> pandoc-tools flag — matching how `!md` separates `DefaultWithPandocFormatter`
> (blob) from `addToolBlobReference` (genesis).

**Step 2: Run; expect FAIL.**

**Step 3: Implement the formatter.** Add an `actionableFormatters()` helper and
wire it into all three Default*Type constructors:

```go
func actionableFormatters() map[string]script_config.WithOutputFormat {
	return map[string]script_config.WithOutputFormat{
		"text": {
			ScriptConfig: script_config.ScriptConfig{
				Description: "Render the dang-typed body with pandoc",
				// Extract the body, drop the leading `#!dang ...` line
				// (Phase-1 convention), normalize with the blob-backed defaults.
				Script: `yq -p toml -r '.body' | sed '1{/^#!dang/d}' | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-edit`,
			},
			FileExtension: "md",
		},
	}
}
```

Set `Formatters: actionableFormatters()` on Default{Task,Chore,Habit}Type.

**Step 4: Thread tool blob references in genesis.** Change
`prepareBuiltinActionableTypes` to accept `toolBlobs toolBlobDigests`, pass it
from the caller (genesis.go:97 — the same `toolBlobs` already created for
`prepareDefaultType`), and inside the per-type loop, when
`bigBang.IncludeDefaultPandocTools`, call `addToolBlobReference` for the three
tools (identical to `prepareDefaultType:215-236`). Extract a shared helper
`attachPandocToolRefs(object, toolBlobs)` to keep `prepareDefaultType` and
`prepareBuiltinActionableTypes` DRY.

**Step 5: Build + unit test.** `just build && just test-go-pkg ./internal/alfa/type_blobs/`

**Step 6: Commit.** `feat(type_blobs,genesis): blob-backed pandoc formatter on actionable body`

### Task A6: Update the existing actionable unit tests

**Files:**
- Modify: `go/internal/alfa/type_blobs/main_test.go:12-60`

Update `TestDefaultTaskType` / `TestDefaultChoreType` for the new field set
(urgency added; chore/habit gain recurrence) and the formatter. Use the
two-pass strategy: assert WRONG, capture actual, write the exact assertion.

**Commit.** `test(type_blobs): update actionable type assertions for urgency/recurrence/formatter`

### Task A7: BATS — genesis + field projection + formatter

**Promotion criteria:** N/A.

**Files:**
- Locate + modify the existing `genesis_opt_in_actionable_types` test (grep
  `genesis_opt_in_actionable_types` under `zz-tests_bats/current_version/`) to
  also assert `!habit:t` is present and that `!task:t` shows the `urgency` field.
- Add: `zz-tests_bats/current_version/actionable_types.bats`

**Tests to add (each `run_dodder_init -include-builtin-actionable-types` first):**

1. `actionable_genesis_creates_task_chore_habit` — `dodder show :t` includes
   `!task`, `!chore`, `!habit`; without the flag, none appear (exact assertion).
2. `actionable_task_projects_urgency_field` — `dodder new '!task'` with a TOML
   blob setting `urgency = "2_week"`, then `dodder show` displays the projected
   `urgency` field. (Reader script projection.)
3. `actionable_chore_has_recurrence` — a `!chore` blob with `recurrence = "P1W"`
   projects the field; a `!task` blob with the same key does **not** (no
   recurrence field on `!task`).
4. `actionable_body_renders_via_pandoc` — `skip_if_no_pandoc`; a `!task` blob
   whose `body` is `#!dang md\n# Hello` formats (via `dodder show -format text`)
   to pandoc-normalized markdown with the `#!dang` line stripped.

Use `just test-bats-targets actionable_types.bats` for the dev loop.

**Step: Regenerate fixtures.** Only the opt-in genesis path changes output;
`just test-bats-update-fixtures`, review
`git diff -- zz-tests_bats/previous_versions/`, commit fixtures + tests
together. The frozen v14 snapshot stays untouched.

**Commit.** `test(bats): actionable types genesis, field projection, body formatting`

---

# Part B — Per-state + recurrence hooks (investigation-gated)

> **This part has a real unknown.** The design has the hooks read the `status`
> field and drive behavior (cancelled → archive; `!task` done → archive;
> `!chore`/`!habit` done → advance `due`, reset `todo`). dodder's lua hook API
> (`WithStringLuaHooks`, `on_format`/`on_pre_commit`, gopher-lua) is **separate**
> from pandoc's lua. Whether a hook can (a) read `Metadata.Index.Fields`,
> (b) mutate the blob (advance `due`), and (c) archive (set dormant / add a tag)
> is unverified. The the personal repo hooks only manipulated tags (`kinder.Etiketten`).
> **Do the spike before writing hook code.**

### Task B1: Spike — what can a lua hook see and mutate?

**Files (read-only investigation):**
- The lua hook host (grep `on_pre_commit` / `on_format` / `Etiketten` /
  `print_tags` across `go/internal/**/lua*` and the hook-invocation site).
- The hook context object passed to `on_pre_commit(kinder, mutter)` — what fields
  it exposes (tags? `Index.Fields`? blob reader/writer?).

**Deliverable:** a short note appended to this plan answering:
1. Can a hook read the projected `status`/`urgency`/`due`/`recurrence` fields?
2. Can a hook mutate fields or the blob (to advance `due` / reset `status`)?
3. Can a hook archive an object (set dormant or add a recognized archive tag)?
4. Does the hook lua env get `$DODDER_BLOB_TREE` / a `require` path (→ can the
   `actionable-common.lua` be blob-backed, or must it be inlined in the `Hooks`
   string)?

**Decision gate:** if hooks **cannot** read fields / mutate, STOP and bring the
finding back to the design — the behavior may need a different mechanism (an
`organize`-time writer-script transition, or a dedicated command) rather than a
commit/format hook. Do not force a speculative implementation.

### Task B2: Author the field-model `actionable-common.lua`

*(Proceed only if B1 confirms hooks can read fields + archive/mutate.)*

**Files:**
- Create: `go/internal/romeo/local_working_copy/embedded/actionable/actionable-common.lua`
- Create/modify: a `go:embed` file (mirror `embedded_pandoc_tools.go`)
- Modify: `go/internal/alfa/type_blobs/main.go` — set `Hooks:` on the three
  Default*Type constructors (inline the embedded lua, or `require` the
  materialized blob per B1's answer)

**Behavior (field model, replacing the personal repo's tag-collapse machinery):**
- `on_pre_commit` / `on_format`:
  - `status == "cancelled"` → archive (all three).
  - `status == "done"`:
    - `!task` → archive.
    - `!chore` / `!habit` → advance `due` by `recurrence`, set `status = "todo"`.
  - else → active; apply field normalization (default `urgency`, guard `due` on
    recurring objects).

**TDD:** a Go unit test exercising the lua module against a synthetic hook
context (status=done on a recurring object advances `due`, resets `status`;
status=cancelled archives; `!task` done archives). If the host makes unit-testing
the lua hard, drive it via a BATS test instead (commit a done `!chore`, assert
`due` advanced + `status=todo`; commit a done `!task`, assert it became dormant).

**Self-containment:** the hook lua must NOT reference an external object (no
`require("[cotton/horsea !lua] ...")`) — it ships embedded/blob-backed so the
types are portable to a fresh clone / dodder.net.

**Commit.** `feat(type_blobs): self-contained actionable-common hooks (status-driven + recurrence)`

### Task B3: BATS — hook behavior

**Files:**
- Add to `zz-tests_bats/current_version/actionable_types.bats`:
  - `actionable_task_done_archives` — set a `!task` to `status=done`; it leaves
    the default (non-dormant) listing.
  - `actionable_chore_done_recurs` — set a `!chore` (`recurrence="P1W"`,
    `due="2026-07-01"`) to `status=done`; after commit `due` advanced one week
    and `status` reset to `todo`.
  - `actionable_cancelled_archives` — `status=cancelled` archives all three.

Regenerate fixtures if genesis output changed; commit fixtures + tests together.

**Commit.** `test(bats): actionable hook per-state + recurrence behavior`

---

## Verification (whole plan)

1. `just build` (nix `.#dodder` + debug) — compiles with the new fields/formatter/hooks.
2. `just test` — full unit + bats (the pre-merge gate runs this anyway; do not
   pre-run it redundantly before merge).
3. Manual smoke: `dodder init -include-builtin-actionable-types` in a scratch
   dir, `dodder show :t` shows `!task`/`!chore`/`!habit`; `dodder new '!task'`
   with a TOML body projects `urgency`; `dodder show -format text` renders the
   `body` via pandoc.
4. Confirm the no-flag path is byte-identical to before (additive guarantee):
   the existing fresh-store fixtures must NOT change.

## Open items / risks

- **Hook API capability (Task B1)** — the gating unknown; may bounce Part B back
  to design.
- **Formatter `body` extraction** — the `yq '.body' | sed | pandoc` pipe is the
  Phase-1 `dang` convention. The real `dang` mechanism (metadata-bound,
  hyphence-pinned identifier; FDR task #19) supersedes the `sed` strip later.
- **Richer formatters** (`html`/`gdoc`/`pdf-beamer`) deferred — each needs its
  own pandoc defaults blob; additive.
- **Haustoria** (`mike/haustoria_caldav`) keeps emitting `status`/`priority`/`due`
  only; the readers tolerate absent `urgency`/`recurrence`. Mapping VTODO →
  those fields is a follow-up, non-blocking.
- **`urgency` default + value spelling** and **recurrence representation** are
  tuning levers (see design doc).

## Execution handoff

Plan complete. Part A is concrete and shippable; Part B starts with a spike that
may return to design. **REQUIRED SUB-SKILL for execution:**
eng:subagent-driven-development (fresh subagent per task + code review), staying
in this session.
