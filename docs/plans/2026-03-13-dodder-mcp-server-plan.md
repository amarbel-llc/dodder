# Dodder MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add a stdio MCP server to dodder exposing read-only zettelkasten operations (show, query, format-blob).

**Architecture:** New `mcp_dodder` package in `internal/hotel/` mirroring `mcp_madder/`. Same Bridge + LimitingWriter pattern. Three read-only tools. New `dodder mcp` and `dodder install-mcp` commands.

**Tech Stack:** Go, `go-mcp` library (`github.com/amarbel-llc/purse-first/libs/go-mcp`), JSON-RPC 2.0 over stdio.

**Rollback:** Purely additive — revert the commit to remove.

---

### Task 1: Create mcp_dodder package with LimitingWriter

**Files:**
- Create: `go/internal/hotel/mcp_dodder/limiting_writer.go`
- Create: `go/internal/hotel/mcp_dodder/limiting_writer_test.go`

**Step 1: Create limiting_writer.go**

Copy from `go/internal/hotel/mcp_madder/limiting_writer.go` with package name changed to `mcp_dodder`.

```go
package mcp_dodder

import "bytes"

type LimitingWriter struct {
	buf       bytes.Buffer
	maxBytes  int
	bytesSeen int
}

func MakeLimitingWriter(maxBytes int) *LimitingWriter {
	return &LimitingWriter{maxBytes: maxBytes}
}

func (w *LimitingWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.bytesSeen += n

	remaining := w.maxBytes - w.buf.Len()
	if remaining <= 0 {
		return n, nil
	}

	if len(p) > remaining {
		p = p[:remaining]
	}

	w.buf.Write(p)
	return n, nil
}

func (w *LimitingWriter) WriteString(s string) (n int, err error) {
	return w.Write([]byte(s))
}

func (w *LimitingWriter) String() string {
	return w.buf.String()
}

func (w *LimitingWriter) Truncated() bool {
	return w.bytesSeen > w.maxBytes
}

func (w *LimitingWriter) BytesSeen() int {
	return w.bytesSeen
}

func (w *LimitingWriter) BytesKept() int {
	return w.buf.Len()
}

func (w *LimitingWriter) Reset() {
	w.buf.Reset()
	w.bytesSeen = 0
}
```

**Step 2: Create limiting_writer_test.go**

Copy from `go/internal/hotel/mcp_madder/limiting_writer_test.go` with package name changed to `mcp_dodder`.

```go
package mcp_dodder

import (
	"strings"
	"testing"
)

func TestLimitingWriterUnderLimit(t *testing.T) {
	w := MakeLimitingWriter(100)
	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("expected n=5, got %d", n)
	}
	if w.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", w.String())
	}
	if w.Truncated() {
		t.Fatal("should not be truncated")
	}
}

func TestLimitingWriterOverLimit(t *testing.T) {
	w := MakeLimitingWriter(10)
	data := strings.Repeat("x", 20)
	n, err := w.Write([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if n != 20 {
		t.Fatalf("expected n=20, got %d", n)
	}
	if len(w.String()) > 10 {
		t.Fatalf("buffer should be at most 10 bytes, got %d", len(w.String()))
	}
	if !w.Truncated() {
		t.Fatal("should be truncated")
	}
	if w.BytesSeen() != 20 {
		t.Fatalf("expected 20 bytes seen, got %d", w.BytesSeen())
	}
}

func TestLimitingWriterMultipleWrites(t *testing.T) {
	w := MakeLimitingWriter(10)
	w.Write([]byte("12345"))
	w.Write([]byte("67890"))
	w.Write([]byte("overflow"))
	if w.String() != "1234567890" {
		t.Fatalf("expected '1234567890', got %q", w.String())
	}
	if !w.Truncated() {
		t.Fatal("should be truncated")
	}
	if w.BytesSeen() != 18 {
		t.Fatalf("expected 18 bytes seen, got %d", w.BytesSeen())
	}
}

func TestLimitingWriterStringWriter(t *testing.T) {
	w := MakeLimitingWriter(100)
	n, err := w.WriteString("hello")
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("expected n=5, got %d", n)
	}
	if w.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", w.String())
	}
}
```

**Step 3: Run tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab/go && go test -v -tags test,debug ./internal/hotel/mcp_dodder/`
Expected: All 4 tests PASS

**Step 4: Commit**

```bash
git add go/internal/hotel/mcp_dodder/limiting_writer.go go/internal/hotel/mcp_dodder/limiting_writer_test.go
git commit -m "feat(mcp_dodder): add LimitingWriter for output capping"
```

---

### Task 2: Create mcp_dodder bridge

**Files:**
- Create: `go/internal/hotel/mcp_dodder/bridge.go`
- Create: `go/internal/hotel/mcp_dodder/bridge_test.go`

**Step 1: Create bridge.go**

Mirror `go/internal/hotel/mcp_madder/bridge.go` but use `"dodder"` as the utility name.

```go
package mcp_dodder

import (
	"context"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/foxtrot/config_cli"
)

type BridgeResult struct {
	Stdout    string
	Stderr    string
	Truncated bool
	BytesSeen int
}

type Bridge struct {
	utility command.Utility
}

func MakeBridge(utility command.Utility) Bridge {
	return Bridge{
		utility: utility,
	}
}

func (b Bridge) RunCommand(
	ctx context.Context,
	cmdName string,
	cliArgs []string,
	maxBytes int,
) (BridgeResult, error) {
	outWriter := MakeLimitingWriter(maxBytes)
	errWriter := MakeLimitingWriter(maxBytes)

	config := &config_cli.Config{
		CustomOut: outWriter,
		CustomErr: errWriter,
	}

	utility := command.MakeUtility("dodder", config)

	for name, cmd := range b.utility.AllCmds() {
		utility.AddCmd(name, cmd)
	}

	errCtx := errors.MakeContext(ctx)

	args := make([]string, 0, 2+len(cliArgs))
	args = append(args, "dodder", cmdName)
	args = append(args, cliArgs...)

	var result BridgeResult

	if err := errCtx.Run(func(ctx errors.Context) {
		cmd, flagSet, ok := utility.MakeCmdAndFlagSet(ctx, args)
		if !ok {
			return
		}

		req, ok := utility.MakeRequest(ctx, cmd, flagSet)
		if !ok {
			return
		}

		cmd.Run(req)
	}); err != nil {
		return result, err
	}

	result = BridgeResult{
		Stdout:    outWriter.String(),
		Stderr:    errWriter.String(),
		Truncated: outWriter.Truncated(),
		BytesSeen: outWriter.BytesSeen(),
	}

	return result, nil
}
```

**Step 2: Create bridge_test.go**

```go
package mcp_dodder

import (
	"context"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/foxtrot/config_cli"
)

func TestBridgeUnknownCommand(t *testing.T) {
	utility := command.MakeUtility("dodder", config_cli.Default())
	bridge := MakeBridge(utility)
	_, err := bridge.RunCommand(
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

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab/go && go test -v -tags test,debug ./internal/hotel/mcp_dodder/`
Expected: All 5 tests PASS

**Step 4: Commit**

```bash
git add go/internal/hotel/mcp_dodder/bridge.go go/internal/hotel/mcp_dodder/bridge_test.go
git commit -m "feat(mcp_dodder): add command execution bridge"
```

---

### Task 3: Create mcp_dodder server with tool registration

**Files:**
- Create: `go/internal/hotel/mcp_dodder/server.go`

**Step 1: Create server.go**

Register three read-only tools: `dodder_show`, `dodder_query`, `dodder_format_blob`.

```go
package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/transport"
)

const defaultMaxBytes = 100_000

var readOnlyAnnotations = &protocol.ToolAnnotations{
	ReadOnlyHint:   protocol.BoolPtr(true),
	IdempotentHint: protocol.BoolPtr(true),
}

func RunServer(utility command.Utility) error {
	bridge := MakeBridge(utility)
	tools := server.NewToolRegistryV1()

	registerTools(tools, bridge)

	t := transport.NewStdio(os.Stdin, os.Stdout)
	srv, err := server.New(t, server.Options{
		ServerName:    "dodder",
		ServerVersion: "0.1.0",
		Tools:         tools,
	})
	if err != nil {
		return err
	}

	return srv.Run(context.Background())
}

func registerTools(tools *server.ToolRegistryV1, bridge Bridge) {
	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_show",
			Description: "View a specific dodder object by ID. Returns metadata and content for zettels, tags, or types.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier (e.g. zettel ID like 'ceroplastes/midtown', tag like '%tag', or type like '!type')"
					},
					"format": {
						"type": "string",
						"description": "Output format (log, text, json, organize). Defaults to log.",
						"enum": ["log", "text", "json", "organize"]
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeBridgeHandler(bridge, "show", func(args json.RawMessage) ([]string, error) {
			var p struct {
				ObjectId string `json:"object_id"`
				Format   string `json:"format"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.Format != "" {
				cliArgs = append(cliArgs, "-format", p.Format)
			}
			cliArgs = append(cliArgs, p.ObjectId)
			return cliArgs, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_query",
			Description: "Search for dodder objects matching a query expression. Query terms are combined with AND. Examples: ':z' (all zettels), ':t' (all tags), '%todo' (tagged with todo), '!article' (type article).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Query terms (e.g. [':z', '%todo'] for zettels tagged todo)"
					},
					"format": {
						"type": "string",
						"description": "Output format (log, text, json, organize). Defaults to log.",
						"enum": ["log", "text", "json", "organize"]
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeBridgeHandler(bridge, "show", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Query  []string `json:"query"`
				Format string   `json:"format"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.Format != "" {
				cliArgs = append(cliArgs, "-format", p.Format)
			}
			cliArgs = append(cliArgs, p.Query...)
			return cliArgs, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_format_blob",
			Description: "Format and display the blob content of a dodder zettel. Renders the zettel's associated file content using the type's configured formatter.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier for the zettel whose blob to format"
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
		makeBridgeHandler(bridge, "format-blob", func(args json.RawMessage) ([]string, error) {
			var p struct {
				ObjectId string `json:"object_id"`
				FormatId string `json:"format_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.FormatId != "" {
				cliArgs = append(cliArgs, p.FormatId)
			}
			cliArgs = append(cliArgs, p.ObjectId)
			return cliArgs, nil
		}),
	)
}

type paramTranslator func(args json.RawMessage) ([]string, error)

func makeBridgeHandler(
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

		result, err := bridge.RunCommand(ctx, cmdName, cliArgs, defaultMaxBytes)
		if err != nil {
			return protocol.ErrorResultV1(err.Error()), nil
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

**Step 2: Verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab/go && go build ./internal/hotel/mcp_dodder/`
Expected: No errors

**Step 3: Run all mcp_dodder tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab/go && go test -v -tags test,debug ./internal/hotel/mcp_dodder/`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add go/internal/hotel/mcp_dodder/server.go
git commit -m "feat(mcp_dodder): register show, query, and format-blob tools"
```

---

### Task 4: Add dodder mcp and install-mcp commands

**Files:**
- Create: `go/internal/victor/commands_dodder/mcp.go`
- Create: `go/internal/victor/commands_dodder/install_mcp.go`

**Step 1: Create mcp.go**

```go
package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/mcp_dodder"
)

func init() {
	utility.AddCmd("mcp", &Mcp{})
}

type Mcp struct{}

func (cmd Mcp) Run(req command.Request) {
	if err := mcp_dodder.RunServer(req.Utility); err != nil {
		req.Cancel(err)
	}
}
```

**Step 2: Create install_mcp.go**

```go
package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	gomcp_command "github.com/amarbel-llc/purse-first/libs/go-mcp/command"
)

func init() {
	utility.AddCmd("install-mcp", &InstallMcp{})
}

type InstallMcp struct{}

func (cmd InstallMcp) Run(req command.Request) {
	app := gomcp_command.NewApp("dodder", "Dodder zettelkasten MCP server")
	app.MCPArgs = []string{"mcp"}

	if err := app.InstallMCP(); err != nil {
		req.Cancel(err)
	}
}
```

**Step 3: Build dodder binary**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab && just build`
Expected: Builds successfully

**Step 4: Verify mcp command exists**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab && ./go/build/debug/dodder complete 2>/dev/null | grep -E '^(mcp|install-mcp)$'`
Expected: Output includes `mcp` and `install-mcp`

**Step 5: Commit**

```bash
git add go/internal/victor/commands_dodder/mcp.go go/internal/victor/commands_dodder/install_mcp.go
git commit -m "feat: add dodder mcp and install-mcp commands"
```

---

### Task 5: Update completion test

**Files:**
- Modify: `zz-tests_bats/current_version/complete.bats`

**Step 1: Add mcp and install-mcp to the completion assertions**

In `zz-tests_bats/current_version/complete.bats`, in the `complete_subcmd` test function, add these two lines in alphabetical order within the existing list:

- Add `install-mcp` after `info-workspace` (around line 146)
- Add `mcp` after `merge-tool` (around line 148)

**Step 2: Run the completion test**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab && just test-bats-targets complete.bats`
Expected: PASS

**Step 3: Commit**

```bash
git add zz-tests_bats/current_version/complete.bats
git commit -m "test: add mcp and install-mcp to completion test"
```

---

### Task 6: Run full test suite

**Step 1: Run unit tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab && just test-go`
Expected: All tests PASS

**Step 2: Run integration tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/crisp-baobab && just test-bats`
Expected: All tests PASS
