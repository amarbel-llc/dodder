# Madder MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task.

**Goal:** Add an `mcp` subcommand to madder that exposes blob store commands as
MCP tools via go-mcp's raw server layer.

**Architecture:** Synthetic Request bridge calls existing madder commands
unchanged. Output captured via `io.Writer`-backed printers threaded through
`config_cli.Config`. A `LimitingWriter` bounds all output.

**Tech Stack:** go-mcp (`github.com/amarbel-llc/go-lib-mcp`), existing
`juliett/command` framework, existing madder commands in `lima/commands_madder`.

**Design doc:** `docs/plans/2026-02-22-madder-mcp-server-design.md`

---

### Task 1: Refactor ui.printer to support io.Writer

The `printer` struct currently stores `*os.File` and writes to it directly.
Refactor to store `io.Writer` so MCP handlers can use buffer-backed printers.

**Files:**
- Modify: `go/internal/bravo/ui/printer.go`

**Step 1: Refactor printer struct to use io.Writer**

Replace the `file *os.File` field with `writer io.Writer` and a separate `file
*os.File` field (nullable, only set when constructed from a file). All write
methods use `writer`. `GetFile()` returns the file when available, nil otherwise.

```go
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/primordial"
	"code.linenisgreat.com/dodder/go/internal/_/stack_frame"
)

func MakePrinter(file *os.File) printer {
	return MakePrinterOn(file, true)
}

func MakePrinterOn(file *os.File, on bool) printer {
	return printer{
		writer: file,
		file:   file,
		isTty:  primordial.IsTty(file),
		on:     on,
	}
}

func MakePrinterFromWriter(w io.Writer) printer {
	return printer{
		writer: w,
		file:   nil,
		isTty:  false,
		on:     true,
	}
}

type printer struct {
	writer io.Writer
	file   *os.File
	isTty  bool
	on     bool
}

var _ Printer = printer{}

func (printer printer) withOn(on bool) printer {
	printer.on = on
	return printer
}

func (printer printer) GetPrinter() Printer {
	return printer
}

func (printer printer) Write(b []byte) (n int, err error) {
	if !printer.on {
		n = len(b)
		return n, err
	}

	return printer.writer.Write(b)
}

func (printer printer) GetFile() *os.File {
	return printer.file
}

func (printer printer) IsTty() bool {
	return printer.isTty
}

//go:noinline
func (printer printer) Caller(skip int) Printer {
	if !printer.on {
		return Null
	}

	stackFrame, _ := stack_frame.MakeFrame(skip + 1)

	return prefixPrinter{
		Printer: printer,
		prefix:  stackFrame.StringNoFunctionName() + " ",
	}
}

func (printer printer) PrintDebug(args ...any) (err error) {
	if !printer.on {
		return err
	}

	_, err = fmt.Fprintf(
		printer.writer,
		strings.Repeat("%#v ", len(args))+"\n",
		args...,
	)

	return err
}

func (printer printer) Print(args ...any) (err error) {
	if !printer.on {
		return err
	}

	_, err = fmt.Fprintln(
		printer.writer,
		args...,
	)

	return err
}

//go:noinline
func (printer printer) printfStack(
	depth int,
	format string,
	args ...any,
) (err error) {
	if !printer.on {
		return err
	}

	stackFrame, _ := stack_frame.MakeFrame(1 + depth)
	format = "%s" + format
	args = append([]any{stackFrame}, args...)

	_, err = fmt.Fprintln(
		printer.writer,
		fmt.Sprintf(format, args...),
	)

	return err
}

func (printer printer) Printf(format string, args ...any) (err error) {
	if !printer.on {
		return err
	}

	_, err = fmt.Fprintln(
		printer.writer,
		fmt.Sprintf(format, args...),
	)

	return err
}
```

**Step 2: Build to verify no compile errors**

Run: `just build` from `go/`

Expected: successful build. All existing code continues to work because
`MakePrinter` and `MakePrinterOn` still accept `*os.File` and set the `writer`
field to it.

**Step 3: Commit**

```
feat(ui): refactor printer to support io.Writer

Add MakePrinterFromWriter constructor and change internal storage from
*os.File to io.Writer. Existing *os.File constructors continue to work.
```

---

### Task 2: Add MakeStdFromWriter to fd

**Files:**
- Modify: `go/internal/foxtrot/fd/std.go`

**Step 1: Add the new constructor**

```go
package fd

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
)

type Std struct {
	ui.Printer
}

func MakeStd(f *os.File) Std {
	return Std{
		Printer: ui.MakePrinter(f),
	}
}

func MakeStdFromWriter(w io.Writer) Std {
	return Std{
		Printer: ui.MakePrinterFromWriter(w),
	}
}
```

**Step 2: Build to verify**

Run: `just build` from `go/`

Expected: successful build.

**Step 3: Commit**

```
feat(fd): add MakeStdFromWriter constructor

Supports creating fd.Std from an io.Writer for MCP output capture.
```

---

### Task 3: Add CustomOut/CustomErr to config_cli.Config

**Files:**
- Modify: `go/internal/echo/config_cli/main.go`

**Step 1: Add writer fields**

Add `CustomOut` and `CustomErr` fields to `Config`. These are `io.Writer` fields
tagged `toml:"-"` so they don't affect serialization. They're nil by default
(normal CLI mode).

```go
package config_cli

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/cli"
	"code.linenisgreat.com/dodder/go/internal/delta/debug"
)

type Config struct {
	Debug   debug.Options
	Verbose bool
	Quiet   bool
	Todo    bool
	dryRun  bool

	// CustomOut and CustomErr override os.Stdout/os.Stderr when set.
	// Used by MCP handlers to capture command output into buffers.
	CustomOut io.Writer `toml:"-"`
	CustomErr io.Writer `toml:"-"`
}
```

The rest of the file (`SetFlagDefinitions`, `Default`, getters) stays unchanged.

**Step 2: Build to verify**

Run: `just build` from `go/`

Expected: successful build. No behavior change — fields default to nil.

**Step 3: Commit**

```
feat(config_cli): add CustomOut/CustomErr writer fields

Used by MCP handlers to redirect command output to buffers.
```

---

### Task 4: Wire custom writers in env_ui.Make

**Files:**
- Modify: `go/internal/kilo/command_components_madder/env_blob_store.go`

**Step 1: Pass custom writers through to env_ui**

The `MakeEnvBlobStore` method already type-asserts `configAny` to
`*config_cli.Config`. After the switch, check for `CustomOut`/`CustomErr` and
pass them as `env_ui.Options`.

First, add writer fields to `env_ui.Options`:

Modify `go/internal/golf/env_ui/options.go`:

```go
package env_ui

import "io"

type OptionsGetter interface {
	GetEnvOptions() Options
}

type Options struct {
	UIFileIsStderr   bool
	IgnoreTtyState   bool
	UIPrintingPrefix string
	CustomOut        io.Writer
	CustomErr        io.Writer
}
```

Then modify `go/internal/golf/env_ui/main.go` in `Make()` to use custom writers when
set:

In the `Make` function, after the existing `env` construction, replace the
hardcoded `os.Stdin/Stdout/Stderr`:

```go
func Make(
	context errors.Context,
	cliConfig domain_interfaces.CLIConfigProvider,
	debugOptions debug.Options,
	options Options,
) *env {
	env := &env{
		Context:   context,
		options:   options,
		in:        fd.MakeStd(os.Stdin),
		cliConfig: cliConfig,
	}

	if options.CustomOut != nil {
		env.out = fd.MakeStdFromWriter(options.CustomOut)
	} else {
		env.out = fd.MakeStd(os.Stdout)
	}

	if options.CustomErr != nil {
		env.err = fd.MakeStdFromWriter(options.CustomErr)
	} else {
		env.err = fd.MakeStd(os.Stderr)
	}

	if options.UIFileIsStderr {
		env.ui = env.err
	} else {
		env.ui = env.out
	}

	{
		var err error

		if env.debug, err = debug.MakeContext(context, debugOptions); err != nil {
			context.Cancel(err)
		}
	}

	if cliConfig != nil && cliConfig.GetVerbose() && !cliConfig.GetQuiet() {
		ui.SetVerbose(true)
	} else {
		ui.SetOutput(io.Discard)
	}

	if cliConfig != nil && cliConfig.GetTodo() {
		ui.SetTodoOn()
	}

	return env
}
```

Then modify `go/internal/kilo/command_components_madder/env_blob_store.go` to thread
the custom writers from config into env_ui options:

```go
func (cmd EnvBlobStore) MakeEnvBlobStore(
	req command.Request,
) env_repo.BlobStoreEnv {
	configAny := req.Utility.GetConfigAny()

	var debugOptions debug.Options
	var cliConfig domain_interfaces.CLIConfigProvider
	var envOptions env_ui.Options

	switch c := configAny.(type) {
	case *config_cli.Config:
		debugOptions = c.Debug
		cliConfig = c
		envOptions.CustomOut = c.CustomOut
		envOptions.CustomErr = c.CustomErr
	case *repo_config_cli.Config:
		debugOptions = c.Debug
		cliConfig = c
	default:
		panic(fmt.Sprintf("unsupported config type: %T", configAny))
	}

	dir := env_dir.MakeDefault(
		req,
		req.Utility.GetName(),
		debugOptions,
	)

	envUI := env_ui.Make(
		req,
		cliConfig,
		debugOptions,
		envOptions,
	)

	envLocal := env_local.Make(envUI, dir)

	return env_repo.MakeBlobStoreEnv(envLocal)
}
```

**Step 2: Build to verify**

Run: `just build` from `go/`

Expected: successful build. No behavior change — CustomOut/CustomErr are nil by
default.

**Step 3: Commit**

```
feat(env_ui): wire custom output writers from config

When config_cli.Config has CustomOut/CustomErr set, env_ui uses
writer-backed fd.Std instead of os.Stdout/os.Stderr.
```

---

### Task 5: Fix global ui calls in madder commands

Several madder commands use `ui.Out()` and `ui.Err()` (global printers) instead
of the env-based `env.GetUI()` and `env.GetErr()`. These bypass output capture.

**Files:**
- Modify: `go/internal/lima/commands_madder/list.go`
- Modify: `go/internal/lima/commands_madder/info_repo.go`

**Step 1: Fix list.go**

Change `ui.Out().Printf(...)` to `envBlobStore.GetUI().Printf(...)`:

```go
func (cmd List) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	for _, blobStore := range blobStores {
		envBlobStore.GetUI().Printf(
			"%s: %s",
			blobStore.Path.GetId(),
			blobStore.GetBlobStoreDescription(),
		)
	}
}
```

Remove the `"code.linenisgreat.com/dodder/go/internal/bravo/ui"` import if it
becomes unused.

**Step 2: Fix info_repo.go line 107**

Change `env.GetUI().GetFile()` to `env.GetUIFile()`:

```go
		case "blob_stores-0-config":
			blobStoreConfig := blobStore.ConfigNamed.Config

			if err := cmd.PrintBlobStoreConfig(
				env,
				&blob_store_configs.TypedConfig{
					Type: blobStoreConfig.Type,
					Blob: blobStoreConfig.Blob,
				},
				env.GetUIFile(),
			); err != nil {
				env.Cancel(err)
				return
			}
```

**Step 3: Build and test**

Run: `just build` from `go/`

Expected: successful build.

Note: `sync.go`, `fsck.go`, `write.go`, and `cat.go` also use global `ui.Out()`
and `ui.Err()` for progress/diagnostic output. These are lower priority — the
main output paths use env-based printers. Fix them in a follow-up if needed.

**Step 4: Commit**

```
fix(madder): use env-based output in list and info-repo commands

Replace global ui.Out()/ui.Err() with env.GetUI()/env.GetUIFile() so
output can be captured by MCP handlers.
```

---

### Task 6: Add go-mcp dependency

**Files:**
- Modify: `go/go.mod`
- Modify: `go/go.sum`

**Step 1: Add the dependency**

Run from `go/`:
```
go get github.com/amarbel-llc/go-lib-mcp@latest
```

**Step 2: Verify**

Run: `go build ./...` from `go/`

Expected: successful build.

**Step 3: Commit**

```
deps: add go-lib-mcp dependency

Provides MCP server, transport, protocol, and output packages.
```

---

### Task 7: Create LimitingWriter

A writer that enforces byte limits during writes. Silently discards data past
the limit so commands don't error.

**Files:**
- Create: `go/internal/lima/mcp_madder/limiting_writer.go`
- Create: `go/internal/lima/mcp_madder/limiting_writer_test.go`

**Step 1: Write the test**

```go
package mcp_madder

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

**Step 2: Run test to verify it fails**

Run: `go test -v ./src/lima/mcp_madder/` from `go/`

Expected: FAIL (package does not exist yet)

**Step 3: Write the implementation**

```go
package mcp_madder

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

**Step 4: Run tests to verify they pass**

Run: `go test -v ./src/lima/mcp_madder/` from `go/`

Expected: PASS (all 4 tests)

**Step 5: Commit**

```
feat(mcp_madder): add LimitingWriter for bounded MCP output
```

---

### Task 8: Create the synthetic Request bridge

The bridge translates MCP tool calls into synthetic `command.Request` executions
with output capture.

**Files:**
- Create: `go/internal/lima/mcp_madder/bridge.go`
- Create: `go/internal/lima/mcp_madder/bridge_test.go`

**Step 1: Write a test that runs `list` through the bridge**

This test requires a blob store to exist. Use a temp directory with madder init.
If that's too complex for a unit test, write a simpler test that verifies the
bridge handles an unknown command correctly.

```go
package mcp_madder

import (
	"context"
	"testing"
)

func TestBridgeUnknownCommand(t *testing.T) {
	bridge := MakeBridge()
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

**Step 2: Run test to verify it fails**

Run: `go test -v ./src/lima/mcp_madder/ -run TestBridgeUnknownCommand` from
`go/`

Expected: FAIL (Bridge type does not exist)

**Step 3: Write the bridge implementation**

```go
package mcp_madder

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/config_cli"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/lima/commands_madder"
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

func MakeBridge() Bridge {
	return Bridge{
		utility: commands_madder.GetUtility(),
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

	utility := command.MakeUtility("madder", config)

	// Re-register all commands from the shared utility
	for name, cmd := range b.utility.AllCmds() {
		utility.AddCmd(name, cmd)
	}

	errCtx := errors.MakeContext(ctx)

	args := make([]string, 0, 2+len(cliArgs))
	args = append(args, "madder", cmdName)
	args = append(args, cliArgs...)

	cmd, flagSet, ok := utility.MakeCmdAndFlagSet(errCtx, args)
	if !ok {
		return BridgeResult{}, errCtx.Cause()
	}

	req, ok := utility.MakeRequest(errCtx, cmd, flagSet)
	if !ok {
		return BridgeResult{}, errCtx.Cause()
	}

	cmd.Run(req)

	result := BridgeResult{
		Stdout:    outWriter.String(),
		Stderr:    errWriter.String(),
		Truncated: outWriter.Truncated(),
		BytesSeen: outWriter.BytesSeen(),
	}

	return result, errCtx.Cause()
}
```

**Step 4: Run tests**

Run: `go test -v ./src/lima/mcp_madder/` from `go/`

Expected: PASS

**Step 5: Commit**

```
feat(mcp_madder): add synthetic Request bridge

Translates command names and CLI args into command.Request executions
with output captured via LimitingWriter.
```

---

### Task 9: Register first MCP tool (list) and create server

Register the simplest command as an MCP tool and wire the server entry point.

**Files:**
- Create: `go/internal/lima/mcp_madder/server.go`
- Create: `go/internal/lima/commands_madder/mcp.go`

**Step 1: Create the MCP server with tool registration**

```go
package mcp_madder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/amarbel-llc/go-lib-mcp/protocol"
	"github.com/amarbel-llc/go-lib-mcp/server"
	"github.com/amarbel-llc/go-lib-mcp/transport"
)

const defaultMaxBytes = 100_000

func RunServer() error {
	bridge := MakeBridge()
	tools := server.NewToolRegistry()

	registerTools(tools, bridge)

	t := transport.NewStdio(os.Stdin, os.Stdout)
	srv, err := server.New(t, server.Options{
		ServerName:    "madder",
		ServerVersion: "0.1.0",
		Tools:         tools,
	})
	if err != nil {
		return err
	}

	return srv.Run(context.Background())
}

func registerTools(tools *server.ToolRegistry, bridge Bridge) {
	tools.Register(
		"madder_list",
		"List available blob stores with their IDs and descriptions",
		json.RawMessage(`{
			"type": "object",
			"properties": {},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "list", nil),
	)
}

type paramTranslator func(args json.RawMessage) ([]string, error)

func makeBridgeHandler(
	bridge Bridge,
	cmdName string,
	translate paramTranslator,
) func(context.Context, json.RawMessage) (*protocol.ToolCallResult, error) {
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResult, error) {
		var cliArgs []string

		if translate != nil {
			var err error
			if cliArgs, err = translate(args); err != nil {
				return protocol.ErrorResult(
					fmt.Sprintf("Invalid arguments: %v", err),
				), nil
			}
		}

		result, err := bridge.RunCommand(ctx, cmdName, cliArgs, defaultMaxBytes)
		if err != nil {
			return protocol.ErrorResult(err.Error()), nil
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

		return &protocol.ToolCallResult{
			Content: []protocol.ContentBlock{
				protocol.TextContent(output),
			},
		}, nil
	}
}
```

**Step 2: Create the mcp subcommand**

```go
package commands_madder

import (
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/lima/mcp_madder"
)

func init() {
	utility.AddCmd("mcp", &Mcp{})
}

type Mcp struct{}

func (cmd Mcp) Run(req command.Request) {
	if err := mcp_madder.RunServer(); err != nil {
		req.Cancel(err)
	}
}
```

**Step 3: Build**

Run: `just build` from `go/`

Expected: successful build.

**Step 4: Smoke test the MCP server manually**

Run from repo root:
```
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}' | go/build/debug/madder mcp
```

Expected: JSON response with server info.

**Step 5: Commit**

```
feat(madder): add mcp subcommand with list tool

Starts an MCP server over stdio exposing madder_list as the first tool.
```

---

### Task 10: Register remaining read-only tools

Add `cat`, `cat-ids`, `info-repo`, and `fsck` tools with parameter translation.

**Files:**
- Modify: `go/internal/lima/mcp_madder/server.go`

**Step 1: Add param translators and tool registrations**

Append to the `registerTools` function:

```go
	// cat: read blob contents by SHA
	tools.Register(
		"madder_cat",
		"Output blob contents by SHA digest",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"sha": {
					"type": "string",
					"description": "SHA digest of the blob to read"
				},
				"prefix_sha": {
					"type": "boolean",
					"description": "Prefix each line with the SHA digest"
				},
				"blob_store": {
					"type": "string",
					"description": "Blob store index to read from"
				}
			},
			"required": ["sha"],
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "cat", func(args json.RawMessage) ([]string, error) {
			var p struct {
				SHA       string `json:"sha"`
				PrefixSha bool   `json:"prefix_sha"`
				BlobStore string `json:"blob_store"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.PrefixSha {
				cliArgs = append(cliArgs, "-prefix-sha")
			}
			if p.BlobStore != "" {
				cliArgs = append(cliArgs, "-blob-store", p.BlobStore)
			}
			cliArgs = append(cliArgs, p.SHA)
			return cliArgs, nil
		}),
	)

	// cat-ids: list all blob IDs
	tools.Register(
		"madder_cat_ids",
		"List all blob IDs in one or more blob stores",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"blob_stores": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Blob store IDs to list (all if omitted)"
				}
			},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "cat-ids", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStores []string `json:"blob_stores"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return p.BlobStores, nil
		}),
	)

	// info-repo: query blob store configuration
	tools.Register(
		"madder_info_repo",
		"Show blob store configuration and metadata",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"blob_store": {
					"type": "string",
					"description": "Blob store index to query"
				},
				"keys": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Config keys to query (e.g. config-immutable, compression-type, xdg)"
				}
			},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "info-repo", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStore string   `json:"blob_store"`
				Keys      []string `json:"keys"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.BlobStore != "" {
				cliArgs = append(cliArgs, p.BlobStore)
			}
			cliArgs = append(cliArgs, p.Keys...)
			return cliArgs, nil
		}),
	)

	// fsck: verify blob store integrity
	tools.Register(
		"madder_fsck",
		"Check blob store consistency and verify blob integrity",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"blob_stores": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Blob store IDs to check (all if omitted)"
				}
			},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "fsck", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStores []string `json:"blob_stores"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return p.BlobStores, nil
		}),
	)
```

**Step 2: Build**

Run: `just build` from `go/`

Expected: successful build.

**Step 3: Commit**

```
feat(mcp_madder): register cat, cat-ids, info-repo, fsck tools
```

---

### Task 11: Register write tools

Add `write`, `sync`, `init`, `init-from`, `init-inventory-archive`,
`init-pointer`, and `pack` tools.

**Files:**
- Modify: `go/internal/lima/mcp_madder/server.go`

**Step 1: Add registrations**

Follow the same pattern as Task 10 for each command. The `init` variants take
no args. `write` takes `paths` array and `check` bool. `sync` takes `source`,
`destinations`, and `limit`. `pack` takes `delete_loose` bool.

**Step 2: Build and commit**

```
feat(mcp_madder): register write, sync, init, and pack tools
```

---

### Task 12: Integration test with bats

Write a bats test that starts the MCP server and exercises a tool.

**Files:**
- Create: `zz-tests_bats/madder_mcp.bats`

**Step 1: Write the test**

Use `@bats:focus-include mcp` tag. The test should:

1. Initialize a blob store in a temp dir
2. Write a blob
3. Start `madder mcp` and send an initialize + tools/list request
4. Verify the response contains `madder_list`

Refer to existing bats tests in `zz-tests_bats/` for setup patterns and
the batman/sandcastle test infrastructure.

**Step 2: Run the test**

Run: `just test-bats-targets madder_mcp.bats` from repo root

**Step 3: Commit**

```
test: add bats integration test for madder MCP server
```

---

### Task 13: Update nix build to include mcp subcommand

The nix build (`flake.nix`) lists `subPackages`. The `mcp` subcommand is part of
the existing `cmd/madder` binary so no nix changes are needed — it's already
built. Verify this.

**Step 1: Build with nix**

Run: `just build-nix` from `go/` (or `nix build` from repo root)

**Step 2: Verify mcp subcommand exists in nix output**

Run: `result/bin/madder mcp` and verify it starts (then ctrl-c).

**Step 3: Commit (if any changes needed)**

```
chore: verify madder mcp subcommand in nix build
```

---

## Deferred Work

These items are explicitly out of scope for this plan:

- **Fix remaining global ui calls** in `sync.go`, `fsck.go`, `write.go`,
  `cat.go` (diagnostic output — lower priority)
- **Dodder read-only tools** (show, status, info — next phase)
- **Error package extraction** into go-mcp (revisit after learning from this
  migration)
- **Concurrent MCP tool calls** (requires refactoring global `ui` package state)
- **Tool mapping declarations** for Claude Code integration (after tools are
  stable)
- **Context saving params** (`head`/`tail`/`offset`/`limit`) on individual
  tools (add when needed based on usage)
