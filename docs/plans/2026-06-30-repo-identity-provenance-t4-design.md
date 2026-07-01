# Repo Identity T4: provenance display self vs foreign

- Date: 2026-06-30
- Issue: dodder #294 / FDR-0021 (T4)
- Builds on: T1 renderer (`echo/repo_identity.Render`), T2/T3 merged.

## Key fact (from investigation): T4 is DISPLAY-ONLY

The originating repo's ed25519 pubkey is **already** stamped on every object
(`internal/delta/objects` `pubRepo markl.Id`, set at
`golf/object_finalizer/main.go:92`), **already** persisted + decoded in the
binary codec (`hotel/stream_index/binary_{encoder,decoder}.go`,
`key_bytes.RepoPubKey`), and **already** rendered (bare) by
`object_metadata_box_builder.AddRepoPubKey` under `-print-sigs`
(`box_format/transacted.go:~265`). No stamping, no codec work.

## Decision

Under `-print-sigs`, distinguish provenance:

- **Self** (object's `GetRepoPubKey()` == the local repo's pubkey):
  render `<handle>@<pubkey>` via `repo_identity.Render(selfHandle, objPubKey)`
  (full pubkey, symmetric with `info-repo id`).
- **Foreign** (pubkey present, != local): bare/abbreviated pubkey — **current
  behavior**, unchanged.
- **Legacy / null** (`GetRepoPubKey().IsNull()`): unchanged (no provenance
  pubkey emitted), as today.

## Mechanism

- Add two fields to `box_format.BoxTransacted`: `selfPubKey markl.Id` and
  `selfHandle string` (both zero by default), plus a setter
  `SetSelfProvenance(pubkey mad_domain_interfaces.MarklId, handle string)`
  (mirrors the existing `SetAbbr`).
- At the `AddRepoPubKey` call site, branch:
  ```
  objPK := metadata.GetRepoPubKey()
  if !objPK.IsNull() && !format.selfPubKey.IsNull() && markl.Equals(objPK, format.selfPubKey) {
      builder.AddRepoIdentity(repo_identity.Render(format.selfHandle, objPK)) // self: <handle>@<pubkey>
  } else {
      builder.AddRepoPubKey(metadata, format.abbr.PubKey.Abbreviate)          // foreign/legacy: current
  }
  ```
  (Add a small `AddRepoIdentity(s string)` to `object_metadata_box_builder` that
  emits the same field key as `AddRepoPubKey` with a pre-rendered value, or
  parametrize the existing method — implementer's call, keep the box field key
  identical so downstream parsing is unaffected.)
- **Set self-provenance only at the user-facing DISPLAY printer** in
  `romeo/local_working_copy/printers.go` (`MakePrinterBoxArchive`, via
  `setBoxSelfProvenance`). CORRECTION (verified during impl): `main.go:~133`'s
  box is NOT a display site — it feeds the inventory-list wire coder
  (`typed_blob_store` → `doddish` → export & persisted lists), so setting
  self-provenance there would rewrite persisted/exported bytes and break import
  round-trip. It (and every other internal / archive constructor:
  `oscar/store/reindex`, `india/config_log`, `india/inventory_list_store`,
  `november/store_config`, `sierra/remote_http`,
  `tango/command_components_dodder/inventory_lists`) leaves it UNSET →
  graceful degradation to today's bare (purpose-prefixed) output; wire form
  stays byte-identical.

## Open thread for the implementer: handle source

The local **pubkey** is reachable via `env_repo` (config-public:
`GetImmutableConfigPublic().GetPublicKey()`). The **handle** is the scoped repo
id (`config.RepoId.String()` from `repo_config_cli.Config`). Determine the
cleanest source reachable at the `local_working_copy` box construction:
- preferred: the CLI `config.RepoId.String()` if `local_working_copy` (or its
  env) carries it;
- fallback: `env_repo`'s repo name (`repo_id.EffectiveName` of the repo the env
  was rooted at) — the name part of the handle.
If neither is cleanly reachable without invasive plumbing, surface that before
proceeding (the self render needs *some* handle; absent one, leaving
self-provenance unset = bare pubkey is the graceful fallback, but defeats T4).

## Tests (no-sandbox lane)

- `import.bats` (two-repo: import objects stamped with a different pubkey):
  under `-print-sigs`, an imported (foreign) object renders the **bare**
  pubkey; a locally-committed (self) object renders `<handle>@<pubkey>`.
- A focused box_format/object_metadata_box_builder unit test if practical
  (self branch vs foreign branch given a set selfPubKey).

## Tuning lever

Self renders the **full** pubkey (per the symmetric-with-`info-repo id`
choice). If mixed self/foreign listings look unbalanced (self full vs foreign
abbreviated), abbreviating self's pubkey within `<handle>@<…>` is the lever;
revisit against real output.

## Rollback

Purely additive (new fields default-empty; unset = current behavior). Revert =
drop the fields/setter and the branch at the call site.
