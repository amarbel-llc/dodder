# Design: `dodder check-workspace dirty`

## Goal

Fast local-only check: "has this workspace-repo changed since its last sync
with the parent?" Designed for shell prompt use — quiet output, exit codes only.

## Mechanism

At **clone/pull time**, two values are stored in the V1 workspace config:

- `SyncTai` — TAI of the workspace's latest inventory list at sync time
- `SyncDigest` — markl digest of that inventory list

At **check time**, load the workspace config, read the workspace's current
latest inventory list, compare:

1. Current TAI > stored SyncTai → dirty (exit 0)
2. TAIs match but digests differ → dirty (exit 0)
3. Both match → clean (exit 1)

> **Note:** Storing mutable sync state in the workspace config is a pragmatic
> shortcut. This data is not configuration — it may warrant a separate file in
> the future.

## CLI

```sh
dodder check-workspace dirty
```

- **Exit 0** — dirty (workspace has local changes since last sync)
- **Exit 1** — clean
- **Exit 2** — not in a workspace-repo (also prints error to stderr)
- Quiet by default (documented in help text)

## What Changes

| Area | Change |
|------|--------|
| `workspace_config_blobs` V1 | Add `SyncTai` and `SyncDigest` fields |
| `env_workspace` | Write sync baseline on clone/pull |
| New command `check-workspace` | Subcommand `dirty`, loads config + latest inventory list, compares, exits |
| `init_workspace.go` | Write baseline after initial clone in `runExperimentalRepo` |
| push/pull commands | Update baseline after successful sync |

## What Does NOT Change

- V0 workspace configs (lightweight workspaces)
- Transfer protocol
- Existing command behavior

## Rollback

Purely additive. New fields in V1 config are ignored by code that doesn't read
them. Removing the command is a single file deletion.
