# OsFilesystemOps + store_fs Wiring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire `store_fs` to use `filesystem_ops.V0` for all filesystem operations, replacing direct OS calls.

**Architecture:** Update the V0 interface with three new methods, create a concrete `OsFilesystemOps` in the same package, inject it into `store_fs.Make()`, and replace all direct `os.*`/`files.*`/`exec.*` calls across 8 store_fs files.

**Tech Stack:** Go, stdlib `os`, `io/fs`, `os/exec`, `filepath`.

**Rollback:** `git revert` the wiring commits. The interface package remains harmless.

---

### Task 1: Update V0 interface with new methods

**Promotion criteria:** N/A — additive change.

**Files:**
- Modify: `go/internal/charlie/filesystem_ops/main.go`

**Step 1: Add ReadDir, Rel, GetCwd to V0 interface**

Update the file to:

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

type VCurrent = V0
```

**Step 2: Verify it compiles**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive/go && go build ./internal/charlie/filesystem_ops/`

**Step 3: Commit**

```
feat: add ReadDir, Rel, GetCwd to FilesystemOps V0
```

---

### Task 2: Create OsFilesystemOps concrete implementation

**Promotion criteria:** N/A — new file.

**Files:**
- Create: `go/internal/charlie/filesystem_ops/os.go`

**Step 1: Implement OsFilesystemOps**

Create `os.go` with a struct implementing `V0`. Each method wraps the corresponding stdlib call:

- `Open(_, OpenModeDefault)` → `os.Open(path)`
- `Open(_, OpenModeExclusive)` → `os.OpenFile(path, os.O_RDONLY|os.O_EXCL, 0o666)`
- `Create(_, CreateModeTruncate)` → `os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)`
- `CreateTemp` → `os.CreateTemp(dir, pattern)`; return `file.Name()` and file as `io.WriteCloser`
- `ReadDir` → `os.ReadDir(path)`
- `Rename` → `os.Rename(oldpath, newpath)`
- `Remove` → `os.RemoveAll(path)`
- `GetCwd` → return stored `cwd` field
- `Rel` → `filepath.Rel(o.cwd, path)`
- `Lstat` → `os.Lstat(path)`
- `EvalSymlinks` → `filepath.EvalSymlinks(path)`
- `WalkDir` → `filepath.WalkDir(root, fn)`

For `Merge`:
1. Write base, current, other readers to three temp files
2. Run `exec.Command("git", "merge-file", "-p", ...)` with those paths
3. Pipe stdout to a fourth temp file
4. If exit code is non-zero `exec.ExitError`, return the reader AND an error (caller checks for conflict)
5. Return an `io.ReadCloser` wrapping the output temp file

The constructor:

```go
func MakeOsFilesystemOps(cwd string) *OsFilesystemOps {
	return &OsFilesystemOps{cwd: cwd}
}
```

Note: `errors.Context` integration is deferred — the constructor does not take
a context yet. Context-driven cleanup will be added when the WASM adapter layer
is built and handle lifetimes become a concern.

**Step 2: Verify it compiles**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive/go && go build ./internal/charlie/filesystem_ops/`

**Step 3: Commit**

```
feat: add OsFilesystemOps concrete implementation
```

---

### Task 3: Wire fsOps into store_fs structs and constructors

**Promotion criteria:** N/A — additive wiring.

**Files:**
- Modify: `go/internal/mike/store_fs/main.go` — Store struct, Make()
- Modify: `go/internal/mike/store_fs/dir_info.go` — dirInfo struct, makeObjectsWithDir()
- Modify: `go/internal/mike/store_fs/file_encoder.go` — fileEncoder struct, MakeFileEncoder()
- Modify: `go/internal/november/env_workspace/main.go` — store_fs.Make() call site

**Step 1: Add fsOps field to Store, dirInfo, fileEncoder**

In `main.go`, add to Store struct:
```go
fsOps filesystem_ops.V0
```

Update `Make()` signature to accept `fsOps filesystem_ops.V0` as a new
parameter after `envRepo`. Store it in the struct. Pass it to
`makeObjectsWithDir()` and `MakeFileEncoder()`.

In `dir_info.go`, add `fsOps filesystem_ops.V0` field to `dirInfo` struct (line
31). Update `makeObjectsWithDir()` to accept and store it.

In `file_encoder.go`, add `fsOps filesystem_ops.V0` field to `fileEncoder`
struct (line 25). Update `MakeFileEncoder()` to accept and store it.

**Step 2: Update env_workspace call site**

In `env_workspace/main.go` (line 129), update the `store_fs.Make()` call to
create an `OsFilesystemOps` and pass it:

```go
fsOps := filesystem_ops.MakeOsFilesystemOps(envRepo.GetCwd())

if outputEnv.storeFS, err = store_fs.Make(
	config,
	deletedPrinter,
	config.GetFileExtensions(),
	envRepo,
	fsOps,
); err != nil {
```

**Step 3: Verify it compiles and tests pass**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

**Step 4: Commit**

```
refactor: wire FilesystemOps V0 into store_fs constructors
```

---

### Task 4: Replace OS calls in dir_info.go

**Promotion criteria:** When no direct `os.*`/`filepath.*` filesystem calls
remain in dir_info.go.

**Files:**
- Modify: `go/internal/mike/store_fs/dir_info.go`

**Step 1: Replace filesystem calls**

Four replacements:

1. Line 75: `filepath.WalkDir(dir, ...)` → `dirInfo.fsOps.WalkDir(dir, ...)`
2. Line 88: `filepath.EvalSymlinks(path)` → `dirInfo.fsOps.EvalSymlinks(path)`
3. Line 97: `os.Lstat(path)` → `dirInfo.fsOps.Lstat(path)`
4. Line 543: `dirInfo.envRepo.Rel(fdee.GetPath())` → `dirInfo.fsOps.Rel(fdee.GetPath())`

Remove unused `os` import if it becomes unused. Keep `filepath` if still used
for `filepath.Match`, `filepath.Base`, `filepath.SkipDir` (these are pure
computation, not OS calls).

**Step 2: Verify tests pass**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

**Step 3: Commit**

```
refactor: replace direct OS calls with fsOps in dir_info.go
```

---

### Task 5: Replace OS calls in primitives.go and file_encoder.go

**Promotion criteria:** When no direct `files.Open`/`files.OpenFile`/
`files.OpenExclusiveReadOnly` calls remain in these files.

**Files:**
- Modify: `go/internal/mike/store_fs/primitives.go`
- Modify: `go/internal/mike/store_fs/file_encoder.go`

**Step 1: Replace calls in primitives.go**

1. Line 112: `files.Open(item.Object.GetPath())` →
   `store.fsOps.Open(item.Object.GetPath(), filesystem_ops.OpenModeDefault)`
   Change `var f *os.File` to `var f io.ReadCloser`.

2. Line 150: `files.OpenExclusiveReadOnly(item.Blob.GetPath())` →
   `store.fsOps.Open(item.Blob.GetPath(), filesystem_ops.OpenModeExclusive)`
   Change `var file *os.File` to `var file io.ReadCloser`.

**Step 2: Replace calls in file_encoder.go**

The `openOrCreate` method (line 52) currently returns `*os.File`. It needs to
return `io.ReadWriteCloser` or be refactored.

Replace:
1. Line 53: `files.OpenFile(path, encoder.mode, encoder.perm)` →
   `encoder.fsOps.Create(path, filesystem_ops.CreateModeTruncate)`
2. Line 60: `files.OpenExclusiveReadOnly(path)` →
   `encoder.fsOps.Open(path, filesystem_ops.OpenModeExclusive)`

The return type of `openOrCreate` changes from `*os.File` to `io.WriteCloser`
(or split into two methods if the fallback-to-read pattern needs both). Check
all callers of `openOrCreate` and update accordingly.

Remove the `mode` and `perm` fields from `fileEncoder` struct — they're no
longer needed since `CreateModeTruncate` encapsulates them.

Remove unused `files` and `os` imports.

**Step 3: Verify tests pass**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

**Step 4: Commit**

```
refactor: replace direct OS calls with fsOps in primitives.go and file_encoder.go
```

---

### Task 6: Replace OS calls in checkout.go

**Promotion criteria:** When no direct `os.*`/`envRepo.GetCwd()`/
`envRepo.GetTempLocal()` calls remain in checkout.go.

**Files:**
- Modify: `go/internal/mike/store_fs/checkout.go`

**Step 1: Replace filesystem calls**

1. Line 253: `store.envRepo.GetCwd()` → `store.fsOps.GetCwd()`

2. Line 260: `store.envRepo.GetTempLocal().FileTempWithTemplate(pattern)` →
   `store.fsOps.CreateTemp("", pattern)`. The return type changes from
   `*os.File` to `(path string, w io.WriteCloser)`. Update the caller to use
   the returned path instead of `file.Name()`.

3. Lines 407, 419: `os.Rename(new, old)` → `store.fsOps.Rename(new, old)`

Remove unused `os` import.

**Step 2: Verify tests pass**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

**Step 3: Commit**

```
refactor: replace direct OS calls with fsOps in checkout.go
```

---

### Task 7: Replace OS calls in delete_checkout.go

**Promotion criteria:** When no direct `s.Delete()`/`files.ReadDir()`/
`s.GetCwd()` calls remain in delete_checkout.go.

**Files:**
- Modify: `go/internal/mike/store_fs/delete_checkout.go`

**Step 1: Update DeleteCheckout.Run signature**

The current signature takes `s env_repo.Env` for filesystem operations. All
uses of `s` in this function are filesystem ops (Delete, GetCwd). Change the
parameter to `fsOps filesystem_ops.V0`:

```go
func (c DeleteCheckout) Run(
	dryRun bool,
	fsOps filesystem_ops.V0,
	p interfaces.FuncIter[*fd.FD],
	fs interfaces.Collection[*fd.FD],
) (err error) {
```

**Step 2: Replace calls**

1. Lines 77, 109: `s.Delete(path)` → `fsOps.Remove(path)`
2. Line 96: `files.ReadDir(d)` → `fsOps.ReadDir(d)`
3. Lines 50, 57, 64: `s.GetCwd()` → `fsOps.GetCwd()`
4. Line 50: `filepath.Rel(s.GetCwd(), fd.String())` → `fsOps.Rel(fd.String())`

Remove `env_repo` and `files` imports.

**Step 3: Update all callers of DeleteCheckout.Run**

Search for `DeleteCheckout` usage across the codebase and update callers to pass
`store.fsOps` instead of `env_repo.Env`.

**Step 4: Verify tests pass**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

**Step 5: Commit**

```
refactor: replace direct OS calls with fsOps in delete_checkout.go
```

---

### Task 8: Rewrite diff.go to use Merge

**Promotion criteria:** When `exec.Command` and `os.Environ` are no longer
called from diff.go.

**Files:**
- Modify: `go/internal/mike/store_fs/diff.go`

**Step 1: Rewrite runDiff3**

The current `runDiff3` creates an `exec.Command("git", "merge-file", ...)`,
pipes stdout to a temp file, and checks exit codes. Replace with:

1. Open the three input files as `io.Reader`s using `store.fsOps.Open(...,
   OpenModeDefault)`:
   - `local.Object.GetPath()` for local
   - `base.Object.GetPath()` for base (or an empty reader if base has no FDs)
   - `remote.Object.GetPath()` for remote

2. Call `store.fsOps.Merge(baseReader, localReader, remoteReader)` to get the
   merged `io.ReadCloser` and potential error.

3. Write the merged output to a temp file using `store.fsOps.CreateTemp("",
   "merge-*")`.

4. Copy from merged reader to the temp writer.

5. Build the result FSItem pointing to the temp file path.

6. If Merge returned a conflict error, wrap it in
   `sku.MakeErrMergeConflict(merged)`.

Remove `os`, `os/exec` imports entirely.

**Step 2: Verify tests pass**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

**Step 3: Commit**

```
refactor: replace exec.Command with fsOps.Merge in diff.go
```

---

### Task 9: Replace remaining OS calls in merge.go, read_external.go, main.go

**Promotion criteria:** When store_fs has zero direct `os.*` syscalls and zero
`envRepo.GetCwd()`/`envRepo.Rel()`/`envRepo.GetTempLocal()` calls.

**Files:**
- Modify: `go/internal/mike/store_fs/merge.go`
- Modify: `go/internal/mike/store_fs/read_external.go`
- Modify: `go/internal/mike/store_fs/main.go`

**Step 1: Replace calls in merge.go**

1. Line 340: `store.envRepo.GetTempLocal().FileTemp()` →
   `store.fsOps.CreateTemp("", "")`. Update `var file *os.File` to use the
   returned `(path, writer)`.
2. Lines 386, 401: `store.envRepo.GetCwd()` → `store.fsOps.GetCwd()`

Note: line 358 (`store.envRepo.GetConfigPrivate().Blob`) stays — that's a
domain/config access, not a filesystem op.

**Step 2: Replace calls in read_external.go**

1. Line 26: `store.envRepo.GetCwd()` → `store.fsOps.GetCwd()`

**Step 3: Replace calls in main.go**

1. Line 45: `envRepo.GetCwd()` in `Make()` → `fsOps.GetCwd()` (the `dir`
   field initialization)
2. Line 439: `store.envRepo.Rel(before)` → `store.fsOps.Rel(before)`

**Step 4: Verify tests pass**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

**Step 5: Commit**

```
refactor: replace remaining OS calls with fsOps in store_fs
```

---

### Task 10: Run full test suite and verify

**Files:** None — verification only.

**Step 1: Run unit tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-go`

Expected: All unit tests pass.

**Step 2: Build binaries**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just build`

Expected: Clean build, debug and release binaries produced.

**Step 3: Run integration tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just test-bats`

Expected: All BATS integration tests pass. The OsFilesystemOps wraps the same
OS calls, so behavior should be identical.

**Step 4: Verify no remaining direct OS calls in store_fs**

Search for any remaining `os.`, `files.`, `exec.`, `filepath.WalkDir`,
`filepath.EvalSymlinks` calls in the store_fs package that should have been
replaced. Pure-computation `filepath.*` calls (`Match`, `Base`, `Dir`, `Rel`,
`Clean`, `SkipDir`, `Join`) are expected to remain.

**Step 5: Commit verification note (if fixtures changed)**

If integration tests required fixture regeneration
(`just test-bats-update-fixtures`), review the diff and commit the updated
fixtures.
