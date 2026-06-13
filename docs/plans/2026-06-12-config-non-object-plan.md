# Config as a Non-Object — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use eng:subagent-driven-development to
> implement this plan task-by-task.

**Goal:** Remove config's object status (genre, signed commits, index
participation), add `show-config`, and persist config history as a repo-local
inventory-list-format log with digest-chained entries.

**Architecture:** Config states stay bare-TOML `!toml-config-v2` blobs. A new
append-only log file (layout name `config_log`, sibling of
`inventory_lists_log`) holds one box-format entry per state, written through
the same MultiWriter append mechanism as `WriteInventoryListObject`. Entries
chain mother-slot digests (previous entry's digSelf) instead of signatures.
Bootstrap reads the log's last entry; append order is the history. Store
version bumps V15 → V16; reindex converts old konfig object history into log
entries.

**Tech Stack:** Go (NATO-tiered internal packages), hyphence/box coders,
markl ids, BATS via just recipes.

**Rollback:** Branch-only until the VCurrent bump task (Task 12). Migration is
additive (old inventory lists untouched); pre-migration stores work with the
old binary. After Task 12, rollback = revert the branch before merge.

**Design doc:** `docs/plans/2026-06-12-config-non-object-design.md`
**FDR:** `docs/features/0020-config-as-non-object.md`

**Conventions that bind every task:**

- Build/test ONLY through just recipes from repo root (or `go/` where noted):
  `just test-go-pkg ./internal/...`, `just test-bats-targets <file>.bats`,
  `just build-go-generate` via `just commit-codegen`.
- Never `errors.Is` on possible EOF — use `errors.IsEOF()`.
- `sku.Transacted` pool rules: never dereference; `GetWithRepool`;
  `TransactedResetter.ResetWith`.
- Tests use `ui.MakeT(t1)`; helpers take `*ui.T`.
- Commit after every green step. Codegen files (`*_tommy.go` etc.) commit
  alongside their sources (`just commit-codegen` from `go/`).

---

> **DESIGN PIVOT (2026-06-13):** Config log entries are **signed**, reusing
> the existing `!inventory_list-v2` type and coder verbatim — the config log
> is literally an inventory list in its own file (`FileConfigLog`). This
> replaces the earlier "unsigned, digest-chained mother" approach, which the
> box format can't carry (it gates mother emission behind a non-null object
> signature, `box_format/transacted.go:264`). Consequences: **Task 1** keeps
> only the layout path (the type string and mother-digest purpose were
> reverted); **Task 2 is eliminated** (no new coder needed); **Task 3+**
> sign entries with the repo key and use the v2 coder. See the design doc /
> FDR 0020.

## Phase A — config log foundation

### Task 1: Layout path (DONE — pivoted)

**Files:**
- Modify: `go/internal/bravo/directory_layout/v3.go` — `FileConfigLog()`
  returning `layout.MakeDirData("config_log").String()`.
- Modify: `go/internal/bravo/directory_layout/main.go` — interface method.

The type string `!inventory_list-config-v0` and the
`PurposeObjectMotherDigestV1` markl purpose that an earlier draft added here
were **reverted**: the config log reuses `!inventory_list-v2` and its signed
coder, so neither is needed. Only the layout path remains.

### Task 2: (ELIMINATED by the design pivot)

No new coder. Config entries are bona fide signed `!inventory_list-v2`
entries; the existing coder and box format carry them unchanged. The config
log is distinguished from the object inventory-list log purely by file
(`FileConfigLog` vs `FileInventoryListLog`).

### Task 3: config_log package (append / head / all)

**Files:**
- Create: `go/internal/india/config_log/main.go` (india tier: may import
  foxtrot/env_repo, golf/object_finalizer, hotel/inventory_list_coders)
- Test: `go/internal/india/config_log/main_test.go`

**API:**

    type Log struct { /* envRepo, pathLog, closet, blobStore, finalizer */ }

    func Make(envRepo env_repo.Env, closet inventory_list_coders.Closet) Log

    // Append finalizes the entry (digSelf), sets mother = previous head's
    // digSelf (absent for root), writes the encoded entry through a
    // MultiWriter to the default blob store and the log file (O_APPEND +
    // Sync), mirroring inventory_list_store/blob_store_v1.go:74-131.
    func (log Log) Append(entry *sku.Transacted) error

    // Head returns the last entry, or ErrEmpty when the log doesn't exist.
    func (log Log) Head() (*sku.Transacted, error)

    // All yields entries oldest→newest (streaming decode like
    // AllInventoryLists, blob_store_v1.go:133-165).
    func (log Log) All() sku.Seq

Entries are SIGNED `!inventory_list-v2` entries. `Append` mirrors
`inventory_list_store/blob_store_v1.go:74-131`: set the entry's mother to the
current head (via `Transacted.SetMother`, which is correct now that entries
are signed), `FinalizeAndSign` with `envRepo.GetConfigPrivate().Blob`, then
MultiWriter the encoded entry to the default blob store and the
`FileConfigLog()` file. Reuse the closet coder for `ids.TypeInventoryListV2`.
Consider whether `config_log` can simply wrap/parameterize the existing
`inventory_list_store.blobStoreV1` with a different `pathLog` rather than
reimplementing append/read — note this in the consolidation follow-up.

**Steps (TDD, one commit per green):**

1. Failing test: `Head()` on missing file → typed sentinel `ErrEmpty`
   (use `errors.MakeTypedSentinel` per design_patterns-typed_error_sentinels).
2. Failing test: `Append` root entry (no mother) then `Head()` returns it,
   mother slot null, object sig non-null (signed); file exists at
   `envRepo.FileConfigLog()`.
3. Failing test: second `Append` chains — `Head().GetMotherObjectSig()`
   equals the first entry's object signature; `All()` yields 2 entries in
   order.
4. Failing test: the entry decodes back and verifies (the v2 coder's
   `afterDecoding` runs `FinalizeAndVerify`).
5. Implement minimally after each failing test;
   run `just go/test-go-pkg ./internal/india/config_log/` between steps.

Test env setup: follow existing india-tier tests that construct a temp
`env_repo` (rg for `env_repo` usage in `_test.go` under internal/india for
the canonical harness).

Commit message family: `feat(config-log): <append|head|all> implementation`

---

## Phase B — CLI

### Task 4: show-config command

**Files:**
- Create: `go/internal/uniform/commands_dodder/show_config.go`
- Modify: `zz-tests_bats/current_version/complete.bats` (~line 117 region,
  add `^show-config[[:space:]]+...$` line)
- Test: `zz-tests_bats/current_version/show_config.bats` (created here, full
  assertions wired in Task 11 after migration exists; this task covers
  fresh-store behavior)

**Step 1:** Command skeleton (mirror `edit_config.go:12-14` registration):

    func init() {
        utility.AddCmd("show-config", &ShowConfig{})
    }

    type ShowConfig struct {
        command_components_dodder.LocalWorkingCopy
        History bool // -history
    }

Behavior:
- no args: read `config_log.Head()`, stream the blob (by entry blob digest)
  to stdout.
- one arg (markl id): `blob_store` read of that digest, stream to stdout
  (print raw; no decode gate beyond markl parse).
- `-history`: iterate `All()`, print each entry box line via the repo's
  existing box printer (`local_working_copy` printer used by show; do NOT
  hand-build output — reuse `PrinterTransacted()` or the box format flag
  default).

**Step 2:** BATS (two-pass assertion strategy; tag
`# bats file_tags=user_story:config`):

    function show_config_fresh { # @test
        run_dodder show-config
        assert_success
        assert_output "WRONG"   # capture pass; replace with exact TOML
    }

Run: `just test-bats-targets show_config.bats` — capture, tighten, re-run
green. Fresh-store init must already produce a root entry — until Task 6
lands, mark the test `skip` with a TODO referencing Task 6, and assert only
`complete.bats` here.

**Step 3:** Commit: `feat(show-config): command skeleton + completion entry`

### Task 5: edit-config writes the log

**Files:**
- Modify: `go/internal/uniform/commands_dodder/edit_config.go:31-68`
- Modify: `go/internal/uniform/commands_dodder/dormant_edit.go:31-68`
- Modify: `go/internal/uniform/commands_dodder/konfig_edit.go` (read side:
  current blob now comes from config store / log head, not
  `ReadTransactedFromObjectId(ids.Config)`)
- Modify: `go/internal/oscar/store/main.go:172-179` (`UpdateKonfig` retarget)
- Modify: `go/internal/oscar/store/create.go:37-91` (drop config path if
  unused by other callers — check `createOrUpdateBlobDigest` callers first)

**Step 1:** Failing BATS: `just test-bats-targets edit_config.bats` after
changing the implementation expectation — i.e. update
`edit_config.bats` assertions FIRST to the new world:
- `edit_config_and_change` asserts `show-config` output and a 2-entry
  `show-config -history`.
- Remove/replace `show :konfig` assertions (that query dies in Task 8;
  use `show-config` now so this file stays green through Phase C).

**Step 2:** Implement: keep `editKonfigInVim` editor flow (temp file from
current head blob; decode-validate through `repo_configs.Coder.Blob`,
return digest — `konfig_edit.go:111-156` unchanged). Replace the
`UpdateKonfig` commit with: under `Lock()`, build entry sku (id `konfig`,
type from current config, blob digest from editor, tai now), then
`configLog.Append(entry)`, then refresh in-memory `store_config`.
`dormant_edit.go` keeps sharing the helper.

**Step 3:** `just test-bats-targets edit_config.bats` → PASS.
`just test-bats-targets dormant_edit.bats` → update + PASS likewise.

**Step 4:** Commit: `feat(edit-config): append to config log instead of object commit`

---

## Phase C — bootstrap, init, genre removal

### Task 6: store_config bootstrap from the log; init writes root entry

**Files:**
- Modify: `go/internal/november/store_config/persist.go:158-230`
  (`loadMutableConfigStreamIndex`: replace the `FileConfig()` Sku load with
  config-log `Head()`; KEEP config-tags/types/repos cache loads untouched)
- Modify: `go/internal/november/store_config/main.go:205-206`
  (`AddTransacted` genres.Config case — becomes the in-memory refresh entry
  point called by Task 5)
- Modify: `go/internal/foxtrot/env_repo/genesis.go` + wherever init first
  populates konfig (rg `UpdateKonfig|genesisObjectIds`): write initial
  config blob + root log entry; stop writing the `FileConfig()` placeholder
  for the Sku (`genesis.go:125` — verify which consumers remain before
  removing the file write; config-tags/types/repos stay).

**Steps:** failing-first via BATS `init.bats` + `show_config.bats`
(unskip fresh-store test from Task 4):

1. Update `init.bats` expectations if init output changes (it should not —
   konfig commit lines may disappear from init output; capture with
   two-pass).
2. Implement; `just test-bats-targets init.bats show_config.bats` → PASS.
3. Go test: `just test-go-pkg ./internal/november/store_config/`.
4. Commit: `feat(config-log): bootstrap from log head; init writes root entry`

### Task 7: stop committing konfig objects (write path removal)

**Files:**
- Modify: `go/internal/oscar/store/reader.go:192-204` — KEEP the
  `genres.Config:` case for now (still serves in-memory reads) but it now
  returns the log-backed sku; remove once Task 8 kills the query path if no
  internal caller remains (check `ReadOneInto(ids.Config…)` callers).
- Modify: `go/internal/oscar/store/mutating.go:278-284` region — config
  objects no longer arrive here; assert via test that commits never see
  genre Config.
- Delete now-dead: `UpdateKonfig` / config branch of
  `createOrUpdateBlobDigest` (after Task 5 retarget, rg callers first).

**Steps:** compile-driven: `cd go && go build ./internal/...` per package;
then `just test-go` once at the end of the task (this is a unit-level sweep,
not the full bats lane). Commit:
`refactor(store): remove konfig object write path`

### Task 8: genre removal from the user surface + targeted error

**Files:**
- Modify: `go/internal/alfa/genres/main.go:198-201` — delete
  `config`/`konfig` cases from `Set` (falls through to unknown-genre error).
- Modify: `go/internal/juliett/queries/` builder — where a parsed term
  resolves to `genres.Config` (token path: `ids.TokenIsConfig`,
  `ids/main.go:258-284` ValidateSeqAndGetGenre), return:
  `config is no longer an object; use show-config / edit-config`.
  IMPORTANT: leave `ids/main.go` low-level parsing intact — box decode of
  log entries and legacy inventory lists depends on it. The error lives in
  the QUERY BUILDER ONLY.
- Verify-and-keep (internal/legacy): `ids/konfig.go`,
  `ids/types_builtin.go:110-113`, `ids/abbr.go:116`,
  `file_extensions/main.go:112`, `repo_configs/main.go:78`,
  `store_abbr/main.go:238` + `in_memory.go:43`, `import_plan/builder.go:115`,
  `remote_transfer/main.go:212` (already rejects Config),
  `local_working_copy/format_type.go:74`.

**Steps:**

1. Failing BATS: add to `show.bats` (or new `config_query_error.bats`):

       function show_konfig_query_errors { # @test
           run_dodder show :konfig
           assert_failure
           assert_output --regexp 'config is no longer an object; use show-config / edit-config'
       }

   plus bare `konfig` and `+konfig` variants.
2. Implement builder error; run `just test-bats-targets show.bats` → PASS.
3. Sweep remaining current_version BATS for `:konfig`/`+konfig`/`,konfig`
   usages (`rg konfig zz-tests_bats/current_version/`) — update each
   (export.bats `+e,konfig,t,z` query loses `konfig`; expected output loses
   the konfig line). `just test-bats-targets` per touched file.
4. Commit: `feat(queries): config genre leaves the user surface with targeted error`

---

## Phase D — migration, clone, version bump

### Task 9: reindex converts old konfig history

**Files:**
- Modify: `go/internal/oscar/store/reindex.go:21-88`
- Test: extend `zz-tests_bats/previous_versions/main.bats` expectations
  (Task 12 regenerates fixtures; here, code only)

**Implementation:** during reindex (`AllInventoryListObjectsAndContents`
iteration, reindex.go:51), objects with genre Config are diverted: instead of
`reindexOne` (stream index commit), collect them, sort by tai
oldest→newest, and `configLog.Append` each with preserved tai and blob
digest (mother chain re-derived from append order). Skip appending states
whose (blob digest, tai) already equal the current log head chain (idempotent
reindex). All other genres unchanged.

**Steps:** unit-test the divert-and-sort helper in isolation
(`just test-go-pkg ./internal/oscar/store/ -run TestReindexConfigDivert`);
full conformance comes from `previous_versions/main.bats` in Task 12.
Commit: `feat(reindex): convert konfig object history into config log`

### Task 10: clone seeding (direct transfer)

**Files:**
- Modify: `go/internal/uniform/commands_dodder/clone.go:64-114` (direct
  transfer branch: after pull, read SOURCE repo's config log head via its
  env, copy head blob into local blob store, append seed entry whose mother
  slot carries the source head's digSelf)
- Test: `zz-tests_bats/current_version/clone.bats` — new test: cloned repo's
  `show-config` equals source's; `show-config -history` shows one entry and
  reports the dangling mother ("history continues in the source repo").

**Network protocols (remote_http / websocket):** out of scope here — file a
GitHub issue via /eng:file-issue during this task ("clone config guidance
over network protocols") and add it to the session task list.

Commit: `feat(clone): seed config guidance from source head (direct transfer)`

### Task 11: show-config -history polish + full BATS

**Files:**
- Modify: `go/internal/uniform/commands_dodder/show_config.go`
- Test: `zz-tests_bats/current_version/show_config.bats`

Cover: latest output, by-digest output, `-history` ordering
(oldest→newest), dangling-mother message post-clone, error message for
unknown digest. Tightest assertions (no `--partial`; two-pass capture).
Run: `just test-bats-targets show_config.bats clone.bats edit_config.bats`.
Commit: `test(show-config): full integration coverage`

### Task 12: store version bump + fixtures

Follow the Version Bump Workflow from CLAUDE.md exactly, in this order:

1. `just test-bats-snapshot-version` (freeze current suite into
   `previous_versions/v15/`)
2. Bump: `go/internal/alfa/store_version/main.go` — add `V16`, set
   `VCurrent = V16` (check `VNext` semantics and `cmd/der-next/main.go:11`).
3. `just test-bats-update-fixtures`; review
   `git diff -- zz-tests_bats/previous_versions/` (expect: new v16 fixtures
   WITHOUT konfig objects in lists, WITH config_log file; `.fixtures.env`
   regenerated — `FIXTURE_KONFIG_SHA` consumers updated or retired).
4. Update `previous_versions/main.bats`: post-reindex assertion drops
   `konfig` from `show +e,konfig,t,z` (query itself must drop the token —
   it now errors) and instead asserts `show-config` + `-history` reflect the
   migrated chain from the v15 fixture.
5. Commit fixtures + assertions together:
   `feat(store): V16 — config leaves the object store`

### Task 13: final sweep

1. `rg -i konfig go/internal zz-tests_bats plugins .claude/skills docs` —
   classify every remaining hit: legacy-decode (keep), stale doc (fix),
   dead code (delete). Update both skill trees per the USE/DEV partition
   (root CLAUDE.md): `plugins/dodder/skills/` for CLI surface changes
   (query docs mentioning konfig, new show-config), `.claude/skills/` for
   internals.
2. Update repo `CLAUDE.md` Query System section if it references konfig.
3. FDR `docs/features/0020-config-as-non-object.md`: status → experimental.
4. Run the loose-ends skill, then `spinclass merge-this-session` (the
   pre-merge hook runs the full `just` lane — do NOT run `just test` first).

---

## Verified facts the plan relies on (from code reading)

- Inventory list log append mechanism: MultiWriter to blob store + log file,
  `inventory_list_store/blob_store_v1.go:74-131`; streaming read at 133-165.
  Append calls `FinalizeAndSign` with `envRepo.GetConfigPrivate().Blob` — the
  config log signs entries the same way.
- Coder hooks per type string: `inventory_list_coders/main.go:21-101`;
  config reuses the V2 entry verbatim (signed encode assert +
  `FinalizeAndVerify` decode). No new coder.
- `box_format/transacted.go:264` gates repo-pubkey/mother/object-sig
  emission behind a non-null object signature — which is WHY config entries
  are signed (an unsigned entry would lose its mother on encode).
- `SetMother()` sets the mother slot to the mother's object SIGNATURE
  (ed25519) — correct for signed config entries.
- Config bootstrap today: `store_config/persist.go:158-230` reads
  `FileConfig()` Sku cache + tags/types/repos caches; only the Sku source
  changes (now the config log head).
- `remote_transfer/main.go:212` already errors on Config genre.
- Reindex reads all inventory list objects (`reindex.go:51`) and preserves
  the lists on disk — migration is additive.

## Open risks (watch during implementation)

- `genesis.go:125` `FileConfig()` placeholder removal — verify no other
  consumer expects the file to exist.
- Log file framing: confirm whether closet writes hyphence-framed docs or
  bare box lines per append (Task 3's round-trip test pins it).
- `dormant_edit` semantics beyond the shared editor helper — if the dormant
  index couples to konfig commits anywhere else, surface it before Task 7.
