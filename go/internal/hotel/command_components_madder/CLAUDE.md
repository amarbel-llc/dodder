# command_components_madder

Blob-store-arg dispatch components used by dodder commands that need to
resolve blob stores from CLI args (import, pull, repo-fsck, genesis).

## Key Types

- `EnvBlobStore`: Factory for creating blob-store-enabled command envs
- `BlobStore`: Helpers for resolving stores from blob-store-id args, config
  paths, or default-store fallback
- `Complete`: Tab-completion provider that lists configured blob stores

## Naming Note

The `_madder` suffix is a historical artifact: this package once held
dodder's in-process copy of the madder CLI (see #144). The facade is gone
but the dispatch helpers stayed because dodder commands genuinely need
them. Renaming the package is queued as a separate refactor.
