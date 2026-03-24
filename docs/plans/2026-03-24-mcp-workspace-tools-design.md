# MCP Workspace Tools Design

## Problem

The dodder MCP server exposes store-level operations (query, show, edit, new,
format-blob, type/tag discovery) but has no workspace awareness. Checked-out
objects, their state (recognized/conflicted/untracked), diffs, and checkin are
not available via MCP. This means an LLM cannot inspect or commit working copy
changes.

## Approach

Add four workspace-aware tools to the MCP server using a second bridge mode that
does not set `IgnoreWorkspace: true`. The existing bridge remains unchanged for
current tools. See [#55](https://github.com/amarbel-llc/dodder/issues/55) for
the underlying query executor bug that motivated the original workaround.

## Tools

### dodder_status (read-only)

List checked-out objects with their state (CheckedOut, Recognized, Untracked,
Conflicted). Bridges to CLI `status` command.

Parameters: - `query` (string array, optional) -- filter objects. Defaults to
all.

Output: box format with state headers (matching CLI output). JSON format
deferred to a follow-up.

### dodder_checkin (write)

Commit working copy changes to the store. Bridges to CLI `checkin` command.
Basic mode only -- no organize, no proto defaults, no open-blob.

Parameters: - `query` (string array, required) -- which objects to check in.

Output: bridge stdout showing committed objects.

### dodder_diff (read-only)

Show differences between internal (store) and external (working copy) versions.
Bridges to CLI `diff` command.

Parameters: - `query` (string array, optional) -- filter objects. Defaults to
all.

Output: text diff output from the CLI.

### dodder_read_checked_out (read-only)

Read the working copy file content of a checked-out object. Bridges to CLI
`format-blob` with the external sigil (`.`).

Parameters: - `object_id` (string, required) -- which object's working copy to
read. - `format_id` (string, optional) -- formatter to use.

Output: formatted blob content from the working copy.

## Implementation

### Bridge changes (bridge.go)

Add `RunWorkspaceCommand` -- identical to `RunCommand` but without
`config.IgnoreWorkspace = true`. Existing `RunCommand` unchanged.

### Tool registration (server.go)

Register the four new tools in `registerTools`, using `RunWorkspaceCommand` for
all of them. Update `mcpInstructions` to document workspace workflows.

### Files modified

1.  `go/internal/tango/mcp_dodder/bridge.go` -- add `RunWorkspaceCommand`
2.  `go/internal/tango/mcp_dodder/server.go` -- register 4 tools, update
    instructions
