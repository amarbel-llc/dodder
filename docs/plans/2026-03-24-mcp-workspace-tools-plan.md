# MCP Workspace Tools Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add workspace-aware MCP tools (status, checkin, diff,
read-checked-out) to dodder's MCP server.

**Architecture:** A second bridge method `RunWorkspaceCommand` (without
`IgnoreWorkspace: true`) serves the four new tools. Existing tools keep the
repo-only bridge. All new tools use `makeBridgeHandler` with the workspace
bridge.

**Tech Stack:** Go, purse-first go-mcp library, dodder CLI bridge pattern.

**Rollback:** Purely additive -- revert the commit.

--------------------------------------------------------------------------------

### Task 1: Add RunWorkspaceCommand to Bridge

**Files:** - Modify: `go/internal/tango/mcp_dodder/bridge.go` - Modify:
`go/internal/tango/mcp_dodder/bridge_test.go`

**Step 1: Refactor RunCommand to use a shared helper**

Extract the body of `RunCommand` into a private `runCommand` method that takes
an `ignoreWorkspace bool` parameter. `RunCommand` calls it with `true`,
`RunWorkspaceCommand` calls it with `false`.

In `go/internal/tango/mcp_dodder/bridge.go`:

``` go
func (b Bridge) RunCommand(
    ctx context.Context,
    cmdName string,
    cliArgs []string,
    maxBytes int,
) (BridgeResult, error) {
    return b.runCommand(ctx, cmdName, cliArgs, maxBytes, true)
}

func (b Bridge) RunWorkspaceCommand(
    ctx context.Context,
    cmdName string,
    cliArgs []string,
    maxBytes int,
) (BridgeResult, error) {
    return b.runCommand(ctx, cmdName, cliArgs, maxBytes, false)
}

func (b Bridge) runCommand(
    ctx context.Context,
    cmdName string,
    cliArgs []string,
    maxBytes int,
    ignoreWorkspace bool,
) (BridgeResult, error) {
    outWriter := MakeLimitingWriter(maxBytes)
    errWriter := MakeLimitingWriter(maxBytes)

    config := repo_config_cli.Default()
    config.CustomOut = outWriter
    config.CustomErr = errWriter
    config.IgnoreWorkspace = ignoreWorkspace

    // ... rest unchanged from current RunCommand ...
}
```

**Step 2: Add test for RunWorkspaceCommand**

In `go/internal/tango/mcp_dodder/bridge_test.go`, add a test that verifies
`RunWorkspaceCommand` exists and returns an error for unknown commands (mirrors
the existing `TestBridgeUnknownCommand`):

``` go
func TestBridgeWorkspaceUnknownCommand(t *testing.T) {
    utility := command.MakeUtility("dodder", config_cli.Default())
    bridge := MakeBridge(utility)
    _, err := bridge.RunWorkspaceCommand(
        context.Background(),
        "nonexistent-command",
        nil,
        100_000,
    )
    if err == nil {
        t.Fatal("expected error for unknown command")
    }
}
```

**Step 3: Run tests**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore/go && go test -v -tags test,debug ./internal/tango/mcp_dodder/`

Expected: PASS

**Step 4: Commit**

    feat(mcp): add workspace-aware bridge method

    RunWorkspaceCommand delegates to CLI commands without setting
    IgnoreWorkspace, allowing workspace-dependent tools (status, checkin,
    diff) to access checked-out objects.

--------------------------------------------------------------------------------

### Task 2: Register dodder_status tool

**Files:** - Modify: `go/internal/tango/mcp_dodder/server.go`

**Step 1: Add makeWorkspaceBridgeHandler helper**

Below the existing `makeBridgeHandler` function (line \~666), add a workspace
variant that calls `bridge.RunWorkspaceCommand` instead of `bridge.RunCommand`.
The logic is identical to `makeBridgeHandler` except for the bridge method
called:

``` go
func makeWorkspaceBridgeHandler(
    bridge Bridge,
    cmdName string,
    translate paramTranslator,
) server.ToolHandlerV1 {
    return func(
        ctx context.Context,
        args json.RawMessage,
    ) (*protocol.ToolCallResultV1, error) {
        var cliArgs []string

        if translate != nil {
            var err error
            if cliArgs, err = translate(args); err != nil {
                return protocol.ErrorResultV1(
                    fmt.Sprintf("Invalid arguments: %v", err),
                ), nil
            }
        }

        result, err := bridge.RunWorkspaceCommand(ctx, cmdName, cliArgs, defaultMaxBytes)
        if err != nil {
            errMsg := formatErrorDetail(err)
            if result.Stderr != "" {
                errMsg += "\n\nstderr:\n" + result.Stderr
            }
            return protocol.ErrorResultV1(errMsg), nil
        }

        output := result.Stdout
        if result.Truncated {
            output += fmt.Sprintf(
                "\n\n[truncated: showed %d of %d bytes]",
                len(result.Stdout),
                result.BytesSeen,
            )
        }

        if result.Stderr != "" {
            output += "\n\nstderr:\n" + result.Stderr
        }

        return &protocol.ToolCallResultV1{
            Content: []protocol.ContentBlockV1{
                protocol.TextContentV1(output),
            },
        }, nil
    }
}
```

**Step 2: Register dodder_status**

Add after the `dodder_edit` registration (line \~544) in `registerTools`:

``` go
tools.Register(
    protocol.ToolV1{
        Name:        "dodder_status",
        Description: "List checked-out objects in the workspace with their state (CheckedOut, Recognized, Untracked, Conflicted). Requires an active workspace. Returns box format with state headers.",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": {
                    "type": "array",
                    "items": {"type": "string"},
                    "description": "Optional query terms to filter objects. Defaults to all checked-out objects."
                }
            },
            "additionalProperties": false
        }`),
        Annotations: readOnlyAnnotations,
    },
    makeWorkspaceBridgeHandler(bridge, "status", func(args json.RawMessage) ([]string, error) {
        var p struct {
            Query []string `json:"query"`
        }
        if err := json.Unmarshal(args, &p); err != nil {
            return nil, err
        }
        return p.Query, nil
    }),
)
```

**Step 3: Verify compilation**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore/go && go build -tags debug ./internal/tango/mcp_dodder/`

Expected: compiles without errors.

**Step 4: Commit**

    feat(mcp): add dodder_status tool

    Exposes workspace status (checked-out object state) via MCP using
    the workspace-aware bridge.

--------------------------------------------------------------------------------

### Task 3: Register dodder_checkin tool

**Files:** - Modify: `go/internal/tango/mcp_dodder/server.go`

**Step 1: Register dodder_checkin**

Add after the `dodder_status` registration:

``` go
tools.Register(
    protocol.ToolV1{
        Name:        "dodder_checkin",
        Description: "Commit working copy changes to the store. Checks in objects matching the query from the workspace. Requires an active workspace.",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": {
                    "type": "array",
                    "items": {"type": "string"},
                    "description": "Query terms selecting which objects to check in (e.g. [':z'] for all zettels, ['ceroplastes/midtown'] for a specific object)"
                }
            },
            "required": ["query"],
            "additionalProperties": false
        }`),
        Annotations: writeAnnotations,
    },
    makeWorkspaceBridgeHandler(bridge, "checkin", func(args json.RawMessage) ([]string, error) {
        var p struct {
            Query []string `json:"query"`
        }
        if err := json.Unmarshal(args, &p); err != nil {
            return nil, err
        }
        return p.Query, nil
    }),
)
```

**Step 2: Verify compilation**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore/go && go build -tags debug ./internal/tango/mcp_dodder/`

Expected: compiles without errors.

**Step 3: Commit**

    feat(mcp): add dodder_checkin tool

    Exposes workspace checkin via MCP, committing working copy changes
    back to the store.

--------------------------------------------------------------------------------

### Task 4: Register dodder_diff tool

**Files:** - Modify: `go/internal/tango/mcp_dodder/server.go`

**Step 1: Register dodder_diff**

Add after the `dodder_checkin` registration:

``` go
tools.Register(
    protocol.ToolV1{
        Name:        "dodder_diff",
        Description: "Show differences between internal (store) and external (working copy) versions of checked-out objects. Requires an active workspace.",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": {
                    "type": "array",
                    "items": {"type": "string"},
                    "description": "Optional query terms to filter objects. Defaults to all checked-out objects."
                }
            },
            "additionalProperties": false
        }`),
        Annotations: readOnlyAnnotations,
    },
    makeWorkspaceBridgeHandler(bridge, "diff", func(args json.RawMessage) ([]string, error) {
        var p struct {
            Query []string `json:"query"`
        }
        if err := json.Unmarshal(args, &p); err != nil {
            return nil, err
        }
        return p.Query, nil
    }),
)
```

**Step 2: Verify compilation**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore/go && go build -tags debug ./internal/tango/mcp_dodder/`

Expected: compiles without errors.

**Step 3: Commit**

    feat(mcp): add dodder_diff tool

    Exposes workspace diff via MCP, showing internal vs external
    differences for checked-out objects.

--------------------------------------------------------------------------------

### Task 5: Register dodder_read_checked_out tool

**Files:** - Modify: `go/internal/tango/mcp_dodder/server.go`

**Step 1: Register dodder_read_checked_out**

This bridges to `format-blob` using the workspace-aware bridge. The `.` sigil
prefix on the object ID tells dodder to use the external (working copy) version.

Add after the `dodder_diff` registration:

``` go
tools.Register(
    protocol.ToolV1{
        Name:        "dodder_read_checked_out",
        Description: "Read the working copy file content of a checked-out object. Returns the external (filesystem) version, not the store version. Requires an active workspace.",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "object_id": {
                    "type": "string",
                    "description": "Object identifier (e.g. 'ceroplastes/midtown')"
                },
                "format_id": {
                    "type": "string",
                    "description": "Formatter ID to use (optional, uses type default if omitted)"
                }
            },
            "required": ["object_id"],
            "additionalProperties": false
        }`),
        Annotations: readOnlyAnnotations,
    },
    makeWorkspaceBridgeHandler(bridge, "format-blob", func(args json.RawMessage) ([]string, error) {
        var p struct {
            ObjectId string `json:"object_id"`
            FormatId string `json:"format_id"`
        }
        if err := json.Unmarshal(args, &p); err != nil {
            return nil, err
        }
        // Prefix with . sigil to read external (working copy) version
        cliArgs := []string{"." + p.ObjectId}
        if p.FormatId != "" {
            cliArgs = append(cliArgs, p.FormatId)
        }
        return cliArgs, nil
    }),
)
```

**Step 2: Verify compilation**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore/go && go build -tags debug ./internal/tango/mcp_dodder/`

Expected: compiles without errors.

**Step 3: Commit**

    feat(mcp): add dodder_read_checked_out tool

    Reads working copy blob content via format-blob with external sigil,
    allowing inspection of checked-out file content.

--------------------------------------------------------------------------------

### Task 6: Update MCP instructions

**Files:** - Modify: `go/internal/tango/mcp_dodder/server.go`

**Step 1: Add workspace section to mcpInstructions**

Insert before the closing backtick of `mcpInstructions` (before line 140), after
the "Resource Drill-Down" section:

    ## Workspace Tools

    When dodder runs inside a workspace (directory with .dodder-workspace config),
    these tools operate on checked-out objects:

    - dodder_status — list checked-out objects with state (Recognized = modified,
      Untracked = new, Conflicted = merge conflict)
    - dodder_diff — show internal vs external differences
    - dodder_read_checked_out — read working copy file content
    - dodder_checkin — commit working copy changes to the store

    ### Workspace Workflow

    Inspect what changed:
      → dodder_status() → see all checked-out objects and their state
      → dodder_diff() → see what changed in modified objects

    Read working copy content:
      → dodder_read_checked_out(object_id) → get current file content

    Commit changes:
      → dodder_checkin(query) → commit matching objects to the store

**Step 2: Verify compilation**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore/go && go build -tags debug ./internal/tango/mcp_dodder/`

Expected: compiles without errors.

**Step 3: Commit**

    docs(mcp): add workspace tools to MCP instructions

    Documents status, diff, read_checked_out, and checkin workflows
    in the embedded MCP server instructions.

--------------------------------------------------------------------------------

### Task 7: Build and integration test

**Step 1: Build**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore && just build`

Expected: builds successfully.

**Step 2: Run unit tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore && just test-go`

Expected: PASS

**Step 3: Run integration tests**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/snug-sycamore && just test-bats`

Expected: PASS. No existing tests should break since this is purely additive.
