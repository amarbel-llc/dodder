# serve-web Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `dodder serve-web` command that serves the MCP resource tree as a
read-only HTTP JSON API.

**Architecture:** Thin HTTP handler in `mcp_dodder` translates request paths to
`dodder://` URIs and delegates to an exported `ResourceReader` interface
wrapping the existing `typeResourceProvider`. New command in `commands_dodder`
wires it up.

**Tech Stack:** Go, gorilla/mux (already a dependency), existing `mcp_dodder`
resource provider.

**Rollback:** N/A --- purely additive new command.

--------------------------------------------------------------------------------

### Task 1: Export ResourceReader interface from mcp_dodder

**Files:**

- Modify: `go/internal/tango/mcp_dodder/resources.go`

**Step 1: Add the exported interface and constructor**

At the top of `resources.go`, after the `typeResourceProvider` struct, add:

``` go
// ResourceReader provides read-only access to dodder's MCP resource tree.
type ResourceReader interface {
    ReadResource(ctx context.Context, uri string) (*protocol.ResourceReadResult, error)
}

func NewResourceReader(
    utility command.Utility,
    repo *local_working_copy.Repo,
) ResourceReader {
    bridge := MakeBridge(utility)
    resources := server.NewResourceRegistry()
    index := makeTypeIndex(bridge)
    tagIdx := makeTagIndex(bridge)
    typeBlobCoder := type_blobs.MakeTypeStore(repo.GetEnvRepo())

    provider := &typeResourceProvider{
        registry:      resources,
        index:         index,
        tagIndex:      tagIdx,
        bridge:        bridge,
        store:         repo.GetStore(),
        typeBlobCoder: typeBlobCoder,
    }

    registerResources(resources, index, tagIdx, bridge)

    return provider
}
```

This requires adding these imports (some may already exist):

- `"code.linenisgreat.com/dodder/go/internal/golf/command"`
- `"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"`
- `"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"`

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/smart-teak/go && go build ./internal/tango/mcp_dodder/...`

Expected: clean build, no errors.

**Step 3: Commit**

    feat(mcp_dodder): export ResourceReader interface

    Allows other packages to read dodder MCP resources without depending
    on the full MCP server or transport layer.

--------------------------------------------------------------------------------

### Task 2: Add dodder://objects resource (all objects listing)

**Files:**

- Modify: `go/internal/tango/mcp_dodder/resources.go`

**Step 1: Register the static resource**

In `registerResources()`, after the existing `dodder://tags` resource
registration (around line 1102), add:

``` go
registry.RegisterResource(
    protocol.Resource{
        URI:         "dodder://objects",
        Name:        "All Objects",
        Description: "List of all objects in box format. See server instructions for box format grammar.",
        MimeType:    "text/plain",
    },
    func(ctx context.Context, uri string) (*protocol.ResourceReadResult, error) {
        result, err := bridge.RunCommand(
            ctx,
            "show",
            []string{"-format", "box", ":z", ":e", ":t"},
            500_000,
        )
        if err != nil {
            return nil, fmt.Errorf("list all objects: %w", err)
        }

        return &protocol.ResourceReadResult{
            Contents: []protocol.ResourceContent{{
                URI:      uri,
                MimeType: "text/plain",
                Text:     result.Stdout,
            }},
        }, nil
    },
)
```

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/smart-teak/go && go build ./internal/tango/mcp_dodder/...`

Expected: clean build.

**Step 3: Commit**

    feat(mcp_dodder): add dodder://objects resource for all-objects listing

--------------------------------------------------------------------------------

### Task 3: Add dodder://query/ resource (path-based query)

**Files:**

- Modify: `go/internal/tango/mcp_dodder/resources.go`

**Step 1: Add query handling to ReadResource**

In `typeResourceProvider.ReadResource()`, add a new case at the top of the
switch (before the existing `dodder://objects/` case):

``` go
case uri == "dodder://objects":
    return p.registry.ReadResource(ctx, uri)

case strings.HasPrefix(uri, "dodder://query/"):
    rest := strings.TrimPrefix(uri, "dodder://query/")
    terms := strings.Split(rest, "/")
    return p.readQuery(ctx, terms)
```

Then add the method:

``` go
func (p *typeResourceProvider) readQuery(
    ctx context.Context,
    terms []string,
) (*protocol.ResourceReadResult, error) {
    args := append([]string{"-format", "json"}, terms...)
    result, err := p.bridge.RunCommand(ctx, "show", args, 500_000)
    if err != nil {
        return nil, fmt.Errorf("query %v: %w", terms, err)
    }

    return &protocol.ResourceReadResult{
        Contents: []protocol.ResourceContent{{
            URI:      "dodder://query/" + strings.Join(terms, "/"),
            MimeType: "application/json",
            Text:     result.Stdout,
        }},
    }, nil
}
```

Also register the template in `registerResources()`:

``` go
registry.RegisterTemplate(
    protocol.ResourceTemplate{
        URITemplate: "dodder://query/{terms}",
        Name:        "Query",
        Description: "Execute a dodder query. Path segments are AND-combined query terms. Returns results in JSON format.",
        MimeType:    "application/json",
    },
    nil,
)
```

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/smart-teak/go && go build ./internal/tango/mcp_dodder/...`

Expected: clean build.

**Step 3: Commit**

    feat(mcp_dodder): add dodder://query/ resource for path-based queries

--------------------------------------------------------------------------------

### Task 4: Create HTTP handler and CORS middleware

**Files:**

- Create: `go/internal/tango/mcp_dodder/server_web.go`

**Step 1: Write the HTTP handler**

Create `go/internal/tango/mcp_dodder/server_web.go`:

``` go
package mcp_dodder

import (
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "strconv"
    "strings"

    "code.linenisgreat.com/dodder/go/lib/_/interfaces"
    "github.com/gorilla/mux"
)

type WebServer struct {
    Reader     ResourceReader
    CorsOrigin string
    UI         interfaces.UIFile
}

func (s *WebServer) Handler() http.Handler {
    r := mux.NewRouter()

    r.PathPrefix("/").HandlerFunc(s.handleResource).Methods("GET")
    r.PathPrefix("/").HandlerFunc(s.handleOptions).Methods("OPTIONS")

    return r
}

func (s *WebServer) handleResource(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/")
    uri := "dodder://" + path

    result, err := s.Reader.ReadResource(r.Context(), uri)
    if err != nil {
        errMsg := err.Error()
        if strings.Contains(errMsg, "not found") {
            s.writeError(w, http.StatusNotFound, uri)
            return
        }
        s.writeError(w, http.StatusInternalServerError, errMsg)
        return
    }

    if len(result.Contents) == 0 {
        s.writeError(w, http.StatusNotFound, uri)
        return
    }

    content := result.Contents[0]
    mimeType := content.MimeType
    if mimeType == "" {
        mimeType = "application/json"
    }

    s.setCorsHeaders(w)
    w.Header().Set("Content-Type", mimeType)
    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, content.Text)
}

func (s *WebServer) handleOptions(w http.ResponseWriter, r *http.Request) {
    s.setCorsHeaders(w)
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    w.WriteHeader(http.StatusNoContent)
}

func (s *WebServer) setCorsHeaders(w http.ResponseWriter) {
    origin := s.CorsOrigin
    if origin == "" {
        origin = "*"
    }
    w.Header().Set("Access-Control-Allow-Origin", origin)
}

func (s *WebServer) writeError(w http.ResponseWriter, status int, msg string) {
    s.setCorsHeaders(w)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    resp := struct {
        Error string `json:"error"`
    }{Error: msg}

    json.NewEncoder(w).Encode(resp)
}

func (s *WebServer) ListenAndServe(network, address string) error {
    listener, err := net.Listen(network, address)
    if err != nil {
        return fmt.Errorf("listen %s %s: %w", network, address, err)
    }

    defer listener.Close()

    addr := listener.Addr().(*net.TCPAddr)
    s.UI.Log().Printf(
        "starting HTTP server on port: %q",
        strconv.Itoa(addr.Port),
    )

    return http.Serve(listener, s.Handler())
}
```

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/smart-teak/go && go build ./internal/tango/mcp_dodder/...`

Expected: clean build. The `interfaces.UIFile` type must have a `Log()` method
--- verify by checking `lux://lsp/hover` on the type. If the interface doesn't
match, adjust to use whatever logging pattern `remote_http/server.go` uses.

**Step 3: Commit**

    feat(mcp_dodder): add WebServer HTTP handler with CORS support

    Translates HTTP request paths to dodder:// URIs and delegates to
    ResourceReader. Prints port on startup in the same format as `dodder serve`.

--------------------------------------------------------------------------------

### Task 5: Create the serve-web command

**Files:**

- Create: `go/internal/victor/commands_dodder/serve_web.go`

**Step 1: Write the command**

Create `go/internal/victor/commands_dodder/serve_web.go`:

``` go
package commands_dodder

import (
    "code.linenisgreat.com/dodder/go/internal/delta/env_ui"
    "code.linenisgreat.com/dodder/go/internal/golf/command"
    "code.linenisgreat.com/dodder/go/internal/hotel/command_components"
    "code.linenisgreat.com/dodder/go/internal/tango/mcp_dodder"
    "code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
    "code.linenisgreat.com/dodder/go/lib/bravo/errors"
    "code.linenisgreat.com/dodder/go/lib/charlie/flags"
)

func init() {
    utility.AddCmd("serve-web", &ServeWeb{})
}

type ServeWeb struct {
    command_components.Env
    command_components_dodder.EnvRepo
    command_components_dodder.LocalWorkingCopy

    CorsOrigin string
}

func (cmd *ServeWeb) SetFlagDefinitions(
    flagSet interfaces.CLIFlagDefinitions,
) {
    cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

    flags.StringVar(
        &cmd.CorsOrigin,
        "cors-origin",
        "*",
        "Access-Control-Allow-Origin header value",
    )
}

func (cmd ServeWeb) Run(req command.Request) {
    args := req.PopArgs()
    errors.ContextSetCancelOnSIGHUP(req)

    envLocal := cmd.MakeEnvWithOptions(
        req,
        env_ui.Options{
            UIFileIsStderr: true,
            IgnoreTtyState: true,
        },
    )

    repo := cmd.MakeLocalWorkingCopyFromEnvLocal(envLocal)

    reader := mcp_dodder.NewResourceReader(req.Utility, repo)

    server := &mcp_dodder.WebServer{
        Reader:     reader,
        CorsOrigin: cmd.CorsOrigin,
        UI:         envLocal,
    }

    var network, address string

    switch len(args) {
    case 0:
        network = "tcp"
        address = ":0"

    case 1:
        network = args[0]

    default:
        network = args[0]
        address = args[1]
    }

    if err := server.ListenAndServe(network, address); err != nil {
        envLocal.Cancel(err)
    }
}
```

Note: The import for `interfaces` needs to be
`"code.linenisgreat.com/dodder/go/lib/_/interfaces"`. Check the existing
`serve.go` to confirm the exact import for `interfaces.CLIFlagDefinitions` ---
it may be `interfaces.CommandComponentWriter` pattern. Match the existing
`Serve` command's structure exactly.

**Step 2: Verify it compiles**

Run:
`cd /home/sasha/eng/repos/dodder/.worktrees/smart-teak/go && go build ./cmd/dodder/...`

Expected: clean build. The `serve-web` subcommand should now be registered.

**Step 3: Verify it runs**

Run from a dodder repo:

``` bash
cd /home/sasha/eng/repos/dodder/.worktrees/smart-teak/go
just build
cd /tmp && dodder init && dodder serve-web tcp :0
```

Expected: prints `starting HTTP server on port: "<port>"` to stderr.

**Step 4: Commit**

    feat: add `dodder serve-web` command

    Read-only HTTP server that exposes the dodder MCP resource tree.
    Supports configurable CORS origin via --cors-origin flag.

--------------------------------------------------------------------------------

### Task 6: Add serve-web to completion test

**Files:**

- Modify: `zz-tests_bats/current_version/complete.bats`

**Step 1: Add serve-web to the expected subcommands**

In the `complete_subcmd` test function, add `serve-web` to the expected output
list, alphabetically after `serve`:

            serve
            serve-web
            show

**Step 2: Run the completion test**

Run: `just test-bats-targets complete.bats`

Expected: PASS.

**Step 3: Commit**

    test: add serve-web to subcommand completion test

--------------------------------------------------------------------------------

### Task 7: Write BATS integration test for serve-web

**Files:**

- Create: `zz-tests_bats/current_version/serve_web.bats`
- Modify: `zz-tests_bats/lib/common.bash` (add `start_web_server` helper)

**Step 1: Add start_web_server helper to common.bash**

After the existing `start_server` function (around line 313), add:

``` bash
function start_web_server {
  dir="$1"

  coproc web_server {
    if [[ -n $dir ]]; then
      cd "$dir"
    fi

    # shellcheck disable=SC2068
    "$DODDER_BIN" serve-web ${cmd_dodder_def[@]} tcp :0
  }

  # shellcheck disable=SC2154
  read -r output <&"${web_server[0]}"

  if [[ $output =~ (starting HTTP server on port: \"([0-9]+)\") ]]; then
    export web_port="${BASH_REMATCH[2]}"
  else
    fail <<-EOM
            unable to get port info from dodder serve-web.
            server output: $output
        EOM
  fi
}
```

**Step 2: Write the test file**

Create `zz-tests_bats/current_version/serve_web.bats`:

``` bash
#! /usr/bin/env bats

setup() {
    load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

    # for shellcheck SC2154
    export output

    copy_from_version "$DIR"
}

teardown() {
    if [[ -n "${web_server_PID:-}" ]]; then
        kill "$web_server_PID" 2>/dev/null || true
    fi

    chflags_nouchg
}

# bats file_tags=user_story:serve_web

function serve_web_types { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/types"
    assert_success
    assert_output --partial '"object-id"'
}

function serve_web_tags { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/tags"
    assert_success
    assert_output --partial '"object-id"'
}

function serve_web_objects { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/objects"
    assert_success
    assert_output --partial '['
}

function serve_web_types_index { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/types_index"
    assert_success
    assert_output --partial '"total_words"'
}

function serve_web_single_type { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/types/md"
    assert_success
    assert_output --partial '"object-id"'
    assert_output --partial '"objects-resource"'
}

function serve_web_type_objects { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/types/md/objects"
    assert_success
    assert_output --partial '!md'
}

function serve_web_query { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/query/:z"
    assert_success
    assert_output --partial '"object-id"'
}

function serve_web_not_found { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/objects/nonexistent/zettel"
    assert_output --partial '"error"'
}

function serve_web_cors_header { # @test
    start_web_server .

    run http GET "http://localhost:$web_port/types"
    assert_success
    assert_output --partial 'Access-Control-Allow-Origin'
}
```

Note: The `http` command (httpie) may not include response headers by default.
Adjust the CORS test based on how `http` is invoked in the test environment ---
it may need `--headers` or `--print=h` flag. If `http` is not available in the
test environment, check what HTTP client tools are in the devshell and adjust
accordingly.

**Step 3: Run the tests**

Run: `just test-bats-targets serve_web.bats`

Expected: all tests pass.

**Step 4: Commit**

    test: add BATS integration tests for serve-web

    Tests type/tag/object listing, single-resource lookup, query endpoint,
    404 handling, and CORS headers.

--------------------------------------------------------------------------------

### Task 8: Verify full test suite passes

**Step 1: Run all tests**

Run: `just test`

Expected: all tests pass, including the new serve-web tests and the updated
completion test.

**Step 2: Final commit (if any fixups needed)**

Only if test failures require adjustments.
