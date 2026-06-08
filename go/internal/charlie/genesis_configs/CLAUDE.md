# genesis_configs

Genesis configuration consumer package for repository initialization. Owns the
blob coders and the mutable interface; the value structs and base interfaces
live in `internal/bravo/genesis_config_blobs` and are re-exported here as
aliases (tommy v0.4.3 codegen-isolation split).

## Owned Here

- `ConfigPrivateMutable`: mutable private config (adds setters +
  `CommandComponentWriter`)
- `TypedConfigPublic` / `TypedConfigPrivate` / `TypedConfigPrivateMutable`:
  typed-blob aliases
- `CoderPrivate` / `CoderPublic`: hyphence blob coders
- `Default()` / `DefaultWithVersion()`: genesis config constructors

## Re-exported from `bravo/genesis_config_blobs`

- `Config` / `ConfigPublic` / `ConfigPrivate` (base interfaces)
- `TomlV2Common` / `TomlV2Private` / `TomlV2Public` (value structs)
