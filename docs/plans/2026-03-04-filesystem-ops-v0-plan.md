# FilesystemOps V0 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create the `FilesystemOpsV0` interface package at `go/internal/charlie/filesystem_ops/`.

**Architecture:** A single package defining the `V0` interface, two mode types (`OpenMode`, `CreateMode`), and a `VCurrent` alias. No concrete implementation yet — this is the interface contract only.

**Tech Stack:** Go, standard library `io` and `io/fs` packages.

**Rollback:** Purely additive — delete the package directory to revert.

---

### Task 1: Create the filesystem_ops package with mode types

**Promotion criteria:** N/A — new package.

**Files:**
- Create: `go/internal/charlie/filesystem_ops/main.go`

**Step 1: Create the package file with mode types and V0 interface**

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

**Step 2: Verify the package compiles**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/plain-olive && just build`

Expected: Clean build, no errors.

**Step 3: Commit**

```bash
git add go/internal/charlie/filesystem_ops/main.go
git commit -m "feat: add FilesystemOps V0 interface for WASM workspace modules"
```
