# Dodder MCP Server Design

## Problem

Dodder exposes zettelkasten operations only through CLI. LLM agents need
programmatic access to query and view zettels. The madder MCP server covers blob
store operations but not zettelkasten-level concepts (zettels, tags, types,
queries).

## Approach

Mirror the madder MCP server pattern: new `mcp_dodder` package in
`internal/hotel/` with the same Bridge + LimitingWriter architecture. Separate
`dodder mcp` command, independent from `blob_store-mcp`.

## Tools (read-only initial set)

| Tool | CLI mapping | Key params |
|------|-------------|-----------|
| `dodder_show` | `dodder show <object_id>` | `object_id` (required), `format` (optional, default "log") |
| `dodder_query` | `dodder show <query...>` | `query` (required string array), `format` (optional, default "log") |
| `dodder_format_blob` | `dodder format-blob [format_id] <object_id>` | `object_id` (required), `format_id` (optional) |

All tools use `readOnlyAnnotations` (read-only + idempotent).

## New Files

- `go/internal/hotel/mcp_dodder/limiting_writer.go` — output capping (copy from madder)
- `go/internal/hotel/mcp_dodder/bridge.go` — command execution bridge
- `go/internal/hotel/mcp_dodder/server.go` — tool registration + server startup
- `go/internal/victor/commands_dodder/mcp.go` — `dodder mcp` command
- `go/internal/victor/commands_dodder/install_mcp.go` — `dodder install-mcp` command

## Modified Files

- `zz-tests_bats/current_version/complete.bats` — add `mcp` and `install-mcp`

## Not Changing

- `mcp_madder/` stays untouched
- `commands_dodder/main.go` unchanged (new commands register via `init()`)
