# OsFilesystemOps + store_fs Wiring Design

## Summary

Update the `FilesystemOps V0` interface with three new methods (`ReadDir`,
`Rel`, `GetCwd`), create a concrete `OsFilesystemOps` implementation wrapping
`os.*` calls, and wire `store_fs` to use `V0` for all filesystem operations
instead of direct OS calls.

## Motivation

`store_fs` currently mixes direct OS calls (`os.Rename`, `os.Lstat`,
`exec.Command`) with abstracted calls via `env_repo.Env`. This inconsistency
blocks WASM compilation. After this change, all filesystem operations go through
`V0`, leaving `env_repo.Env` in `store_fs` for domain-only operations (blob
store, config).

## Updated V0 Interface

Three methods added to the original 9:

```go
type V0 interface {
    Open(path string, mode OpenMode) (io.ReadCloser, error)
    Create(path string, mode CreateMode) (io.WriteCloser, error)
    CreateTemp(dir, pattern string) (path string, w io.WriteCloser, err error)
    ReadDir(path string) ([]fs.DirEntry, error)
    Rename(oldpath, newpath string) error
    Remove(path string) error
    GetCwd() string
    Rel(path string) (string, error)
    Lstat(path string) (fs.FileInfo, error)
    EvalSymlinks(path string) (string, error)
    WalkDir(root string, fn fs.WalkDirFunc) error
    Merge(base, current, other io.Reader) (io.ReadCloser, error)
}
```

| New method | Replaces | Usage in store_fs |
|-----------|----------|-------------------|
| `ReadDir` | `files.ReadDir` | delete_checkout.go — check if dir is empty |
| `Rel` | `envRepo.Rel` | dir_info.go, main.go — relative path from cwd |
| `GetCwd` | `envRepo.GetCwd` | checkout.go, merge.go, read_external.go — working directory |

## Concrete Implementation

`OsFilesystemOps` lives in the same package: `charlie/filesystem_ops/os.go`.
Only depends on stdlib + `errors.Context`, fits at charlie layer.

```go
type OsFilesystemOps struct {
    ctx errors.Context
    cwd string
}

func MakeOsFilesystemOps(ctx errors.Context, cwd string) *OsFilesystemOps {
    return &OsFilesystemOps{ctx: ctx, cwd: cwd}
}
```

Each method wraps the corresponding OS call:

| V0 method | OS call |
|-----------|---------|
| `Open(_, OpenModeDefault)` | `os.Open(path)` |
| `Open(_, OpenModeExclusive)` | `os.OpenFile(path, O_RDONLY\|O_EXCL, 0o666)` |
| `Create(_, CreateModeTruncate)` | `os.OpenFile(path, O_WRONLY\|O_CREATE\|O_TRUNC, 0o666)` |
| `CreateTemp` | `os.CreateTemp(dir, pattern)` |
| `ReadDir` | `os.ReadDir(path)` |
| `Rename` | `os.Rename(old, new)` |
| `Remove` | `os.RemoveAll(path)` |
| `GetCwd` | Returns stored `cwd` string |
| `Rel` | `filepath.Rel(o.cwd, path)` |
| `Lstat` | `os.Lstat(path)` |
| `EvalSymlinks` | `filepath.EvalSymlinks(path)` |
| `WalkDir` | `filepath.WalkDir(root, fn)` |
| `Merge` | `exec.Command("git", "merge-file", ...)` with temp files |

`Merge` creates three temp files from the input readers, runs `git merge-file
-p`, and returns the stdout as an `io.ReadCloser`. The temp file cleanup is
registered with `ctx.After()`.

## Wiring into store_fs

### Make() signature change

```go
func Make(
    config sku.Config,
    deletedPrinter interfaces.FuncIter[*fd.FD],
    fileExtensions file_extensions.Config,
    envRepo env_repo.Env,
    fsOps filesystem_ops.V0,  // new parameter
) (*Store, error)
```

### Store struct change

```go
type Store struct {
    // ...existing fields...
    fsOps filesystem_ops.V0
}
```

The `dirInfo` embedded struct and `fileEncoder` also receive `fsOps` (they
currently hold `envRepo` for filesystem calls).

### Call site: env_workspace

The one construction site (`env_workspace/main.go`) creates `OsFilesystemOps`
and passes it:

```go
fsOps := filesystem_ops.MakeOsFilesystemOps(ctx, envRepo.GetCwd())

outputEnv.storeFS, err = store_fs.Make(
    config,
    deletedPrinter,
    config.GetFileExtensions(),
    envRepo,
    fsOps,
)
```

## store_fs File Changes

| File | Direct OS call | Becomes |
|------|---------------|---------|
| primitives.go:112 | `files.Open(path)` | `store.fsOps.Open(path, OpenModeDefault)` |
| primitives.go:150 | `files.OpenExclusiveReadOnly(path)` | `store.fsOps.Open(path, OpenModeExclusive)` |
| file_encoder.go:53 | `files.OpenFile(path, mode, perm)` | `encoder.fsOps.Create(path, CreateModeTruncate)` |
| file_encoder.go:60 | `files.OpenExclusiveReadOnly(path)` | `encoder.fsOps.Open(path, OpenModeExclusive)` |
| checkout.go:253 | `store.envRepo.GetCwd()` | `store.fsOps.GetCwd()` |
| checkout.go:260 | `envRepo.GetTempLocal().FileTempWithTemplate(...)` | `store.fsOps.CreateTemp(dir, pattern)` |
| checkout.go:407,419 | `os.Rename(old, new)` | `store.fsOps.Rename(old, new)` |
| delete_checkout.go:78,112 | `s.Delete(path)` | `store.fsOps.Remove(path)` |
| delete_checkout.go:98 | `files.ReadDir(d)` | `store.fsOps.ReadDir(d)` |
| dir_info.go:75 | `filepath.WalkDir(root, fn)` | `dirInfo.fsOps.WalkDir(root, fn)` |
| dir_info.go:88 | `filepath.EvalSymlinks(path)` | `dirInfo.fsOps.EvalSymlinks(path)` |
| dir_info.go:97 | `os.Lstat(path)` | `dirInfo.fsOps.Lstat(path)` |
| dir_info.go:543 | `dirInfo.envRepo.Rel(path)` | `dirInfo.fsOps.Rel(path)` |
| main.go:439 | `store.envRepo.Rel(path)` | `store.fsOps.Rel(path)` |
| diff.go:22-76 | `exec.Command("git", "merge-file", ...)` | `store.fsOps.Merge(base, current, other)` |
| diff.go:51 | `envRepo.GetTempLocal().FileTemp()` | `store.fsOps.CreateTemp("", "merge-*")` |
| merge.go:340 | `store.envRepo.GetTempLocal().FileTemp()` | `store.fsOps.CreateTemp("", "")` |
| merge.go:386,401 | `store.envRepo.GetCwd()` | `store.fsOps.GetCwd()` |
| read_external.go:26 | `store.envRepo.GetCwd()` | `store.fsOps.GetCwd()` |

## What Stays on env_repo.Env

After wiring, `env_repo.Env` is used in store_fs for exactly:

- `GetDefaultBlobStore()` — blob read/write (5 sites)
- `GetConfigPrivate()` — config access (1 site)

These are domain/repo operations, not filesystem operations. They will need
their own WASM interface in a future step.

## Rollback Strategy

**Dual-architecture period**: `OsFilesystemOps` wraps the same `os.*` calls that
store_fs uses today. Behavior is identical — the abstraction is a refactor, not a
behavior change.

**Rollback procedure**: `git revert` the wiring commits. The interface package
and concrete implementation remain harmless.

**Promotion criteria**: All existing tests pass (`just test`). The old direct-call
path is fully superseded once the WASM adapter is built and store_fs compiles to
WASM.
