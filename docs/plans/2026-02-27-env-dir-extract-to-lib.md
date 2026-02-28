# Extract TemporaryFS and Hash Bucket Utils to lib/delta/files Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract `TemporaryFS` and hash bucket path utilities from
`internal/india/env_dir` into `lib/delta/files`, then re-export type aliases
from `env_dir` so callers migrate transparently.

**Architecture:** Move the generic types and functions into `lib/delta/files`
(which already exists as a file utility grab-bag). Leave the dodder-specific
`MakeHashBucketPathFromMerkleId()` wrapper in `env_dir` — it calls the generic
`MakeHashBucketPath()` from its new home. After moving, add type/function
aliases in `env_dir` so downstream callers don't need to change their imports
yet. A separate follow-up can migrate callers directly if desired.

**Tech Stack:** Go, NATO tier hierarchy (`lib/delta` depends on `lib/alfa` and
below)

---

## Dependency Analysis

### Functions moving to `lib/delta/files`

| Function/Type | Current imports needed | lib-eligible? |
|---------------|----------------------|---------------|
| `TemporaryFS` struct + 4 methods | `os`, `lib/bravo/errors` | Yes |
| `MakeHashBucketPath()` | `bytes`, `fmt`, `path/filepath`, `strings`, `lib/alfa/unicorn` | Yes |
| `PathFromHeadAndTail()` | `path/filepath`, `lib/_/interfaces` | Yes |
| `MakeHashBucketPathJoinFunc()` | (calls `MakeHashBucketPath`) | Yes |
| `MakeDirIfNecessary()` | `os`, `path/filepath`, `lib/bravo/errors` | Yes |
| `MakeDirIfNecessaryForStringerWithHeadAndTail()` | (calls `PathFromHeadAndTail`) | Yes |

### Function staying in `internal/india/env_dir`

| Function | Why |
|----------|-----|
| `MakeHashBucketPathFromMerkleId()` | Imports `internal/alfa/domain_interfaces`, `internal/foxtrot/markl` |
| `resetTempOnExit()` | Private method on `env`, uses `env.debugOptions` |

### New imports for `lib/delta/files`

| Import | Tier | Already imported? |
|--------|------|-------------------|
| `lib/alfa/unicorn` | alfa | No — new |
| `lib/_/interfaces` | _ | Yes (in `dirnames_iter.go`) |
| `lib/bravo/errors` | bravo | Yes |

All within delta's tier constraints.

---

### Task 1: Add `TemporaryFS` to `lib/delta/files`

**Files:**
- Create: `go/lib/delta/files/temp.go`

**Step 1: Create `temp.go` with the `TemporaryFS` type**

```go
package files

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type TemporaryFS struct {
	BasePath string
}

func (fs TemporaryFS) DirTemp() (d string, err error) {
	return fs.DirTempWithTemplate("")
}

func (fs TemporaryFS) DirTempWithTemplate(
	template string,
) (dir string, err error) {
	if dir, err = os.MkdirTemp(fs.BasePath, template); err != nil {
		err = errors.Wrap(err)
		return dir, err
	}

	return dir, err
}

func (fs TemporaryFS) FileTemp() (file *os.File, err error) {
	if file, err = fs.FileTempWithTemplate(""); err != nil {
		err = errors.Wrap(err)
		return file, err
	}

	return file, err
}

func (fs TemporaryFS) FileTempWithTemplate(
	template string,
) (file *os.File, err error) {
	if file, err = os.CreateTemp(fs.BasePath, template); err != nil {
		err = errors.Wrap(err)
		return file, err
	}

	return file, err
}
```

**Step 2: Build to verify**

Run: `just build` from `go/`
Expected: Succeeds

**Step 3: Commit**

```
feat(files): add TemporaryFS type for temp dir/file creation
```

---

### Task 2: Add hash bucket path utilities to `lib/delta/files`

**Files:**
- Create: `go/lib/delta/files/hash_bucket.go`

**Step 1: Create `hash_bucket.go` with the generic functions**

```go
package files

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/unicorn"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeHashBucketPath(
	hashBytes []byte,
	buckets []int,
	pathComponents ...string,
) string {
	var buffer bytes.Buffer

	for _, pathComponent := range pathComponents {
		pathComponent = strings.TrimRight(
			pathComponent,
			string(filepath.Separator),
		)

		buffer.WriteString(pathComponent)
		buffer.WriteByte(filepath.Separator)
	}

	remaining := hashBytes

	for _, bucket := range buckets {
		if len(remaining) < bucket {
			panic(
				fmt.Sprintf(
					"buckets too large for string. buckets: %v, string: %q, remaining: %q",
					buckets,
					hashBytes,
					remaining,
				),
			)
		}

		var added []byte

		added, remaining = unicorn.CutNCharacters(remaining, bucket)

		buffer.Write(added)
		buffer.WriteByte(filepath.Separator)
	}

	if len(remaining) > 0 {
		buffer.Write(remaining)
	}

	return buffer.String()
}

func PathFromHeadAndTail(
	stringer interfaces.StringerWithHeadAndTail,
	pathComponents ...string,
) string {
	pathComponents = append(
		pathComponents,
		stringer.GetHead(),
		stringer.GetTail(),
	)

	return filepath.Join(pathComponents...)
}

func MakeHashBucketPathJoinFunc(
	buckets []int,
) func(string, ...string) string {
	return func(initial string, pathComponents ...string) string {
		return MakeHashBucketPath(
			[]byte(initial),
			buckets,
			pathComponents...,
		)
	}
}

func MakeDirIfNecessary(
	base string,
	joinFunc func(string, ...string) string,
	pathComponents ...string,
) (path string, err error) {
	path = joinFunc(base, pathComponents...)
	dir := filepath.Dir(path)

	if err = os.MkdirAll(dir, os.ModeDir|0o755); err != nil {
		err = errors.Wrap(err)
		return path, err
	}

	return path, err
}

func MakeDirIfNecessaryForStringerWithHeadAndTail(
	stringer interfaces.StringerWithHeadAndTail,
	pathComponents ...string,
) (path string, err error) {
	path = PathFromHeadAndTail(stringer, pathComponents...)
	dir := filepath.Dir(path)

	if err = os.MkdirAll(dir, os.ModeDir|0o755); err != nil {
		err = errors.Wrap(err)
		return path, err
	}

	return path, err
}
```

**Step 2: Build to verify**

Run: `just build` from `go/`
Expected: Succeeds

**Step 3: Commit**

```
feat(files): add hash bucket path utilities
```

---

### Task 3: Replace `env_dir` implementations with aliases to `lib/delta/files`

**Files:**
- Modify: `go/internal/india/env_dir/temp.go`
- Modify: `go/internal/india/env_dir/util.go`

**Step 1: Replace `TemporaryFS` in `temp.go` with a type alias**

Replace the entire `TemporaryFS` definition and its 4 methods in
`go/internal/india/env_dir/temp.go` with a type alias. Keep
`resetTempOnExit()` since it's a private method on `env`.

The file should become:

```go
package env_dir

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

// TODO only call reset temp when actually not resetting temp
func (env env) resetTempOnExit(ctx interfaces.ActiveContext) (err error) {
	errIn := ctx.Cause()

	if errIn != nil || env.debugOptions.NoTempDirCleanup {
		// ui.Err().Printf("temp dir: %q", s.TempLocal.BasePath)
	} else {
		if err = os.RemoveAll(env.GetTempLocal().BasePath); err != nil {
			err = errors.Wrapf(err, "failed to remove temp local")
			return err
		}
	}

	return err
}

type TemporaryFS = files.TemporaryFS
```

**Step 2: Replace generic functions in `util.go` with delegation**

In `go/internal/india/env_dir/util.go`, replace the generic function bodies
with calls to `files.*`. Keep `MakeHashBucketPathFromMerkleId()` as-is since
it has `internal/` dependencies.

The file should become:

```go
package env_dir

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func MakeHashBucketPathFromMerkleId(
	id domain_interfaces.MarklId,
	buckets []int,
	multiHash bool,
	pathComponents ...string,
) string {
	if multiHash {
		pathComponents = append(
			pathComponents,
			id.GetMarklFormat().GetMarklFormatId(),
		)
	}

	return files.MakeHashBucketPath(
		[]byte(markl.FormatBytesAsHex(id)),
		buckets,
		pathComponents...,
	)
}

var MakeHashBucketPath = files.MakeHashBucketPath

var PathFromHeadAndTail = files.PathFromHeadAndTail

var MakeHashBucketPathJoinFunc = files.MakeHashBucketPathJoinFunc

var MakeDirIfNecessary = files.MakeDirIfNecessary

var MakeDirIfNecessaryForStringerWithHeadAndTail = files.MakeDirIfNecessaryForStringerWithHeadAndTail
```

Note: Using package-level `var` assignments for function aliases preserves the
existing call sites (`env_dir.MakeHashBucketPath(...)` continues to work). The
removed imports (`bytes`, `fmt`, `os`, `path/filepath`, `strings`,
`lib/alfa/unicorn`, `lib/bravo/errors`) are no longer needed — check that
`MakeHashBucketPathFromMerkleId` doesn't require any of them directly.

**Step 3: Build to verify all callers still compile**

Run: `just build` from `go/`
Expected: Succeeds

**Step 4: Run unit tests**

Run: `just test-go` from `go/`
Expected: All pass

**Step 5: Commit**

```
refactor(env_dir): delegate TemporaryFS and hash bucket utils to lib/delta/files
```

---

### Task 4: Update CLAUDE.md files

**Files:**
- Modify: `go/lib/delta/files/CLAUDE.md`

**Step 1: Update `lib/delta/files/CLAUDE.md`**

Add sections for the new types/functions. After "Timeout-based retry logic" in
the Features list, add:

```
- Temporary file/directory creation via TemporaryFS
- Hash bucket path generation (Git-style bucketed paths)
```

And add a new section:

```
## Temp FS

- `TemporaryFS`: Struct wrapping a base path for temp file/dir creation
  - `DirTemp()`, `DirTempWithTemplate()`: Create temp directories
  - `FileTemp()`, `FileTempWithTemplate()`: Create temp files

## Hash Bucket Paths

- `MakeHashBucketPath()`: Build paths with hash-based directory bucketing
- `MakeHashBucketPathJoinFunc()`: Curried version returning a join function
- `PathFromHeadAndTail()`: Build paths from head/tail string interface
- `MakeDirIfNecessary()`: Create parent directories and return joined path
- `MakeDirIfNecessaryForStringerWithHeadAndTail()`: Variant using head/tail
```

**Step 2: Commit**

```
docs: update files CLAUDE.md with TemporaryFS and hash bucket docs
```

---

### Task 5: Run full test suite

**Step 1: Build debug binaries**

Run: `just build` from `go/`

**Step 2: Run full test suite**

Run: `just test` from `go/`
Expected: All pass (unit + integration)

**Step 3: If tests fail, investigate and fix**

This is a pure refactor — all external call sites see identical signatures via
the aliases. Failures would indicate a type compatibility issue with the alias
approach.

---

## Summary of Changes

| What | Before | After |
|------|--------|-------|
| `TemporaryFS` | Defined in `internal/india/env_dir/temp.go` | Defined in `lib/delta/files/temp.go`, aliased in `env_dir` |
| `MakeHashBucketPath` | Defined in `env_dir/util.go` | Defined in `lib/delta/files/hash_bucket.go`, var alias in `env_dir` |
| `PathFromHeadAndTail` | Defined in `env_dir/util.go` | Defined in `lib/delta/files/hash_bucket.go`, var alias in `env_dir` |
| `MakeHashBucketPathJoinFunc` | Defined in `env_dir/util.go` | Defined in `lib/delta/files/hash_bucket.go`, var alias in `env_dir` |
| `MakeDirIfNecessary` | Defined in `env_dir/util.go` | Defined in `lib/delta/files/hash_bucket.go`, var alias in `env_dir` |
| `MakeDirIfNecessaryForStringerWithHeadAndTail` | Defined in `env_dir/util.go` | Defined in `lib/delta/files/hash_bucket.go`, var alias in `env_dir` |
| `MakeHashBucketPathFromMerkleId` | Calls local `MakeHashBucketPath` | Calls `files.MakeHashBucketPath` |
| Downstream callers | Import `env_dir` | No change needed (aliases preserve API) |
