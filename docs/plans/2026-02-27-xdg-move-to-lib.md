# Move xdg Package to lib/echo Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move `internal/echo/xdg` to `lib/echo/xdg` by removing the
`blob_store_id` dependency and making the dodder-specific env var name
configurable.

**Architecture:** The `xdg` package is a generic XDG Base Directory
Specification implementation parameterized by utility name. Its only
`internal/` dependency is `blob_store_id.LocationType`, used by a single method
`GetLocationType()`. We replace it with an `IsOverridden() bool` accessor and
let callers in `internal/` map that to their own location type. We also make the
`DODDER_XDG_UTILITY_OVERRIDE` env var name a configurable field on `InitArgs`.

**Tech Stack:** Go, NATO tier hierarchy (lib/echo depends on lib/delta and
below)

---

## Pre-work: Dependency Analysis

Current `internal/echo/xdg` imports:

| Import | Tree | Status |
|--------|------|--------|
| `lib/_/interfaces` | lib | OK |
| `lib/alfa/pool` | lib | OK |
| `lib/bravo/errors` | lib | OK |
| `lib/charlie/env_vars` | lib | OK |
| `lib/delta/files` | lib | OK |
| `lib/delta/xdg_defaults` | lib | OK |
| `internal/charlie/blob_store_id` | internal | **Must remove** |

After this plan, the package will be at `lib/echo/xdg` and import only `lib/`
packages.

---

### Task 1: Add `IsOverridden()` method to `xdg.XDG`

**Files:**
- Modify: `go/internal/echo/xdg/main.go`

**Step 1: Add `IsOverridden()` alongside `GetLocationType()`**

In `go/internal/echo/xdg/main.go`, add this method directly after
`GetLocationType()` (after line 125):

```go
func (xdg XDG) IsOverridden() bool {
	return xdg.overridePath != ""
}
```

**Step 2: Build to verify no errors**

Run: `just build` from `go/`
Expected: Succeeds

**Step 3: Commit**

```
feat(xdg): add IsOverridden accessor
```

---

### Task 2: Migrate callers from `GetLocationType()` to `IsOverridden()`

There are 2 direct callers of `xdg.XDG.GetLocationType()`:

1. `internal/foxtrot/directory_layout/v3.go:143` — delegates to
   `layout.xdg.GetLocationType()`
2. `internal/juliett/blob_stores/main.go:84` — checks
   `envDir.GetXDG().GetLocationType() == blob_store_id.LocationTypeCwd`

Both compare against `LocationTypeCwd`, which is equivalent to
`IsOverridden() == true`.

**Files:**
- Modify: `go/internal/foxtrot/directory_layout/v3.go:143-145`
- Modify: `go/internal/foxtrot/directory_layout/main.go:14-17`
- Modify: `go/internal/juliett/blob_stores/main.go:84`

**Step 1: Update `directory_layout` — change `v3.GetLocationType()`**

The `Common` interface currently embeds `blob_store_id.LocationTypeGetter`.
Replace with a new method that returns `blob_store_id.LocationType` by deriving
it from `IsOverridden()`.

In `go/internal/foxtrot/directory_layout/v3.go`, change `GetLocationType()`:

```go
func (layout v3) GetLocationType() blob_store_id.LocationType {
	if layout.xdg.IsOverridden() {
		return blob_store_id.LocationTypeCwd
	}

	return blob_store_id.LocationTypeXDGUser
}
```

Note: `directory_layout` already imports `blob_store_id`, so this adds no new
dependency.

**Step 2: Update `blob_stores/main.go` — use `IsOverridden()` directly**

In `go/internal/juliett/blob_stores/main.go`, change line 84 from:

```go
if envDir.GetXDG().GetLocationType() == blob_store_id.LocationTypeCwd {
```

to:

```go
if envDir.GetXDG().IsOverridden() {
```

Check whether `blob_store_id` is still imported elsewhere in the file. If this
was the only usage of `blob_store_id` in this file, remove the import.

**Step 3: Build to verify**

Run: `just build` from `go/`
Expected: Succeeds

**Step 4: Run unit tests**

Run: `just test-go` from `go/`
Expected: All pass

**Step 5: Commit**

```
refactor(directory_layout, blob_stores): derive location type from IsOverridden
```

---

### Task 3: Remove `GetLocationType()` from `xdg.XDG`

**Files:**
- Modify: `go/internal/echo/xdg/main.go`

**Step 1: Delete `GetLocationType()` and the `blob_store_id` import**

In `go/internal/echo/xdg/main.go`, delete:

```go
func (xdg XDG) GetLocationType() blob_store_id.LocationType {
	if xdg.overridePath == "" {
		return blob_store_id.LocationTypeXDGUser
	} else {
		return blob_store_id.LocationTypeCwd
	}
}
```

And remove the `blob_store_id` import line:

```go
"code.linenisgreat.com/dodder/go/internal/charlie/blob_store_id"
```

**Step 2: Build to verify no remaining callers**

Run: `just build` from `go/`
Expected: Succeeds — no callers of `GetLocationType()` remain on `xdg.XDG`

**Step 3: Commit**

```
refactor(xdg): remove GetLocationType and blob_store_id dependency
```

---

### Task 4: Make `DODDER_XDG_UTILITY_OVERRIDE` env var configurable

**Files:**
- Modify: `go/internal/echo/xdg/init_args.go`

**Step 1: Add `OverrideEnvVarName` field to `InitArgs`, remove the constant**

In `go/internal/echo/xdg/init_args.go`, replace:

```go
const EnvXDGUtilityNameOverride = "DODDER_XDG_UTILITY_OVERRIDE"
```

with the field on `InitArgs`:

```go
type InitArgs struct {
	Home        string
	Cwd         string
	UtilityName string
	ExecPath    string
	Pid         int

	OverrideEnvVarName string
}
```

**Step 2: Update `Initialize()` to use the field**

Change the env var lookup in `Initialize()` from:

```go
if utilityNameOverride := os.Getenv(EnvXDGUtilityNameOverride); utilityNameOverride != "" {
```

to:

```go
if initArgs.OverrideEnvVarName != "" {
	if utilityNameOverride := os.Getenv(initArgs.OverrideEnvVarName); utilityNameOverride != "" {
		utilityName = utilityNameOverride
	}
}
```

**Step 3: Update callers to set `OverrideEnvVarName`**

Search for all sites that construct `xdg.InitArgs{}` or reference
`xdg.EnvXDGUtilityNameOverride`. The `InitArgs` is constructed by `env_dir`
(`before_xdg.go` calls `initArgs.Initialize()`) — confirm that no caller
passes `OverrideEnvVarName` yet. The default zero value `""` means no override,
which changes behavior: callers that relied on the hardcoded env var must now
set the field.

Find the call site in `go/internal/india/env_dir/before_xdg.go` (line 24) and
update to set the field before calling `Initialize()`:

```go
func (env *beforeXDG) initialize(
	debugOptions debug.Options,
	utilityName string,
) (err error) {
	env.debugOptions = debugOptions
	env.xdgInitArgs.OverrideEnvVarName = "DODDER_XDG_UTILITY_OVERRIDE"

	if err = env.xdgInitArgs.Initialize(utilityName); err != nil {
```

Also check `CloneWithUtilityName`, `CloneWithOverridePath`,
`CloneWithoutOverride` in `main.go` — these construct `InitArgs` inline. They
don't call `Initialize()`, so they don't need the field.

**Step 4: Build and test**

Run: `just build && just test-go` from `go/`
Expected: All pass

**Step 5: Commit**

```
refactor(xdg): make override env var name configurable on InitArgs
```

---

### Task 5: Move `xdg` from `internal/echo` to `lib/echo`

**Files:**
- Move: `go/internal/echo/xdg/` → `go/lib/echo/xdg/`
- Modify: All importers (update import paths)

**Step 1: Verify xdg has no remaining `internal/` imports**

Run from `go/`:

```bash
grep -r 'internal/' internal/echo/xdg/*.go
```

Expected: No matches

**Step 2: Move the directory**

```bash
git mv go/internal/echo/xdg go/lib/echo/xdg
```

**Step 3: Update all import paths**

Replace all occurrences across the codebase:

```
Old: code.linenisgreat.com/dodder/go/internal/echo/xdg
New: code.linenisgreat.com/dodder/go/lib/echo/xdg
```

Files that import this package (from exploration):
- `go/internal/india/env_dir/main.go`
- `go/internal/india/env_dir/construction.go`
- `go/internal/india/env_dir/before_xdg.go`
- `go/internal/hotel/repo_blobs/toml_xdg_v0.go`
- `go/internal/hotel/repo_blobs/main.go`
- `go/internal/foxtrot/directory_layout/main.go`
- `go/internal/zulu/commands_dodder/info_repo.go`
- `go/internal/mike/commands_madder/info_repo.go`

Use sed or editor find-replace. Verify no stale references remain:

```bash
grep -r 'internal/echo/xdg' go/
```

Expected: No matches

**Step 4: Update the `XDG` type alias in `directory_layout`**

In `go/internal/foxtrot/directory_layout/main.go`, confirm the import now reads:

```go
"code.linenisgreat.com/dodder/go/lib/echo/xdg"
```

The type alias `XDG = xdg.XDG` continues to work unchanged.

**Step 5: Build and test**

Run: `just build && just test-go` from `go/`
Expected: All pass

**Step 6: Commit**

```
refactor(xdg): move from internal/echo to lib/echo
```

---

### Task 6: Update CLAUDE.md files

**Files:**
- Create: `go/lib/echo/xdg/CLAUDE.md` (moved with git mv, verify content)
- Modify: `docs/plans/2026-02-27-lib-internal-split-design.md` — remove xdg
  from "Refactoring Candidates" list

**Step 1: Verify CLAUDE.md moved correctly**

The `git mv` in Task 5 should have moved the existing
`internal/echo/xdg/CLAUDE.md` to `lib/echo/xdg/CLAUDE.md`. Verify it exists
and its content is still accurate.

**Step 2: Update lib-internal-split-design.md**

In the "Refactoring Candidates" section, remove the xdg bullet:

```
- `delta/xdg` — remove `blob_store_id.LocationType` return from
  `GetLocationType()`
```

**Step 3: Commit**

```
docs: update references after xdg move to lib
```

---

### Task 7: Run integration tests

**Step 1: Build debug binaries**

Run: `just build` from `go/`

**Step 2: Run full test suite**

Run: `just test` from `go/`
Expected: All pass (unit + integration)

**Step 3: If tests fail, investigate and fix**

This is a pure refactor (no behavior change), so failures would indicate a
missed import update or a caller that was referencing the old path via a
transitive dependency.

---

## Summary of Changes

| What | Before | After |
|------|--------|-------|
| Package path | `internal/echo/xdg` | `lib/echo/xdg` |
| `GetLocationType()` | On `xdg.XDG`, returns `blob_store_id.LocationType` | Removed |
| `IsOverridden()` | N/A | On `xdg.XDG`, returns `bool` |
| `EnvXDGUtilityNameOverride` | Package-level const `"DODDER_XDG_UTILITY_OVERRIDE"` | `InitArgs.OverrideEnvVarName` field, set by callers |
| `blob_store_id` import | In `xdg` package | Only in `directory_layout` and other `internal/` consumers |
| Location type derivation | `xdg.XDG` knows about `LocationType` | `directory_layout.v3` derives `LocationType` from `IsOverridden()` |
