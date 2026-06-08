# workspace_config_blobs

Workspace configuration consumer package. Owns the blob coder, the composite
interfaces, and the hand-coded temporary config; the versioned value structs
live in `internal/delta/workspace_config_value_blobs` and are re-exported here
as aliases (tommy v0.4.3 codegen-isolation split).

## Owned Here

- `Config`: base workspace configuration interface
- `ConfigWithRepo` / `ConfigWithDefaultQueryString` / `ConfigWithDryRun` /
  `ConfigWithParentPath` / `ConfigWithSyncBaseline` / `ConfigWithHaustoria` /
  `ConfigWithIgnore`: capability interfaces
- `ConfigTemporary` + `Temporary`: temporary (unpersisted) workspace config
- `TypedConfig`: triple-hyphen typed blob
- `Coder`: hyphence blob coder

## Re-exported from `delta/workspace_config_value_blobs`

- `V0`–`V3` (versioned value structs) plus the haustoria sub-config structs
