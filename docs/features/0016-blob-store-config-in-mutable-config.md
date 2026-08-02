---
status: exploring
date: 2026-05-31
promotion-criteria: |
  Blocked until madder FDR-0009 ("multi-store as a bonafide config
  type", umbrella code.linenisgreat.com/madder/issues/217) ships. Promote to
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

## Revised design (2026-05-31): multi default, NOT a separate seed store

The earlier direction below explored a **dedicated private "seed"
blob store** for the konfig blob. That was implemented and **rejected**:
dodder verifies every object's blob *through the default store*
(`fsck.go:226` → `VerifyBlob(GetDefaultBlobStore(), digest)`; `VerifyBlob`
is a generic `MakeBlobReader`), so a konfig blob living in a *separate*
store — discoverable or not — is unreachable via the default and breaks
`fsck`/`cat-ids`/transfer (proven: `fsck.bats` 6/13 failed).

**Chosen mechanism (D1):** the repo's default store is a madder `multi`
(FDR-0009, `!toml-blob_store_config-multi-v0`) whose `write_store` is a
**local** store and whose `read_stores` are digest-pinned fallbacks. The
konfig (and every object blob) is written **through the default** and
lands in the local `write_store`; `fsck`/`cat`/transfer route through
`GetDefaultBlobStore()` (= the multi) and find it via `MakeBlobReader`,
unchanged. The bootstrap read is local-first → **no dial**, and because
the `write_store` is always local the konfig is always locally readable
even when fallbacks are remote — the structural fix for the
remote-default bootstrap problem, with **no separate seed store**.

Multi references are **digest-pinned** (FDR-0009 requires it; bare ids
are rejected), so genesis mints each child store's FDR-0008 Phase-1
config digest and emits `Id.WithDigest` references — making dodder the
first persisted-reference call site (madder #220).

**The full, current implementation plan, the rejection rationale, the
open verifications, and the remaining phases live in
[dodder #223](https://code.linenisgreat.com/dodder/issues/223)** — start
there. The sections below predate this revision and describe the
abandoned seed-store framing; they are retained for history.

## Implementation note (2026-08-02): mirror considered and reverted

While implementing dodder#223 Phase 2, genesis was briefly changed to
author the default multi as `mode = "mirror"` (every write lands in
both the local write-store and the caller-named `-blob_store-id`
store) rather than this FDR's originally written `write_through`
design above. The reasoning at the time: `-blob_store-id <remote>`
likely means the user wants content actually replicated there, which
`write_through`'s read-only fallback role doesn't provide.

That deviation reintroduced a real blocker: mirror mode requires every
member to write the identical digest, which requires the identical
hash type (recently made a loud, enforced error rather than a silent
corruption/crash by madder#268). A local default store defaulting to
blake2b256 can then never mirror a legacy single-hash remote (e.g. a
pre-existing rsync.net-style sha256 store) — which is the literal
original real-world case that motivated this whole investigation
(dodder#365/#366). `write_through` has no such constraint: each
member (write-store or read-store) uses its own native hash type
independently, with no cross-member agreement at all.

Genesis has been reverted to `write_through` (this FDR's original D1
design, local as `write_store`, caller-named store(s) as
`read_stores`) for this reason. Options considered for the
cross-hash-type case, captured here in case this needs revisiting:

- **A. `write_through`, local as write-store, remote(s) as read-only
  fallback (chosen).** No cross-member hash-type constraint; also the
  design this FDR originally specified. Tradeoff: genesis's own new
  writes, and this repo's writes generally, land ONLY in the local
  write-store — a caller-named remote never receives new content
  automatically (it's a fallback for existing/historical content, not
  a live replication target). Ongoing replication to a remote, if
  wanted, would be a separate feature (e.g. `push`), not something the
  default-store construction itself provides.
- **B. Hybrid: `mirror` when hash types already match, `write_through`
  when they don't.** Rejected — in the mismatched branch, local must
  still be the write-store (else the bootstrap-config-read guarantee
  breaks again), which makes that branch identical to option A. The
  only thing a hybrid buys is keeping mirror's "both stores get every
  write" behavior for the matching-hash-type case, at the cost of two
  different behaviors an operator has to understand depending on which
  remote they picked.
- **C. Let the shared local write-store adopt a mirrored remote's hash
  type.** Rejected — `default-local` is XDG-scope-shared across repos
  (D1 decision #4 above); making its hash type follow whichever remote
  a repo happens to name would risk collisions for other repos in the
  same scope wanting a different type. Would require either per-repo
  local stores (reverses D1 decision #4) or hash-type-suffixed shared
  stores (`default-local-sha256`, …) — the most invasive option, for
  the least benefit.
- **D. Reject the combination outright**, clear dodder-side error at
  genesis time. Simplest, but directly fails the original motivating
  case (dodder#365/#366) — a repo whose only blob store is a legacy
  single-hash remote would simply be unsupported.

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
code.linenisgreat.com/madder/issues/217). FDR-0009 adds the
`!toml-blob_store_config-multi-v0` hyphence-typed config so a repo can
have a multi-default store configured *in madder* (read fallback as
config, resolved by madder), retiring per-command fallback and runtime
Multi-building.

- madder FDR-0009 is **proposed**, not shipped.
- FDR-0009 depends on madder FDR-0008 Phase 2 (digest-bearing
  blob-store-ids, code.linenisgreat.com/madder/issues/198) — which is
  **done**.
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
