# Repo Identity T3-C: positional arg = location handle, config-seed dropped

- Date: 2026-06-30
- Issue: dodder #294 / FDR-0021 (T3-C — resolves the *local* half of open Q2)
- Builds on: the merged first increment (T1 renderer, T2 info-repo id) + T5
  (handshake verified pubkey-keyed).

## Decision (linchpin)

A new repo's **location handle** (FDR-0019 `scoped_id`) is named by a
**required positional** on `init`/`clone` (optional on `init-default`), parsed
with the full grammar — scope comes from the spelling (`work` = user,
`.default` = cwd, `//backup` = system). It is the **sole** namer of a new repo.
The config-seed id is **never written** (decoder kept). `-repo_id` /
`DODDER_REPO_ID` are **removed from the init naming path**; they survive only for
*addressing existing* repos in operate commands (`show`, `info-repo`, `pull`,
…; FDR-0019, out of scope). The in-graph repo-object <-> pubkey resolver (Q2's
*remote* half) stays **deferred** (gated on remote transport, not implemented).

Rationale: `-repo_id` did two jobs — *name a new repo* (init) and *address an
existing repo* (operate). Today the init positional names the *deprecated*
config-seed id while `-repo_id` names the location, so the two are passed side
by side and deliberately disagree (`init -repo_id .default test-repo-id`
everywhere in the suite). That overlap is the root of #294. Splitting the jobs —
positional names new repos, `-repo_id` selects existing ones — gives exactly one
namer for each, with no overlap.

## Per-command behavior

- **`init <handle>`** (required positional): parse -> `config.RepoId`; create
  `repos/<EffectiveName>/`. No config-seed written. A non-auto
  `-repo_id`/`DODDER_REPO_ID` is **rejected** with an error pointing at the
  positional (same mechanism init-workspace already uses — `-repo_id` is a
  global config flag, so we reject it rather than drop it from the flagset).
- **`clone <handle>`** (required positional): same for the new *local* repo.
  VERIFY during impl: clone's *remote* is named via `-url`/`-proto`/
  `RemoteTransfer`, not `-repo_id` — confirm removing `-repo_id`'s naming role
  doesn't disturb remote selection.
- **`init-default [handle]`** (optional — its no-arg/unattended bootstrap
  purpose is preserved): given -> location (full grammar); omitted -> cwd
  `.default` (`CwdDefault()`). `deriveRepoIdFromDir` + the
  `initDefaultRepoIdUnsafe` regex are **removed** (they only fed config-seed);
  drop now-unused `regexp`/`strings` imports. No `-repo_id` naming.
- **`init-workspace`** (EXCEPTION, unchanged): forced cwd `.default`, `-repo_id`
  already rejected; the positional remains the parent-pointer blob-store name
  (`pointerId := "." + workspaceRepoIdString`), NOT a location. It inherits only
  the config-seed drop via `OnTheFirstDay`. "Positional = location" structurally
  cannot apply without breaking the cwd-singleton invariant.

## genesis.go (`OnTheFirstDay`)

- Remove the `repoIdString` parameter and the config-seed write (the
  `repoId.Set(repoIdString)` parse + `cmd.GenesisConfig.Blob.SetRepoId`,
  genesis.go:112-118; drop the now-unused `bravo/ids` import).
  `OnTheFirstDay(req)` uses `config.RepoId` (already set by each command's `Run`
  from the required positional) to drive `MakeDefaultAndInitialize` /
  `repo_id.EffectiveId` as today.
- Genesis config's `RepoId` field + decoder retained (back-compat for old
  `config-seed` files); never populated for new repos. `SetRepoId` setter kept
  but uncalled.

## Where Run sets config.RepoId

Mutation goes through the `*Config` **pointer**
(`req.Utility.GetConfigAny().(*repo_config_cli.Config)` — as init-default
already does), since `repo_config_cli.FromAny` returns a copy. Parse the
positional with `config.RepoId.Set(arg)` (the `flag.Value` path). For
init/clone, before parsing, reject a non-auto `config.RepoId`
(`!repo_id.IsAuto(...)`) since that means `-repo_id`/`DODDER_REPO_ID` was set. A
small shared helper on the `Genesis` component keeps init/clone DRY.

## Test churn (verify via the `--no-sandbox` bats lane)

- `common.bash` helpers (`run_dodder_init`, `_disable_age`, `_sha256`,
  `_disable_age_xdg`): move the location from `-repo_id .default <config-seed>`
  to the positional — `init … .default` (drop `-repo_id .default` and the
  `test-repo-id`/`test` config-seed positional). Default arg becomes `.default`.
- `info_repo.bats`: drop the config-seed positional from direct
  `init … test-repo-id` sites; `info_config_immutable` no longer asserts
  `id = "test-repo-id"` (config-seed gone); `info-repo -repo_id work id`
  (addressing) **stays**; `info_repo_id_shows_handle_at_pubkey` becomes
  `init … work` (positional location) + `info-repo -repo_id work id`
  -> `work@ed25519_pub-…`.
- Add: `init -repo_id other .default` errors (rejection); `init work` then
  `repos/work/` exists.
- Migration conformance (`previous_versions/main.bats`): old fixtures' populated
  config-seed id still decodes.
- `complete.bats` `complete_subcmd` is unaffected (no new subcommand), but the
  `init`/`clone` arg descriptions change — grep for asserted help text.

## Docs / help

- `init`/`clone`/`init-default` arg descriptions: the positional is the repo's
  **location handle** (scope via spelling). Drop "used for remote
  synchronization" (the pubkey does that).
- `docs/man.1` (if any init/clone page exists) + plugin USE-skills mentioning
  `init -repo_id` for naming.

## Deferred (tracked follow-ups)

- In-graph repo-object <-> pubkey resolution for remotes (Q2's remote half) —
  gated on remote transport.
- Foreign pubkey -> handle resolution for provenance/remote display.

## Rollback

The config-seed `RepoId` field + decoder are retained throughout. Revert =
restore the `repoIdString` param + `SetRepoId` write in `OnTheFirstDay` and the
`-repo_id`-as-namer behavior in each command's `Run`.
