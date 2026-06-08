# workspace_config_value_blobs

Tommy-codegen leaf package for workspace configuration value structs. Holds the
`//go:generate tommy generate` structs and their generated `*_tommy.go` so no
consumer of the generated `Decode*` API shares this package (tommy v0.4.3
codegen-isolation requirement).

## Key Types

- `V0`–`V3`: versioned workspace config blobs (V1 embeds V0, V2 embeds V1, V3
  embeds V2 — the chain lives together so embedding resolves intra-package)
- `HaustoriaConfig`, `CalDAVConfig`, `CalendarConfig`, `OrgmodeConfig`,
  `OrgmodeWebDAV`, `OrgmodeSFTP`, `FolderConfig`: haustoria sub-config structs
- `ResolvedOrgmodeConfig` / `ResolvedCalDAVConfig`: env-merged resolution
  results returned by `OrgmodeConfig.ResolveOrgmode()` / `CalDAVConfig.Resolve()`

## Consumer

`internal/echo/workspace_config_blobs` re-exports these via type aliases and
owns the `Coder` blob coder, the `Config`/`ConfigWith*` composite interfaces,
and the hand-coded `Temporary` config. Mirrors the
`charlie/repo_configs` <-> `delta/repo_configs` precedent.
