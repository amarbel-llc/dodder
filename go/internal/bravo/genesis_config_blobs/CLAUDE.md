# genesis_config_blobs

Tommy-codegen leaf package for genesis configuration value structs. Holds the
`//go:generate tommy generate` structs and their generated `*_tommy.go` so no
consumer of the generated `Decode*` API shares this package (tommy v0.4.3
codegen-isolation requirement).

## Key Types

- `TomlV2Common`: shared genesis fields (store version, repo id, type ids)
- `TomlV2Private`: private genesis config (carries the private key)
- `TomlV2Public`: public genesis config (carries the public key)
- `Config` / `ConfigPublic` / `ConfigPrivate`: base interfaces the structs
  implement; defined here because `GetGenesisConfig*` methods return them

## Consumer

`internal/charlie/genesis_configs` re-exports these via type aliases and owns
the `CoderPrivate`/`CoderPublic` blob coders plus the mutable
`ConfigPrivateMutable` interface. Mirrors the
`alfa/repo_blobs` <-> `charlie/repo_blobs` precedent.
