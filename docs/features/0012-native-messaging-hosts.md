---
date: 2026-04-05
promotion-criteria: A WASI-enabled WASM module communicates with a native
  messaging host binary over stdin/stdout pipes; Chaos spawns the host, wires
  the pipes to WASI fd 0/1, calls the WASM module's haustoria exports, and
  tears down the host; a round-trip BATS test compiles and decompiles a single
  object through the WASM+host pair
status: exploring
---

# Native Messaging Hosts for Haustoria

## Problem Statement

Haustoria (FDR-0007) translate between dodder's internal object representation
and external systems. The current implementations (`haustoria_caldav`,
`haustoria_orgmode`) hardcode both format translation AND transport in Chaos (the
Go binary). Each new external system (browser bookmarks, NewsBlur articles,
Kanban boards, proprietary APIs) requires a new Go package compiled into the
dodder binary.

FDR-0000's cosmology envisions WASM codec blobs handling format translation, but
leaves an open concern: WASM modules are sandboxed and cannot do I/O. When a
codec needs to fetch a VTODO from a CalDAV server or read an orgmode file over
SFTP, how does it reach the external system?

## Design

### Opaque Pipe via WASI File Descriptors

A haustoria declares an opaque external binary --- the **native messaging
host**. Chaos spawns this binary as a child process and exposes its stdin/stdout
to the WASM module as WASI file descriptors (fd 0 and fd 1). The WASM module
and host binary communicate over the pipe using whatever protocol they agree
on. Dodder does not specify the wire format between them.

Inspired by Chrome's native messaging hosts, but simpler: Chaos is just
plumbing.

Chaos's responsibilities:

1.  Spawn the host binary as a child process with configured environment
2.  Wire the host's stdout to the WASM module's WASI stdin (fd 0)
3.  Wire the WASM module's WASI stdout (fd 1) to the host's stdin
4.  Call the WASM module's exported haustoria functions
5.  Tear down the host process when done

The dodder-specified contract is **only the WASM exports**. Everything below
that --- how the codec talks to its host --- is opaque to dodder.

### Why This Works

wazero (v1.11.0) supports WASI `snapshot_preview1` with `ModuleConfig`:

``` go
wazero.NewModuleConfig().
    WithStdin(hostStdoutReader).   // host stdout → WASM stdin (fd 0)
    WithStdout(hostStdinWriter).   // WASM stdout → host stdin (fd 1)
    WithStderr(os.Stderr).
    WithName("")
```

The WASM module uses standard WASI `fd_read`/`fd_write` on fd 0/1. No custom
host-imported functions. No custom ABI for I/O. No dodder-specified wire
protocol.

This requires instantiating WASI in the wazero runtime, which dodder currently
does not do (`go/lib/charlie/wasm/main.go` uses a bare
`wazero.NewRuntimeConfig()` with no WASI). Existing non-WASI modules (tag
filters) are unaffected --- WASI host functions are only resolved if the guest
module imports them.

### Two-Tier Split

The haustoria's responsibilities split cleanly:

  ------------------------------------------------------------------------------------
  Tier                      Runs in           Responsibilities
  ------------------------- ----------------- ----------------------------------------
  **WASM codec**            wazero sandbox    Format translation + transport
                            (WASI)            orchestration via pipe.
                                              Content-addressed.

  **Native messaging host** OS process        Actual I/O: HTTP, SFTP, auth,
                                              credentials. Language-agnostic binary.
  ------------------------------------------------------------------------------------

The WASM module is not purely computational --- it drives the conversation with
the host binary over the pipe. But its only I/O channel is the WASI pipe to the
host process. It cannot access the network, filesystem, or credentials directly.

The codec WASM and the host binary are a **matched pair**. They agree on
whatever wire format makes sense for their domain --- JSON-RPC, protobuf, raw
bytes, length-prefixed messages, line-delimited text. A CalDAV codec might speak
a simple HTTP-proxy protocol with its host. An orgmode codec might use
newline-delimited file paths and content. Dodder doesn't care.

### WASM Exports (The Dodder-Specified Contract)

The haustoria WASM module exports four functions matching the existing
`charlie/haustoria.Haustoria` interface. Chaos calls these via the canonical ABI
(already implemented in `lib/charlie/wasm/canonical_abi.go`):

``` text
export compile(external_id_ptr: u32, external_id_len: u32)
    -> (result_ptr: u32, result_len: u32)

export decompile(request_ptr: u32, request_len: u32)
    -> (result_ptr: u32, result_len: u32)

export discover() -> (result_ptr: u32, result_len: u32)

export delete(external_id_ptr: u32, external_id_len: u32)
    -> (result_ptr: u32, result_len: u32)
```

Arguments and return values are serialized structs in WASM linear memory.
Serialization format matches the existing canonical ABI pattern
(`hotel/sku_wasm/sku_record.go`). Return values include error information.

During execution of any of these exports, the WASM module can read and write on
WASI fd 0/1 to communicate with the host process as needed.

### Host Process Lifecycle

Per-sync. The host process is started when a haustoria sync operation begins
and terminated when it completes.

1.  Chaos reads workspace config `[haustoria.native]`
2.  Spawns host binary with configured environment variables (credentials, URLs)
3.  Creates pipe pairs: host stdout → WASM stdin, WASM stdout → host stdin
4.  Instantiates WASI in the wazero runtime (if not already done)
5.  Instantiates WASM codec module with `WithStdin`/`WithStdout` wired to pipes
6.  Calls WASM exports (compile, decompile, discover, delete)
7.  Closes stdin pipe, sends SIGHUP, waits for exit

Step 7 reuses the existing pattern from `RoundTripperStdio.cancel()` in
`go/internal/tango/remote_http/round_tripper_stdio.go`: close stdin pipe, signal
the process, wait for completion, drain stderr.

### Workspace Config

``` toml
[haustoria]
type = "native"

[haustoria.native]
codec = "<@blake2b256-abc123... !wasm>"
host-binary = "/usr/lib/dodder/hosts/dodder-caldav-host"

[haustoria.native.env]
CALDAV_URL = "https://caldav.example.com/dav/calendars/user/tasks/"
CALDAV_USERNAME = "alice"

[haustoria.calendars.tasks]
url = "https://caldav.example.com/dav/calendars/user/tasks/"
type = "!task"
tags = ["project-alpha"]
```

The host binary is declared in the workspace config, not the type blob. The
same `!vtodo` type blob works across workspaces with different CalDAV servers
and credentials. The type defines what the object *is*; the workspace defines
how to *reach* the external system.

### Relationship to Primordial ABI

**Orthogonal.** Native messaging is I/O plumbing, not a type system function.
It does not change FDR-0000's 6-function primordial ABI.

When Chaos dispatches `project(vtodo_blob, "actionable", "summary")` through
FDR-0000's chain (Chaos → `!` → `!toml-type-v2` → codec), the codec WASM can
read/write on its WASI fds to fetch data from the host process. The pipe is
available throughout the WASM module's lifetime --- it's part of the module's
WASI environment, not a per-call mechanism.

### Security

- Host binary path is explicit in workspace config (same trust model as git
  hooks, ssh config, shell aliases)
- WASM codec is sandboxed; only I/O channel is WASI fd 0/1 to the host pipe
- Host stderr captured with `"(host) "` prefix via
  `delim_io.CopyWithPrefixOnDelim` (existing pattern in
  `round_tripper_stdio.go`)
- Future: content-addressed host binaries stored as blobs for tamper detection
  and reproducible deployment

## Implementation

### Go changes

**`go/lib/charlie/wasm/main.go`** --- instantiate WASI on the runtime:

``` go
import "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

func MakeRuntime(ctx context.Context) (rt *Runtime, err error) {
    inner := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())
    wasi_snapshot_preview1.MustInstantiate(ctx, inner)
    rt = &Runtime{inner: inner}
    return rt, err
}
```

**`go/lib/charlie/wasm/module_pool_builder.go`** --- accept stdio:

``` go
func (b *ModulePoolBuilder) WithStdio(
    stdin io.Reader, stdout io.Writer,
) *ModulePoolBuilder
```

**`go/lib/charlie/wasm/module_pool.go`** --- forward `ModuleConfig` with
stdin/stdout into `InstantiateModule`.

**`go/internal/mike/haustoria_native/main.go`** --- generic haustoria
implementation: spawn host binary, create WASM pool with stdio, implement
`haustoria.Haustoria` + `store_workspace.StoreLike`.

**`go/internal/echo/workspace_config_blobs/`** --- add `NativeConfig` to
workspace config.

### Host binaries

**`cmd/dodder-caldav-host/`** --- standalone binary reusing
`internal/hotel/caldav/` client. Reads from stdin, performs CalDAV operations,
writes to stdout. ~200 lines of glue.

**`cmd/dodder-orgmode-host/`** --- standalone binary extracting transport from
`haustoria_orgmode/transport_webdav.go` and `transport_sftp.go`.

## Migration Path

1.  Add WASI support to wazero Runtime. Non-breaking: existing modules don't
    import WASI functions.
2.  Add stdio plumbing to `ModulePoolBuilder`. Non-breaking: only used when
    configured.
3.  Build `haustoria_native` package. Coexists with existing implementations.
4.  Extract CalDAV transport into host binary.
5.  Build iCal codec WASM (Rust or TinyGo targeting `wasm32-wasi`).
6.  Workspace config `type = "native"` activates `haustoria_native`. Old
    `type = "caldav"` / `type = "orgmode"` remain until deprecated.

## Open Questions

- **WASI and existing modules.** Instantiating `wasi_snapshot_preview1` globally
  on the Runtime should not affect existing tag-filter WASM modules that don't
  import WASI functions, but this needs verification.
- **Module pooling with per-instance stdio.** Each pool instance needs its own
  stdin/stdout pipe. The current pool creates instances from a single compiled
  module with a shared config. May need per-borrow stdio injection or abandon
  pooling for haustoria modules (they're long-lived per-sync, not hot-path
  per-query).
- **Binary distribution.** Bundled with dodder? Installed separately?
  Content-addressed blobs in the repo?
- **Streaming large blobs.** WASM linear memory limits may constrain large
  CalDAV attachments. Chunked transfer over the pipe may be needed.
- **Error propagation.** When the host process crashes mid-call, WASI `fd_read`
  returns EOF. The WASM module must handle this gracefully and propagate the
  error through the export return value.

## More Information

- [FDR-0000: Type Interface Contracts](0000-from-chaos.md) --- WASM codec blobs,
  primordial ABI, the cosmology that native messaging hosts extend with I/O
- [FDR-0007: Pluggable Checkout Stores](0007-checkout-bridges.md) --- haustoria
  architecture, compilation model, the interface that WASM exports mirror
- [FDR-0009: External Object Index](0009-workspaces-indexes.md) --- caching and
  sync state infrastructure for haustoria
- Chrome native messaging:
  <https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging>
- wazero WASI support: <https://wazero.io/docs/how_the_optimizing_compiler_works/>
