# Madder MCP Server via go-mcp

## Goal

Add an `mcp` subcommand to madder that starts an MCP server (stdio transport)
exposing madder's blob store commands as MCP tools. Uses go-mcp's raw server
layer (not `command.App`) with a synthetic Request bridge that calls into
existing madder commands unchanged.

This is the first step toward MCP support for dodder. Madder's simpler command
set (14 commands) lets us validate the synthetic Request pattern before
migrating dodder's 47+ commands.

## Phasing

1. **This design:** madder blob store tools via MCP
2. **Next pass:** dodder read-only tools (show, status, info, cat, list)
3. **Future:** dodder read-write tools (checkin, checkout, new, edit)
4. **Future:** full dodder command set

## Architecture

```
                    +---------------------+
                    |  madder mcp         |  (new subcommand)
                    |  starts stdio server|
                    +--------+------------+
                             |
                    +--------v------------+
                    |  go-mcp             |
                    |  server.New()       |
                    |  transport.Stdio    |
                    |  ToolRegistry       |
                    +--------+------------+
                             | CallTool(name, args)
                    +--------v------------+
                    |  lima/mcp_madder    |  (new package)
                    |  - JSON -> CLI args |
                    |  - synthetic Request|
                    |  - output capture   |
                    +--------+------------+
                             | cmd.Run(req)
                    +--------v------------+
                    |  existing madder    |
                    |  commands (lima/)   |
                    |  unchanged          |
                    +---------------------+
```

### New Code

- `lima/mcp_madder/` -- tool registration, arg translation, bridge
- `lima/commands_madder/mcp.go` -- the `mcp` subcommand

### Modified Code (small, backward-compatible)

- `bravo/ui/printer.go` -- `io.Writer` support in printer
- `foxtrot/fd/std.go` -- `MakeStdFromWriter` constructor
- `golf/env_ui/` -- custom writer support via config
- `echo/config_cli/` -- `CustomOut`/`CustomErr` fields on Config
- `lima/commands_madder/info_repo.go` -- one-line fix: `GetFile()` to
  `GetUIFile()`

### New Dependency

- `github.com/amarbel-llc/go-lib-mcp` in go.mod

## Synthetic Request Bridge

MCP tool handlers translate JSON params to CLI args, then use existing Utility
methods to build and execute a Request:

```
MCP tool call (name="madder_cat", args={"sha": "abc123"})
    |
    v
translate JSON params to CLI args: ["madder", "cat", "abc123"]
    |
    v
utility.MakeCmdAndFlagSet(ctx, args)  ->  cmd + flagSet
    |
    v
utility.MakeRequest(ctx, cmd, flagSet)  ->  Request
    |
    v
cmd.Run(req)  ->  output in LimitingWriter, errors in ctx
    |
    v
writer contents -> protocol.TextContent in ToolCallResult
ctx.Cause()    -> protocol.ErrorResult if non-nil
```

### Bridge Function Shape

```go
func (b *Bridge) RunCommand(
    ctx context.Context,
    cmdName string,
    cliArgs []string,
    limits output.TextLimits,
) (result string, truncInfo *output.TruncationInfo, err error) {
    errCtx := errors.MakeContext(ctx)
    args := append([]string{"madder", cmdName}, cliArgs...)

    cmd, flagSet, ok := b.utility.MakeCmdAndFlagSet(errCtx, args)
    if !ok {
        return "", nil, errCtx.Cause()
    }

    req, ok := b.utility.MakeRequest(errCtx, cmd, flagSet)
    if !ok {
        return "", nil, errCtx.Cause()
    }

    cmd.Run(req)
    return outWriter.String(), outWriter.TruncationInfo(), errCtx.Cause()
}
```

### Threading Writers via Config

Custom output writers flow through `config_cli.Config` fields:

1. MCP handler creates a `LimitingWriter` for stdout and stderr
2. Sets them on config: `config.CustomOut = limitingWriter`
3. Builds utility with this config
4. `MakeEnvBlobStore(req)` calls `env_ui.Make()` which checks config for custom
   writers and uses `fd.MakeStdFromWriter()` instead of `fd.MakeStd(os.Stdout)`

## Output Capture

### Problem

`fd.MakeStd` and `ui.MakePrinter` require `*os.File`. The `*os.File`
requirement exists for TTY detection. In MCP mode, output is never a TTY.

### Solution

Add `io.Writer`-based constructors:

**`bravo/ui/printer.go`:** Refactor `printer` to store `io.Writer` instead of
`*os.File`. Add `MakePrinterFromWriter(w io.Writer)` with `isTty: false`.
Existing `MakePrinter(*os.File)` wraps the file and sets `isTty` from
`primordial.IsTty`.

**`foxtrot/fd/std.go`:** Add `MakeStdFromWriter(w io.Writer)`.

**`golf/env_ui/main.go`:** In `Make()`, check config for `CustomOut`/`CustomErr`
and use writer-based constructors when set.

**`echo/config_cli/main.go`:** Add fields:

```go
type Config struct {
    // ... existing fields ...
    CustomOut io.Writer `toml:"-"`
    CustomErr io.Writer `toml:"-"`
}
```

**`lima/commands_madder/info_repo.go:107`:** Change `env.GetUI().GetFile()` to
`env.GetUIFile()`. This is the only `GetFile()` call in madder commands. The
receiving function (`PrintBlobStoreConfig`) already accepts `io.Writer`.

### Validation

Grep confirms only one `GetFile()` call in madder command code
(`info_repo.go:107`). All other output uses `GetUI().Print()`,
`GetUIFile()` (returns `WriterAndStringWriter`), or `io.Copy` to
`GetUIFile()` -- all compatible with `io.Writer`.

## Limiting Writer

MCP protocol (both V0 and V1) requires complete responses in a single JSON-RPC
message. No streaming. Every tool response must be bounded.

### Design

A `LimitingWriter` wraps a `bytes.Buffer` with write-time limits:

- Accepts writes up to `MaxBytes` (default 100KB from
  `output.StandardDefaults()`)
- After hitting the limit, silently discards further writes (returns `len(p),
  nil` so commands don't error)
- Tracks truncation metadata (original bytes seen, kept bytes, position)
- For tail mode, uses a ring buffer internally

### Application

Every tool uses a `LimitingWriter` uniformly. Tools that produce potentially
large output (`cat`, `cat-ids`, `fsck`) expose `head`, `tail`, `max_bytes`
params so the caller can override defaults. Small-output tools use defaults with
negligible overhead.

When output was truncated, the MCP response includes truncation metadata so the
LLM knows it's seeing partial results and can request a different range.

## Tool Definitions

| Madder Command            | MCP Tool Name                    | Key Parameters                                                         |
|---------------------------|----------------------------------|------------------------------------------------------------------------|
| `cat`                     | `madder_cat`                     | `sha` (req), `blob_store_index`, `utility`, `prefix_sha`, `head/tail` |
| `cat-ids`                 | `madder_cat_ids`                 | `blob_store_index`, `offset/limit`                                     |
| `list`                    | `madder_list`                    | (none)                                                                 |
| `read`                    | `madder_read`                    | `input` (JSON string, piped to stdin)                                  |
| `write`                   | `madder_write`                   | `paths` (array), `check` (bool)                                        |
| `info-repo`               | `madder_info_repo`               | `blob_store_index`, `keys` (array)                                     |
| `fsck`                    | `madder_fsck`                    | `blob_store_index`, `head/tail`                                        |
| `sync`                    | `madder_sync`                    | `source`, `destinations` (array), `limit` (int)                        |
| `init`                    | `madder_init`                    | (none)                                                                 |
| `init-from`               | `madder_init_from`               | `config_path`                                                          |
| `init-inventory-archive`  | `madder_init_inventory_archive`  | (none)                                                                 |
| `init-pointer`            | `madder_init_pointer`            | (none)                                                                 |
| `pack`                    | `madder_pack`                    | `delete_loose` (bool)                                                  |

### Naming Convention

`madder_<command>` with hyphens converted to underscores. When dodder tools are
added later, they'll use `dodder_<command>`.

### JSON Schema

Hand-written in tool registration code. Each handler translates its JSON params
to CLI args. Example: `madder_cat` with `{"sha": "abc123", "prefix_sha": true}`
becomes `["madder", "cat", "-prefix-sha", "abc123"]`.

## Error Translation

```
cmd.Run(req) completes
    |
    +-- ctx.Cause() == nil
    |   -> ToolCallResult{Content: TextContent(output)}
    |
    +-- ctx.Cause() != nil
        |
        +-- errors.Is400BadRequest(err)
        |   -> ErrorResult("Bad request: " + err.Error())
        |
        +-- errors.IsErrNotFound(err)
        |   -> ErrorResult("Not found: " + err.Error())
        |
        +-- other
            -> ErrorResult(err.Error())
```

All errors become `ToolCallResult` with `isError=true` (not JSON-RPC errors).
JSON-RPC errors are reserved for protocol-level problems (tool not found,
malformed request).

## Risks and Mitigations

### Risk: Adapter friction with synthetic Requests

Some commands may not work cleanly with the synthetic Request pattern
(unexpected flag interactions, environment setup order, etc.).

**Mitigation:** Start with the simplest commands (`list`, `info-repo`) and work
up to complex ones (`cat`, `sync`). If adapter issues are severe, pivot to
direct function calls (extract core logic from commands into shared functions).

### Risk: `interfaces.Printer.GetFile()` returns nil for writer-backed printers

Code that calls `GetFile()` on a writer-backed printer gets nil instead of
`*os.File`.

**Mitigation:** Only one `GetFile()` call exists in madder commands
(`info_repo.go:107`), and it's being changed. The `interfaces.Printer` interface
retains `GetFile()` for backward compatibility -- MCP code paths won't call it.

### Risk: Global state in ui package

`ui.SetVerbose()` and `ui.SetOutput()` are global. Concurrent MCP tool calls
could interfere.

**Mitigation:** MCP tool calls should be serialized initially. Concurrent
execution requires refactoring the `ui` package to be instance-based rather than
global. This is a known debt, not a blocker for the initial implementation.

## Decisions Made

- **Don't extract alfa/errors into go-mcp yet.** Keep in dodder. Revisit after
  learning from the madder migration.
- **Extend cmd/madder with `mcp` subcommand** rather than creating a separate
  binary. Single binary, dual mode.
- **Synthetic Request bridge** rather than per-command wrappers or subprocess
  execution. Tests the pattern dodder will need.
- **Thread writers via Config fields** rather than context values or separate
  constructors. Explicit, type-safe.
- **Uniform LimitingWriter** for all tools. No special-casing small vs large
  output.
