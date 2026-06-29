# Repo Identity (Pubkey-Anchored) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use eng:subagent-driven-development to implement this plan task-by-task.

**Goal:** Make the repo ed25519 pubkey the human-facing absolute identity (`info-repo id` shows `<handle>@<pubkey>`), stop writing the config-seed id at init, and render object provenance as `<handle>@<pubkey>` for self / bare pubkey for foreign — the FDR-0021 core slice (dodder #294).

**Architecture:** The pubkey is already the sync-handshake peer identity (`control.PublicKey`, `assertPublicKeyMatches`) and already stamps provenance (`object_finalizer` writes `GetRepoPubKey` = `config.GetPublicKey()`). So this slice is mostly *presentation + deprecation*: a shared `<handle>@<pubkey>` renderer, rewire `info-repo id`, stop populating the config-seed `RepoId`, and polish provenance display. No wire-format or binary-codec change.

**Tech Stack:** Go (dodder, NATO-tier layout), madder `pkgs/markl` + `pkgs/scoped_id`, BATS (nix lane). Base: madder go/v0.4.0 + `internal/0/hyphence`.

**Rollback:** Purely additive + presentation. config-seed `RepoId` field is retained and decodable; nothing is deleted. Revert = restore the two changed print/finalize sites and re-enable the init `SetRepoId` call (single-commit revert). N/A for wire format (unchanged).

**Surprising findings from the touchpoint map (read before starting):**
- Handshake **already** sends + pins/compares the pubkey (`sierra/remote_proto/handshake.go:81,111-119,256-277`); `control.RepoId` is legacy/fallback. Task 5 is mostly *verification + tests*, not new code.
- Provenance **already** stamps the pubkey (`golf/object_finalizer/main.go:299-303`, `hotel/inventory_list_coders/main.go:35-38`); the `RepoPubKey` markl field already persists. Only *display* changes (Task 4).
- Design doc: `docs/plans/2026-06-29-repo-identity-pubkey-design.md`. Touchpoint map: this plan's citations.

---

### Task 1: Shared `<handle>@<pubkey>` identity renderer

**Promotion criteria:** N/A (new helper).

**Files:**
- Create: `go/internal/echo/repo_identity/main.go` (new package; tier `echo` is below `golf`/`uniform` consumers and above `bravo/ids`). Confirm tier placement compiles against the NATO DAG; if `scoped_id`/`markl` force a different tier, place accordingly and note it.
- Test: `go/internal/echo/repo_identity/main_test.go`

**Decision (design §Decisions.2, map §8):** render as a **display-only string** `handle + "@" + pubkey.StringWithFormat()` — do NOT set the handle as a registered markl purpose. Handle may be empty (e.g. provenance with no local handle) → then render the bare pubkey with no `@`.

**Step 1 — failing test.** Table test for `Render(handle string, pubkey mad_domain_interfaces.MarklId) string`:
- `("default", <ed25519_pub-…>)` → `"default@ed25519_pub-…"`
- `("", <ed25519_pub-…>)` → `"ed25519_pub-…"` (no leading `@`)
- `(".work", <pub>)` → `".work@ed25519_pub-…"`
Use a fixed test markl.Id (construct via `markl.Id.Set("ed25519_pub-…")` with a known fixture value; see `assertPublicKeyMatches` for the `.Set` pattern).

**Step 2 — verify fail:** `just test-go-pkg ./internal/echo/repo_identity/` → FAIL (undefined `Render`).

**Step 3 — implement:** `Render` does the concat; if `pubkey == nil || pubkey.IsNull()` return `handle` (or empty); if `handle == ""` return `pubkey.StringWithFormat()`.

**Step 4 — verify pass:** `just test-go-pkg ./internal/echo/repo_identity/` → PASS.

**Step 5 — commit:** `feat(repo_identity): add <handle>@<pubkey> renderer (#294)`

---

### Task 2: `info-repo id` emits `<handle>@<pubkey>`

**Promotion criteria:** Old `info-repo id` (config-seed name) is replaced; criterion to keep new form = the headline BATS proof (Task 3) passes.

**Files:**
- Modify: `go/internal/uniform/commands_dodder/info_repo.go:108-109` (the `case "id"` print site; env from `cmd.MakeEnvRepo(req, false)` at :81).
- Test: `zz-tests_bats/current_version/info_repo.bats`

**Discovery step (do first):** find how to get the current repo's **location handle** (scoped_id name) from `env`. Candidates: the resolved `-repo_id` (`repo_config_cli.Config.RepoId`, a `scoped_id.Id`, `charlie/repo_config_cli/main.go:134`) reachable from the command request, or an accessor on the env. The handle string = the scoped_id's name/spelling (`scoped_id`'s `String()`/name method — verify in `madder/go/pkgs/scoped_id`). Write a 1-line note in the commit if the accessor differs from the guess.

**Step 1 — failing BATS test** in `info_repo.bats` (mirror `scoped_repos.bats:21-56` style): init a repo with location id `work`; `run_dodder info-repo -repo_id work id`; assert output matches `^work@ed25519_pub-` (use `assert_output --regexp` since the pubkey is non-deterministic per fresh-store).

**Step 2 — verify fail:** `just test-bats-targets info_repo.bats` → FAIL (still prints config-seed id).

**Step 3 — implement:** replace line 109 with
```go
case "id":
	env.GetUI().Print(
		repo_identity.Render(<handle>, configPublicBlob.GetPublicKey()),
	)
```
wiring `<handle>` from the discovery step.

**Step 4 — verify pass:** `just test-bats-targets info_repo.bats` → PASS. Also `just test-go-pkg ./internal/uniform/commands_dodder/` compiles.

**Step 5 — commit:** `feat(info-repo): id shows <handle>@<pubkey> (#294)`

---

### Task 3: Stop writing the config-seed id at init (headline proof)

**Promotion criteria:** config-seed `RepoId` no longer drives identity; field retained + decodable. Done when the headline proof passes.

**Files:**
- Modify: the genesis/init flow that calls `(*TomlV2Private).SetRepoId` (`go/internal/bravo/genesis_config_blobs/toml_v2.go:119`). Find the caller in the init path (`uniform/commands_dodder/init*.go` / the BigBang/genesis setup) and stop populating it for NEW repos. Keep the setter + the `toml:"id"` decoder (`toml_v2.go:16`) intact.
- Test: `zz-tests_bats/current_version/scoped_repos.bats` (or a new `repo_identity.bats`)

**Step 1 — failing BATS test (the FDR headline proof):** create a repo whose **location id is `default`** but whose **config-seed id would historically differ** (simulate the legacy divergence: either via a fixture with a populated differing config-seed id, or by asserting new repos no longer carry one). Assert: `info-repo id` → `default@ed25519_pub-…`; `info-repo repos`/discovery lists `default`; `-repo_id default` resolves; NO surface reports a contradictory config-seed name. (Two-pass: capture actual, then tighten with `--regexp`.)

**Step 2 — verify fail:** `just test-bats-targets <file>.bats` → FAIL (new repo still writes a config-seed id / surfaces it).

**Step 3 — implement:** remove/guard the `SetRepoId` call at init so new repos leave `RepoId` zero. Verify decode of an EXISTING populated config-seed still works (don't touch the decoder).

**Step 4 — verify pass:** `just test-bats-targets <file>.bats` → PASS. Confirm legacy decode: a `previous_versions/v*` fixture with a populated config-seed id still loads (run the migration-conformance lane).

**Step 5 — commit:** `feat(genesis): stop writing config-seed id at init; keep decodable (#294)`

---

### Task 4: Provenance display — self `<handle>@<pubkey>`, foreign bare pubkey

**Promotion criteria:** N/A (display only; stamping unchanged).

**Files:**
- Modify: `go/internal/echo/object_metadata_fmt/main.go:51-59` (`AddRepoPubKey`) + `makeMarklIdField` (:109-124).
- Test: `zz-tests_bats/current_version/` box-format display test (find an existing provenance/box test; else add one) + a unit test if `object_metadata_fmt` has `_test.go`.

**Decision:** `AddRepoPubKey` compares the object's `GetRepoPubKey()` bytes to *this repo's* `config.GetPublicKey()` bytes (pattern: `assertPublicKeyMatches`, `bytes.Equal(expected.GetBytes(), actual.GetBytes())`). If equal → render `repo_identity.Render(<handle>, pubkey)`. If not (foreign) → render the bare abbreviated pubkey (existing `makeMarklIdField` behavior). Legacy name-stamped objects (no `RepoPubKey`) render as today.

**Step 1 — failing test:** an object committed by THIS repo shows `<handle>@<pubkey>` in box output; an object with a different `RepoPubKey` shows the bare pubkey. (Construct the foreign case via a clone/pull fixture, or a unit test feeding two metadata values.)

**Step 2 — verify fail.** **Step 3 — implement** the self/foreign branch (needs `config.GetPublicKey()` + the handle threaded into the formatter; check how `object_metadata_fmt` is constructed and whether it already has access to the repo config/handle — thread it if not). **Step 4 — verify pass.**

**Step 5 — commit:** `feat(box_format): provenance shows <handle>@<pubkey> for self, pubkey for foreign (#294)`

---

### Task 5: Verify the handshake keys identity off the pubkey (tests, likely no new code)

**Promotion criteria:** Once verified, the legacy `control.RepoId` field can later be retired (separate follow-up); keep it for now (fallback / old-peer interop).

**Files:**
- Inspect: `go/internal/sierra/remote_proto/{handshake.go,control.go,client.go,server.go}`. Confirm `serverCaps.PublicKey` is the authoritative peer identity and `RepoId` is not used for an identity *decision* (only metadata/fallback). If any identity decision keys off `RepoId`, switch it to pubkey.
- Test: `zz-tests_bats/current_version/clone.bats` (or new `remote_identity.bats`), using `bootstrap()` / `run_clone_default_with()` (`lib/clone.bash`).

**Step 1 — test:** two repos both locally named `default` (distinct pubkeys) remain distinguishable across a clone/pull handshake (no identity collision). If pinning is exercised, a pubkey mismatch is rejected (`assertPublicKeyMatches`).

**Step 2 — verify:** run the test; if it passes with no code change, document that the handshake is already pubkey-identified. If it fails, make the minimal change to key the decision off `PublicKey`.

**Step 3 — commit** (test-only or test+fix): `test(remote_proto): peer identity keyed off pubkey, not name (#294)`

---

### Task 6: Whole-suite gate + wrap

**Step 1:** `just complete.bats` check — if any subcommand surface changed, none added here, so likely N/A; confirm `complete_subcmd` still passes.

**Step 2:** Run the full lane via the merge pre-merge hook (do NOT run `just test-bats` standalone — CPU-coordinate with peers; the merge hook is the gate).

**Step 3:** Open follow-up issues for the deferred items (FDR-0021 open Q2; provenance-history migration; foreign pubkey→handle resolution) and reference them from #294 / FDR-0021. Do NOT auto-close #294 unless the full FDR is done — this is the core slice; close only if the driver confirms the slice satisfies #294, else comment with what landed + what's deferred.

---

## Tuning levers
- **Pubkey abbreviation length** in display (`ed25519_pub-9ft3…`). Current: existing markl abbreviation default. Change signal: collisions or human confusion.

## Deferred (tracked)
- FDR-0021 open Q2 (location vs in-graph repo-object namespace).
- Provenance-history migration (mixed name/pubkey stamps).
- Foreign pubkey→handle resolution for provenance/remotes.
