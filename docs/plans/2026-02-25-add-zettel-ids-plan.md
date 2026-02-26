# Add Zettel IDs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace mutable flat Yin/Yang provider files with content-addressed delta blobs tracked by a signed append-only object ID log.

**Architecture:** New object ID log at `DirData("object_id_log")` uses binary stream index encoding with repo pub key signatures. Each log entry references a delta blob of new words. Provider loading replays the log or falls back to flat files for pre-migration repos. Three new commands: `add-zettel-ids-yin`, `add-zettel-ids-yang`, `migrate-zettel-ids`.

**Tech Stack:** Go, binary stream index encoding (key_bytes fields), content-addressed blob store, horizontal versioning via `triple_hyphen_io` coders.

**Design doc:** `docs/plans/2026-02-25-add-zettel-ids-design.md`

**Skills to reference:**
- `features-zettel_ids` — ZettelId system overview
- `design_patterns-horizontal_versioning` — versioned type registration pattern
- `design_patterns-hamster_style` — Go code conventions for dodder
- `robin:bats-testing` — BATS integration test patterns

---

### Task 1: Clean Up Orphan TypeZettelIdListV0

**Files:**
- Modify: `go/src/echo/ids/types_builtin.go` — remove `TypeZettelIdListV0` constant and its `registerBuiltinTypeString` call

**Step 1: Remove the constant**

In `go/src/echo/ids/types_builtin.go`, delete the `TypeZettelIdListV0` constant (line 52, `"!zettel_id_list-v0"`) and its registration in `init()` (line 131).

**Step 2: Verify no references exist**

Use `lux` references tool on the `TypeZettelIdListV0` symbol to confirm nothing else uses it. If references are found, evaluate whether they should point to the new type string instead.

**Step 3: Run tests**

Run: `just test-go`
Expected: PASS — this type was marked "not used yet"

**Step 4: Commit**

```
git add go/src/echo/ids/types_builtin.go
git commit -m "chore: remove orphan TypeZettelIdListV0 type string"
```

---

### Task 2: Register TypeObjectIdLogV1

**Files:**
- Modify: `go/src/echo/ids/types_builtin.go` — add new type constants and registration

**Step 1: Add type constants**

Add to the const block in `go/src/echo/ids/types_builtin.go`:

```go
TypeObjectIdLogV1       = "!object_id_log-v1"
TypeObjectIdLogVCurrent = TypeObjectIdLogV1
```

**Step 2: Register in init()**

Add to the `init()` function:

```go
registerBuiltinTypeString(TypeObjectIdLogV1, genres.Unknown, false)
```

Follow the pattern used by `TypeTomlBlobStoreConfigV1` and similar entries.

**Step 3: Run tests**

Run: `just test-go`
Expected: PASS

**Step 4: Commit**

```
git add go/src/echo/ids/types_builtin.go
git commit -m "feat: register TypeObjectIdLogV1 type string"
```

---

### Task 3: Add DirObjectIdLog to Directory Layout

**Files:**
- Modify: `go/src/echo/directory_layout/v3.go` — add `DirObjectIdLog()` method
- Modify: `go/src/echo/directory_layout/main.go` — add to `Repo` interface

**Step 1: Add method to v3**

In `go/src/echo/directory_layout/v3.go`, add alongside `FileInventoryListLog()`:

```go
func (layout v3) DirObjectIdLog() string {
	return layout.MakeDirData("object_id_log").String()
}
```

**Step 2: Add to interface**

In `go/src/echo/directory_layout/main.go`, add `DirObjectIdLog() string` to the `Repo` interface (or the appropriate sub-interface where `FileInventoryListLog()` is declared).

**Step 3: Add to DirsGenesis()**

In `go/src/echo/directory_layout/v3.go`, add `layout.DirObjectIdLog()` to the `DirsGenesis()` return slice.

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
git add go/src/echo/directory_layout/v3.go go/src/echo/directory_layout/main.go
git commit -m "feat: add DirObjectIdLog to directory layout"
```

---

### Task 4: Define Object ID Log Entry Interface and V1 Struct

Determine the correct package for this type by examining the NATO hierarchy. The entry references `markl.Id` (echo-level) and needs to be consumed by the provider (`foxtrot`) and the store (`tango`). It should live at the `foxtrot` or `golf` level — check which is appropriate given existing imports.

**Files:**
- Create: `go/src/<layer>/object_id_log/main.go` — interface + Side enum
- Create: `go/src/<layer>/object_id_log/v1.go` — V1 struct
- Create: `go/src/<layer>/object_id_log/coding.go` — Architecture A coder

**Step 1: Define the Side enum**

```go
type Side uint8

const (
	SideYin  Side = iota
	SideYang
)
```

**Step 2: Define the interface**

```go
type Entry interface {
	GetSide() Side
	GetTai() tai.TAI
	GetMarklId() markl.Id
	GetWordCount() int
}
```

**Step 3: Define V1 struct**

```go
type V1 struct {
	Side      Side     `toml:"side"`
	Tai       tai.TAI  `toml:"tai"`
	MarklId   markl.Id `toml:"markl-id"`
	WordCount int      `toml:"word-count"`
}
```

Implement the `Entry` interface on `V1`.

**Step 4: Register the coder**

Follow `golf/blob_store_configs/coding.go` as the template. Create a `Coder` variable using `triple_hyphen_io.CoderToTypedBlob` with a `CoderTypeMapWithoutType` map containing one entry for `ids.TypeObjectIdLogV1`.

**Step 5: Run tests**

Run: `just test-go`
Expected: PASS (compiles, no consumers yet)

**Step 6: Commit**

```
git add go/src/<layer>/object_id_log/
git commit -m "feat: add object ID log entry interface, V1 struct, and coder"
```

---

### Task 5: Implement Object ID Log Reader/Writer

This handles reading/writing the append-only log file using the binary stream index encoding with signatures.

**Files:**
- Create: `go/src/<layer>/object_id_log/log.go` — log reader/writer

**Step 1: Examine the inventory list log writer**

Study how `writeInventoryListLog()` in `go/src/juliett/env_repo/genesis.go` creates and writes log entries. Study `go/src/lima/stream_index/binary_encoder.go` for the binary field encoding pattern with `RepoPubKey` and `RepoSig` fields.

**Step 2: Implement the log writer**

Create a function that:
1. Opens the log file at `DirObjectIdLog()` in append mode
2. Encodes a V1 entry using the box-format binary encoding with repo pub key and signature
3. Writes and closes

**Step 3: Implement the log reader**

Create a function that:
1. Opens the log file at `DirObjectIdLog()`
2. Decodes all entries sequentially
3. Returns `[]Entry` (or iterates via callback)

**Step 4: Run tests**

Run: `just test-go`
Expected: PASS

**Step 5: Commit**

```
git add go/src/<layer>/object_id_log/log.go
git commit -m "feat: implement object ID log reader and writer"
```

---

### Task 6: Update Provider Loading to Support Log Replay

**Files:**
- Modify: `go/src/foxtrot/object_id_provider/factory.go` — add log-based loading path
- Modify: `go/src/foxtrot/object_id_provider/main.go` — support appending words

**Step 1: Add log replay to provider factory**

Modify `New()` in `factory.go` to:
1. Check if the object ID log exists at `DirObjectIdLog()`
2. If it exists: replay the log, concatenating all Yin entries into the yin provider and all Yang entries into the yang provider (fetching each delta blob and appending its words)
3. If it does not exist: fall back to reading flat Yin/Yang files (current behavior)

**Step 2: Add append support to provider**

The `provider` type (`[]string`) needs a method to append words from a delta blob. Add a function that reads a blob by MarklId, parses newline-delimited words, and appends them to the provider slice.

**Step 3: Run tests**

Run: `just test-go`
Expected: PASS (no logs exist yet, so all repos use fallback path)

**Step 4: Commit**

```
git add go/src/foxtrot/object_id_provider/factory.go go/src/foxtrot/object_id_provider/main.go
git commit -m "feat: support log-based provider loading with flat file fallback"
```

---

### Task 7: Implement migrate-zettel-ids Command

**Files:**
- Create: `go/src/yankee/commands_dodder/migrate_zettel_ids.go`

**Step 1: Write the BATS test**

Create a test that:
1. Inits a repo with flat Yin/Yang files (old style)
2. Runs `dodder migrate-zettel-ids`
3. Verifies the object ID log was created
4. Verifies `dodder new` still allocates IDs correctly after migration
5. Verifies running `migrate-zettel-ids` a second time is a no-op or errors gracefully

**Step 2: Run test to verify it fails**

Run: `just test-bats-targets migrate_zettel_ids.bats`
Expected: FAIL — command does not exist

**Step 3: Implement the command**

Register via `utility.AddCmd("migrate-zettel-ids", ...)` in an `init()` function. The command:
1. Opens the repo (requires lock)
2. Checks if the log already exists — if so, print message and exit
3. Reads existing flat Yin file into a string
4. Writes it as a blob to the repo's blob store, gets MarklId
5. Appends a signed V1 log entry (Side=Yin, MarklId, word count)
6. Repeats for Yang
7. Rebuilds flat file caches from the log (validates round-trip)
8. Resets and rebuilds the zettel ID availability index

**Step 4: Run tests**

Run: `just test-bats-targets migrate_zettel_ids.bats`
Expected: PASS

**Step 5: Run full test suite**

Run: `just test`
Expected: PASS

**Step 6: Commit**

```
git add go/src/yankee/commands_dodder/migrate_zettel_ids.go zz-tests_bats/migrate_zettel_ids.bats
git commit -m "feat: add migrate-zettel-ids command"
```

---

### Task 8: Implement add-zettel-ids-yin and add-zettel-ids-yang Commands

**Files:**
- Create: `go/src/yankee/commands_dodder/add_zettel_ids_yin.go`
- Create: `go/src/yankee/commands_dodder/add_zettel_ids_yang.go`

These two commands share nearly all logic, differing only in the target side. Extract shared logic into a helper or use a shared struct with a side parameter.

**Step 1: Write the BATS tests**

Create tests that:
1. Init a repo (using `migrate-zettel-ids` or new genesis)
2. Pipe raw text to `dodder add-zettel-ids-yin`
3. Verify new words were added (check output for count)
4. Verify `dodder peek-zettel-ids` shows a larger pool
5. Pipe text with overlapping words — verify dedup (count should be lower)
6. Pipe text whose words overlap with Yang — verify cross-side rejection
7. Pipe text with no new words — verify no-op message
8. Repeat key tests for `add-zettel-ids-yang`

**Step 2: Run tests to verify they fail**

Run: `just test-bats-targets add_zettel_ids.bats`
Expected: FAIL — commands do not exist

**Step 3: Implement the commands**

Each command:
1. Reads stdin
2. Runs `unicorn.ExtractUniqueComponents` on the input lines
3. Loads both Yin and Yang providers (from cache)
4. Builds a set of all existing words across both sides
5. Filters candidates: reject any word in the existing set
6. If no new words remain, prints a message and exits
7. Writes the filtered word list as a blob to the repo's blob store
8. **Acquires repo lock**
9. Appends a signed V1 log entry
10. Rebuilds the flat file cache for the target side
11. Resets and rebuilds the zettel ID availability index
12. Prints count of new words added and new total pool size

Register via `utility.AddCmd("add-zettel-ids-yin", ...)` and `utility.AddCmd("add-zettel-ids-yang", ...)`.

**Step 4: Run tests**

Run: `just test-bats-targets add_zettel_ids.bats`
Expected: PASS

**Step 5: Run full test suite**

Run: `just test`
Expected: PASS

**Step 6: Commit**

```
git add go/src/yankee/commands_dodder/add_zettel_ids_yin.go go/src/yankee/commands_dodder/add_zettel_ids_yang.go zz-tests_bats/add_zettel_ids.bats
git commit -m "feat: add add-zettel-ids-yin and add-zettel-ids-yang commands"
```

---

### Task 9: Update Genesis to Use Object ID Log

**Files:**
- Modify: `go/src/juliett/env_repo/genesis.go` — replace `CopyFileLines` with blob writes + log entries
- Modify: `go/src/xray/command_components_dodder/genesis.go` — update flag descriptions

**Step 1: Write the BATS test**

Add a test (or modify existing init tests) that:
1. Inits a repo with `-yin <(raw text)` and `-yang <(raw text)`
2. Verifies the object ID log was created
3. Verifies `dodder new` allocates IDs from the processed words
4. Verifies cross-side uniqueness was enforced during init

**Step 2: Run test to verify it fails**

Expected: FAIL — genesis still uses `CopyFileLines`

**Step 3: Update genesis**

In `go/src/juliett/env_repo/genesis.go`, replace the `ohio_files.CopyFileLines` calls (lines 63-77) with:
1. Read each input file
2. Run `unicorn.ExtractUniqueComponents` on the lines
3. Enforce cross-side uniqueness (reject words appearing in both)
4. Write each word list as a blob
5. Append two signed V1 log entries
6. Write flat file caches for immediate provider use

Update `-yin` and `-yang` flag descriptions in `go/src/xray/command_components_dodder/genesis.go` to indicate they accept raw text, not pre-processed word lists.

**Step 4: Run tests**

Run: `just test`
Expected: PASS

**Step 5: Update fixtures**

Run: `just test-bats-update-fixtures`
Review the diff — genesis output format changed.

**Step 6: Commit**

```
git add go/src/juliett/env_repo/genesis.go go/src/xray/command_components_dodder/genesis.go zz-tests_bats/
git commit -m "feat: genesis writes object ID log instead of flat files"
```

---

### Task 10: Update Completion Tests

**Files:**
- Modify: `zz-tests_bats/complete.bats` — add new subcommands

**Step 1: Run completion test**

Run: `just test-bats-targets complete.bats`
Expected: May already FAIL if the new commands are registered but not in the expected output

**Step 2: Update expected output**

Add `add-zettel-ids-yin`, `add-zettel-ids-yang`, and `migrate-zettel-ids` to the `complete_subcmd` test's expected output list.

**Step 3: Run test**

Run: `just test-bats-targets complete.bats`
Expected: PASS

**Step 4: Commit**

```
git add zz-tests_bats/complete.bats
git commit -m "test: add new zettel ID commands to completion test"
```

---

### Task 11: Full Test Suite and Fixture Update

**Step 1: Run full test suite**

Run: `just test`
Expected: PASS

**Step 2: Update fixtures if needed**

Run: `just test-bats-update-fixtures`
Review the diff — commit if changed.

**Step 3: Final commit**

```
git add -A
git commit -m "chore: update fixtures for object ID log changes"
```
