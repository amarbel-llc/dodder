# env_repo

Repository environment managing configuration, directory layout, blob stores, and locking.

## Key Types

- `Env`: Main repository environment with config and directory layouts
- `Options`: Repository initialization options (base path, permit flags)
- `BlobStoreEnv`: Blob store environment for content storage

## Features

- Loads and manages genesis configuration (private/public)
- Creates directory layout for blobs and repository data
- Manages file locking via locksmith pattern
- Handles XDG base directory paths
- Provides blob store access and inventory list storage
- Supports cache reset operations

## Genesis blob store authoring (FDR-0016 D1 / #223)

Blob store implementations and config types live in amarbel-llc/madder
(`internal/charlie/blob_store_configs` there); this package only AUTHORS
configs during genesis (`genesis.go`):

- `writeBlobStoreConfigIfNecessary` runs on every genesis and authors the
  repo's default store as a `mode="write_through"` madder multi named
  `default-<EffectiveName(repoId)>` (repo-scoped name so same-scope repos
  don't collide on a flat "default").
- The multi's write store is always `storeNameLocal` (`default-local`),
  a scope-shared local store created or reused by `ensureLocalWriteStore`
  --- a second repo initialized in the same scope reads the existing
  config's digest back instead of re-minting.
- A caller-named `-blob_store-id` store becomes a digest-pinned READ-ONLY
  fallback (`ReadStores`), never a write target. `ensureBlobStoreDigest`
  mints a Phase-1 (FDR-0008) digest in place for legacy configs that
  predate it, resolving the store through the scope-aware
  `blobStoreEnv.GetBlobStore` lookup (the id can live in a different
  scope than the repo's own).
- Exception: `BlobStoreConfigInit` (init-workspace's pointer-store flow,
  #200) bypasses the multi entirely --- Genesis pins the blob store order
  directly to the caller-named store.
- Genesis pins `SetBlobStoreOrder([multiId])` so madder's own
  default-selection (alphabetical over discovered stores) can't silently
  win (#365).
- Re-init reuse tradeoff: a surviving `default-local`/multi config from a
  prior init is reused as-is, so re-initializing with a DIFFERENT
  `-blob_store-id` keeps the old multi's member set.
