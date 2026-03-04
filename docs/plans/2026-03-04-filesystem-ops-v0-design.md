# FilesystemOps V0 Design

## Summary

Define a versioned `FilesystemOpsV0` interface that abstracts all OS operations
needed by `store_fs` (and eventually `store_browser`). This is the first step
toward compiling workspace modules to WASM, where the guest implements workspace
logic and calls back to the host for file I/O.

## Motivation

- **WASM workspace modules**: `store_fs` logic should compile to WASM. The guest
  cannot call `os.*` directly; it needs host-provided functions for all I/O.
- **Testability**: a clean interface makes `store_fs` testable with in-memory
  filesystem implementations.
- **Portability**: decouples workspace logic from OS-specific syscalls.

## Architecture

```
FilesystemOpsV0 interface       <- clean Go interface, no OS imports
        |
        | implemented by
        v
OsFilesystemOps (concrete)      <- wraps os.*, takes errors.Context at construction
        |
        | wrapped by (future step)
        v
WasmFilesystemOpsAdapter        <- translates V0 into integer-handle WASM imports
```

The interface uses idiomatic Go types (`io.ReadCloser`, `fs.WalkDirFunc`, etc.).
A separate WASM ABI layer (future step) translates between Go interfaces and
WASM integer handles. This follows the existing pattern where `lib/bravo/wasm/`
provides the ABI layer separately from domain types.

## Package

`go/internal/charlie/filesystem_ops/` — NATO layer charlie, same as `fd` and
`store_workspace`. Importable by `store_fs` (mike) and `store_browser` (sierra).

## Horizontal Versioning

The interface follows dodder's horizontal versioning convention:

- `V0` is the initial interface, frozen once defined.
- Future evolution creates `V1`, `V2`, etc. as new interface types.
- `VCurrent` aliases the latest version.
- WASM modules declare which version they target by their import namespace.

## V0 Interface

```go
package filesystem_ops

import (
    "io"
    "io/fs"
)

type OpenMode uint8

const (
    OpenModeDefault   OpenMode = iota // shared read
    OpenModeExclusive                 // exclusive lock (for blob reads)
)

type CreateMode uint8

const (
    CreateModeTruncate CreateMode = iota // create or truncate
)

type V0 interface {
    Open(path string, mode OpenMode) (io.ReadCloser, error)
    Create(path string, mode CreateMode) (io.WriteCloser, error)
    CreateTemp(dir, pattern string) (path string, w io.WriteCloser, err error)
    Rename(oldpath, newpath string) error
    Remove(path string) error
    Lstat(path string) (fs.FileInfo, error)
    EvalSymlinks(path string) (string, error)
    WalkDir(root string, fn fs.WalkDirFunc) error
    Merge(base, current, other io.Reader) (io.ReadCloser, error)
}

type VCurrent = V0
```

## Method Mapping

| V0 method | store_fs OS call it replaces |
|-----------|------------------------------|
| `Open(_, OpenModeDefault)` | `files.Open()` |
| `Open(_, OpenModeExclusive)` | `files.OpenExclusiveReadOnly()` |
| `Create(_, CreateModeTruncate)` | `files.OpenFile(_, O_WRONLY\|O_CREATE\|O_TRUNC, 0o666)` |
| `CreateTemp` | `env_dir.FileTempWithTemplate()` |
| `Rename` | `os.Rename()` |
| `Remove` | `env_repo.Delete()` / `os.RemoveAll()` |
| `Lstat` | `os.Lstat()` |
| `EvalSymlinks` | `filepath.EvalSymlinks()` |
| `WalkDir` | `filepath.WalkDir()` |
| `Merge` | `exec.Command("git merge-file")` |

## What Stays in the Guest

Pure computation that needs no host call:

- `filepath.Rel()`, `filepath.Dir()`, `filepath.Clean()`, `filepath.Match()`
- `os.FileInfoToDirEntry()` (type conversion)
- `os.Environ()` (was only needed for `git merge-file`, now absorbed by `Merge`)

## Merge Design

The `Merge` host callback replaces the `git merge-file` subprocess. It accepts
three `io.Reader` streams (base, current, other) and returns a merged
`io.ReadCloser`. The host implementation can use `git merge-file` with temp
files, a streaming diff3 library, or any other strategy.

The guest never holds full blobs in memory. It passes stream handles; the host
manages the merge. The returned reader streams the result back in chunks.

## Statefulness and errors.Context

The concrete implementation (not the interface) takes `errors.Context` at
construction. Open file handles returned by `Open`, `Create`, and `CreateTemp`
are registered with `ctx.After()` as cleanup safety nets. The interface does not
embed `errors.Flusher`; cleanup is context-driven and internal to the
implementation.

```go
func Make(ctx errors.Context) V0 {
    impl := &v0{ctx: ctx}
    return impl
}
```

## What's NOT in V0

- `errors.Flusher` — cleanup is context-driven, not exposed on the interface
- Permissions parameter on `Create` — all V0 writes use `0o666`
- Append mode — not used by `store_fs` today
- Directory creation (`MkdirAll`) — `store_fs` delegates this to `env_repo`
- Blob store access — separate interface boundary

## Rollback Strategy

This is purely additive. The `V0` interface is a new package; no existing code
changes until a later step wires `store_fs` to use it. If the interface proves
wrong, delete the package with zero impact on existing code paths.

## Future Steps

1. **Concrete host implementation** — `OsFilesystemOps` wrapping `os.*` calls
2. **Wire store_fs** — replace direct OS calls with `V0` method calls
3. **WASM ABI adapter** — translate `V0` into integer-handle WASM host imports
4. **Compile store_fs to WASM** — `GOOS=wasip1 GOARCH=wasm`
5. **Repeat for store_browser** — likely requires its own interface or V1
   extension for network operations
