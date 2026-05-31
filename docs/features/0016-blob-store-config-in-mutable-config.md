---
status: proposed
date: 2026-05-31
promotion-criteria: |
  Blocked until madder FDR-0009 ("multi-store as a bonafide config
  type", umbrella amarbel-llc/madder#217) ships. Promote to
  `experimental` once: (1) dodder's mutable repo config owns its
  blob-store configuration end to end — the store set + order (already
  present), the default store, and the digest-bearing id of the store
  holding the seed/config blob; (2) the genesis-time authoring of
  madder per-store `blob_store-config` files
  (`writeBlobStoreConfig*`) is removed; (3) the 5 `clone_history_*`
  bats tests plus the init/init-workspace flows stay green; (4) the
  runtime Multi-building accessors (`GetReadBlobStore` /
  `GetLocalReadBlobStore`) are either retired in favor of a madder
  multi-default store or explicitly retained with a documented reason.
---

# Blob-Store Config in the Mutable Config

> **Status (2026-05-31):** Proposed and **blocked on madder FDR-0009**.
> This FDR records design intent; no dodder code changes land until the
> madder multi-store config type is available. The eager-remote-dial
> bug that motivated this investigation is already fixed and merged
> (see FDR-0015 and the local-only bootstrap read); this FDR is the
> principled consolidation, not an urgent fix.

## Problem Statement

dodder's blob-store configuration is split across three homes, and the
split is an artifact of history rather than design. The config-seed was
authored before madder and dodder were decoupled; madder now owns blob
stores entirely (discovery and per-store config), yet dodder still
reaches across that boundary at genesis and rebuilds blob-store topology
at runtime. The result is a chicken-and-egg on the bootstrap config-blob
read (the blob-store order is decoded from the very blob the read needs
to find), no first-class place to record which store holds a given
blob, and a config-seed that straddles a madder/dodder line that no
longer exists.

## Current State

- **madder owns blob stores** post-decoupling: it discovers them (CWD
  walk-up of `.madder/` directories plus XDG) and owns each store's
  `blob_store-config` file. dodder consumes madder's `blob_store_env`.
- **dodder's mutable repo config** already carries the store **order
  list**: `repo_configs.V2.BlobStores []blob_store_id.Id`
  (`go/internal/charlie/repo_configs/v2.go:12-18`). This is the
  dodder-owned blob-store config that exists today.
- **dodder's seed/genesis config** holds only identity —
  `StoreVersion`, `RepoId`, `InventoryListType`, `ObjectSigType`, and
  keys (`genesis_configs.TomlV2Common`,
  `go/internal/charlie/genesis_configs/toml_v2.go:13-19`). It carries
  **no** blob-store fields.
- **Pre-decoupling residue:** at init, dodder *authors madder's*
  per-store `blob_store-config` files via
  `writeBlobStoreConfigIfNecessary` / `writeBlobStoreConfigInit`
  (`go/internal/foxtrot/env_repo/genesis.go:179-262`) — dodder reaching
  into madder's config domain.
- **Runtime:** `SetBlobStoreOrder(repoConfig.GetBlobStores())` runs
  *after* config load (`go/internal/romeo/local_working_copy/main.go:153`),
  and `GetReadBlobStore` builds a madder `Multi` on the fly
  (`go/internal/foxtrot/env_repo/main.go`). The bootstrap config-blob
  read cannot know the order — it is decoded from the very blob being
  read (`config.configRepo` is assigned inside `loadMutableConfigBlob`,
  `go/internal/november/store_config/persist.go:512`) — so a local-only
  read is the shipped floor (FDR-0015).

## Design

### Direction

Consolidate dodder's blob-store configuration into the **mutable repo
config**, and lean on madder FDR-0009 for topology:

1. **Stop dodder authoring madder per-store configs at genesis.**
   Delegate store creation to madder (`madder init` / referencing
   existing stores by FDR-0008 digest-bearing id), removing the
   `writeBlobStoreConfig*` residue.
2. **The mutable config owns dodder's blob-store config end to end:**
   the store set + order (already `BlobStores`), the **default store**,
   and — subsuming dodder#223 — the **digest-bearing id of the store
   holding the seed/config blob** (provenance), so steady-state and
   clone reads resolve directly to the holding store instead of probing
   every fallback.
3. **Prefer a `multi` blob-store config as the default (madder
   FDR-0009).** FDR-0009 makes `multi` a first-class
   `blob_store-config`, discriminated by its hyphence type tag
   `!toml-blob_store_config-multi-v0` — there is no `store_type` field;
   the tag *is* the type, per the established blob_store_config pattern
   (`!toml-blob_store_config-v3`, `!toml-blob_store_config-pointer-v1`,
   …). Authored via `madder init-multi`, its write-through body names a
   `write_store` and an ordered list of digest-bearing `read_stores`.
   If dodder's
   configured default store resolves to such a multi config, the
   read-fallback set and order are madder config (resolved by madder
   via the two-pass store-map build), so dodder stops building a
   `Multi` at runtime — `GetReadBlobStore` / `GetLocalReadBlobStore`
   retire. The digest-bearing references double as the provenance
   from (2).
4. **Resolve the bootstrap read deterministically.** The very first
   config-blob read happens before any dodder config is loaded, so it
   can't use the consolidated config; today the shipped local-only
   read (FDR-0015) is the floor. A stronger option, raised in review,
   is to give dodder's critical seed/config blobs their own home — see
   the dedicated-private-store open question below — so the bootstrap
   path never depends on discovery/probing at all.

### Relationship to dodder#223

dodder#223 ("pin blob provenance so reads resolve directly instead of
probe-dialing") is **subsumed** by this FDR: provenance becomes a field
of the consolidated mutable config rather than a sidecar file or a
mutation of the immutable seed.

## Dependencies & Sequencing

This is **one half of a cross-repo move**; the other half is madder
FDR-0009 "multi-store as a bonafide config type" (umbrella
amarbel-llc/madder#217). FDR-0009 adds the
`!toml-blob_store_config-multi-v0` hyphence-typed config so a repo can
have a multi-default store configured *in madder* (read fallback as
config, resolved by madder), retiring per-command fallback and runtime
Multi-building.

- madder FDR-0009 is **proposed**, not shipped.
- FDR-0009 depends on madder FDR-0008 Phase 2 (digest-bearing
  blob-store-ids, amarbel-llc/madder#198) — which is **done**.
- Therefore this dodder FDR is **proposed and sequenced after**
  FDR-0009. The dodder design cannot be finalized until FDR-0009's
  shape lands (it determines how much topology is madder's multi-config
  versus dodder's mutable config).

## Migration

- `repo_configs` evolves V2 → V3 with the new fields (default store,
  seed-blob provenance). Old configs stay decodable per the
  horizontal-versioning pattern (`design_patterns-horizontal_versioning`).
  A `store_version` bump may be required; if so, follow the repo's
  fixture/version-bump workflow (snapshot previous version, bump
  `VCurrent`, regenerate fixtures) and keep old versions decodable for
  migration.
- The genesis `writeBlobStoreConfig*` residue is removed; existing
  repos migrate (their already-written madder configs remain valid;
  dodder simply stops authoring new ones).

## Open Questions

- **madder/dodder boundary.** How much topology is madder's
  multi-config versus dodder's mutable config? Likely madder owns the
  multi store config and dodder references it as the default plus
  records seed provenance — but FDR-0009's final shape decides this.
- **Seed minimalism.** Should the seed/genesis config carry *zero*
  blob-store data (pure identity/keys), or a minimal pointer needed for
  the very first read?
- **Bootstrap benefit.** Does provenance-in-the-mutable-config-blob
  actually help the bootstrap read (that blob is itself read *through*
  stores), or is the local-only floor plus a madder multi-default
  store's discovery sufficient there?
- **Dedicated private store for the critical seed/config blobs
  (review feedback).** Should a dodder repo bootstrap a
  *completely-private madder blob store* reserved for its critical
  seed/config files, separate from the general blob store(s)? That
  would give the bootstrap config read a known, private, always-local
  home and remove its dependence on discovery/probing entirely —
  a stronger guarantee than the local-only read floor. Open questions
  this raises: how that private store is named/located so it's found
  with zero config, whether it participates in clone/pull/push, and how
  it composes with a `multi` default (the critical store would sit
  outside the multi's read-fallback set).

## Key Files

| File | Role |
|------|------|
| `go/internal/charlie/repo_configs/v2.go` | Mutable repo config — the target container (`BlobStores` lives here; gains default + provenance). |
| `go/internal/charlie/genesis_configs/toml_v2.go` | Seed/genesis config — identity only today; candidate to stay blob-store-free. |
| `go/internal/foxtrot/env_repo/genesis.go` | `writeBlobStoreConfig*` residue (`:179-262`) to remove. |
| `go/internal/foxtrot/env_repo/main.go` | `GetReadBlobStore` / `GetLocalReadBlobStore` runtime Multi-building — retire under FDR-0009. |
| `go/internal/november/store_config/persist.go` | Config-blob read/write; `configRepo` set inside `loadMutableConfigBlob`. |

## More Information

- dodder FDR-0015 — Multi-Store Blob Lookup
  (`docs/features/0015-multi-store-blob-lookup.md`): the shipped
  read-fallback and the local-only bootstrap floor this FDR builds on.
- dodder#223 — "Eager remote (SFTP) dial on every read: pin blob
  provenance…": subsumed by this FDR.
- madder FDR-0009 — "multi-store as a bonafide config type": the
  sequencing partner this FDR depends on.
- madder#217 — umbrella sequencing FDR-0008 → FDR-0009.
- madder#198 — FDR-0008 Phase 2 (digest-bearing blob-store-ids), done;
  the mechanism provenance is recorded with.
