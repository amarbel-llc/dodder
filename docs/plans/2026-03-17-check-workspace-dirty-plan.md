# `check-workspace dirty` Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `dodder check-workspace dirty` — a fast, quiet command that exits 0 if the workspace-repo has local changes since its last parent sync, exit 1 if clean, exit 2 if not in a workspace-repo.

**Architecture:** Store sync baseline (TAI string + object digest string) in V1 workspace config at clone/pull time. The check command loads the config, reads the latest inventory list, and compares TAI then digest. No parent repo access needed.

**Tech Stack:** Go, TOML workspace config, inventory list store, BATS integration tests.

**Rollback:** Purely additive. New V1 fields are `omitempty` and ignored by code that doesn't read them. Remove the command file + revert V1 struct changes.

---

### Task 1: Add sync baseline fields to V1 workspace config

**Files:**
- Modify: `go/internal/echo/workspace_config_blobs/v1.go`
- Modify: `go/internal/echo/workspace_config_blobs/main.go`

**Step 1: Add SyncTai and SyncDigest string fields to V1**

In `go/internal/echo/workspace_config_blobs/v1.go`, add two string fields and a
new interface for accessing them:

```go
type V1 struct {
	V0
	ParentPath string `toml:"parent-path,omitempty"`
	SyncTai    string `toml:"sync-tai,omitempty"`
	SyncDigest string `toml:"sync-digest,omitempty"`
}
```

TAI and markl.Id don't implement `encoding.TextMarshaler`, so store them as
pre-formatted strings. `ids.Tai.String()` and `markl.Id.String()` produce the
text representation; `ids.Tai.Set()` and `markl.Id.Set()` parse it back.

**Step 2: Add getter methods and interface**

In `go/internal/echo/workspace_config_blobs/v1.go`:

```go
func (blob V1) GetSyncTai() string {
	return blob.SyncTai
}

func (blob V1) GetSyncDigest() string {
	return blob.SyncDigest
}
```

In `go/internal/echo/workspace_config_blobs/main.go`, add a new interface:

```go
ConfigWithSyncBaseline interface {
	Config
	GetSyncTai() string
	GetSyncDigest() string
}
```

Add the interface assertion in `v1.go`:

```go
var (
	_ ConfigWithParentPath     = V1{}
	_ ConfigWithSyncBaseline   = V1{}
	_ ConfigWithDefaultQueryString = V1{}
	_ ConfigWithDryRun         = V1{}
)
```

**Step 3: Add accessor to env_workspace**

In `go/internal/november/env_workspace/main.go`, add to the `Env` interface and
implement:

```go
// In Env interface:
GetSyncBaseline() (tai string, digest string)

// Implementation:
func (env *env) GetSyncBaseline() (tai string, digest string) {
	if sb, ok := env.blob.(workspace_config_blobs.ConfigWithSyncBaseline); ok {
		return sb.GetSyncTai(), sb.GetSyncDigest()
	}
	return "", ""
}
```

**Step 4: Build and verify compilation**

Run: `just build` from repo root.
Expected: Compiles successfully.

**Step 5: Commit**

```
feat(workspace): add sync baseline fields to V1 workspace config
```

---

### Task 2: Write sync baseline at clone/pull/push time

**Files:**
- Modify: `go/internal/november/env_workspace/main.go` — add `UpdateSyncBaseline` method
- Modify: `go/internal/victor/commands_dodder/init_workspace.go:164-230` — write baseline after clone
- Modify: `go/internal/victor/commands_dodder/pull.go:32-77` — write baseline after pull
- Modify: `go/internal/victor/commands_dodder/push.go:32-77` — write baseline after push

**Step 1: Add UpdateSyncBaseline to env_workspace**

In `go/internal/november/env_workspace/main.go`, add to the `Env` interface and
implement a method that reads the latest inventory list and writes the baseline
to the config:

```go
// In Env interface:
UpdateSyncBaseline(inventoryListStore sku.InventoryListStore) error

// Implementation:
func (env *env) UpdateSyncBaseline(
	inventoryListStore sku.InventoryListStore,
) (err error) {
	v1, ok := env.blob.(*workspace_config_blobs.V1)
	if !ok {
		return nil // V0 workspace, no-op
	}

	last, err := inventoryListStore.ReadLast()
	if err != nil {
		return errors.Wrap(err)
	}

	v1.SyncTai = last.GetTai().String()
	v1.SyncDigest = last.GetMetadata().GetObjectDigest().String()

	return env.rewriteConfig()
}
```

Also add a `rewriteConfig` helper that re-encodes the current blob to the
workspace config file path (same logic as the tail of `CreateWorkspace`):

```go
func (env *env) rewriteConfig() error {
	object := env.GetWorkspaceConfigTyped()

	if err := hyphence.EncodeToFile(
		workspace_config_blobs.Coder,
		&object,
		env.GetWorkspaceConfigFilePath(),
	); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
```

Note: `EncodeToFile` may fail with `IsExist` if the file already exists. Check
the existing `CreateWorkspace` code — it treats `IsExist` as an error. For
`rewriteConfig`, the file already exists and we want to overwrite. Verify that
`hyphence.EncodeToFile` supports overwriting or use a write-then-rename pattern.
If it doesn't, use `hyphence.EncodeTo` with an `os.Create` call.

**Step 2: Wire into init_workspace.go runExperimentalRepo**

In `go/internal/victor/commands_dodder/init_workspace.go`, after the
`PullQueryGroupFromRemote` call (line 195-202) and before writing the workspace
config (line 214-225), capture the sync baseline. The simplest approach: after
`CreateWorkspace` at line 227, call `UpdateSyncBaseline`:

```go
// After line 228 (CreateWorkspace call):
if err := local.GetEnvWorkspace().UpdateSyncBaseline(
	local.GetInventoryListStore(),
); err != nil {
	req.Cancel(err)
}
```

**Step 3: Wire into pull.go**

In `go/internal/victor/commands_dodder/pull.go`, after the
`PullQueryGroupFromRemote` call (line 70-76), add:

```go
if err := localWorkingCopy.GetEnvWorkspace().UpdateSyncBaseline(
	localWorkingCopy.GetInventoryListStore(),
); err != nil {
	localWorkingCopy.Cancel(err)
}
```

**Step 4: Wire into push.go**

In `go/internal/victor/commands_dodder/push.go`, after the
`PullQueryGroupFromRemote` call (line 70-76), add:

```go
if err := local.GetEnvWorkspace().UpdateSyncBaseline(
	local.GetInventoryListStore(),
); err != nil {
	local.Cancel(err)
}
```

**Step 5: Build and verify compilation**

Run: `just build` from repo root.
Expected: Compiles successfully.

**Step 6: Commit**

```
feat(workspace): write sync baseline on clone, pull, and push
```

---

### Task 3: Implement check-workspace dirty command

**Files:**
- Create: `go/internal/victor/commands_dodder/check_workspace.go`

**Step 1: Write the command**

```go
package commands_dodder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd(
		"check-workspace",
		&CheckWorkspace{},
	)
}

type CheckWorkspace struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd CheckWorkspace) Run(req command.Request) {
	subcmd := req.PopArg("subcommand (dirty)")

	if subcmd != "dirty" {
		req.Cancel(errors.BadRequestf("unknown check-workspace subcommand: %s", subcmd))
		return
	}

	cmd.runDirty(req)
}

func (cmd CheckWorkspace) runDirty(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)
	envWorkspace := localWorkingCopy.GetEnvWorkspace()

	if envWorkspace.IsTemporary() {
		fmt.Fprintln(os.Stderr, "not in a workspace")
		os.Exit(2)
	}

	syncTai, syncDigest := envWorkspace.GetSyncBaseline()
	if syncTai == "" {
		// No baseline stored — treat as dirty (workspace predates this feature
		// or is a V0 lightweight workspace with no sync tracking)
		os.Exit(0)
	}

	last, err := localWorkingCopy.GetInventoryListStore().ReadLast()
	if err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	currentTai := last.GetTai().String()
	currentDigest := last.GetMetadata().GetObjectDigest().String()

	// Compare TAI first (cheapest), then digest as belt-and-suspenders
	if currentTai != syncTai || currentDigest != syncDigest {
		os.Exit(0) // dirty
	}

	os.Exit(1) // clean
}
```

Note: This command uses `os.Exit` directly for the three exit codes. The
framework's `Cancel` mechanism only supports exit 0 and 1. For a shell-prompt
command that communicates via exit codes, direct `os.Exit` is appropriate. The
`dirty` check avoids parsing TAI or markl values — string comparison is
sufficient and faster.

**Step 2: Build and verify compilation**

Run: `just build` from repo root.
Expected: Compiles successfully.

**Step 3: Commit**

```
feat: add check-workspace dirty command
```

---

### Task 4: Add check-workspace to completion test

**Files:**
- Modify: `zz-tests_bats/current_version/complete.bats:84-168`

**Step 1: Add check-workspace to complete_subcmd expected output**

In the `complete_subcmd` test, add `check-workspace` to the expected output list
(alphabetical order, between `cat-alfred` and `checkin`):

```
		cat-alfred
		check-workspace
		checkin
```

**Step 2: Run completion test**

Run: `just test-bats-targets complete.bats` from repo root.
Expected: PASS.

**Step 3: Commit**

```
test: add check-workspace to completion test
```

---

### Task 5: Write BATS integration tests

**Files:**
- Create: `zz-tests_bats/current_version/check_workspace_dirty.bats`

**Step 1: Write tests**

```bash
#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:workspace,user_story:repo

function bootstrap_workspace_repo {
	local parent="$1"
	local workspace="$2"

	(
		mkdir -p "$parent"
		pushd "$parent" || exit 1
		run_dodder_init -repo_id . "parent-repo-id"

		run_dodder new -edit=false - <<-EOM
			---
			# test zettel
			- project-alpha
			! md
			---

			test body
		EOM
		assert_success
	)

	local parent_path
	parent_path="$(realpath "$parent")"

	mkdir -p "$workspace"
	pushd "$workspace" || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett

	assert_success
}

function check_workspace_dirty_clean_after_init { # @test
	bootstrap_workspace_repo parent workspace

	# Immediately after init, workspace should be clean
	run_dodder check-workspace dirty
	assert_failure  # exit 1 = clean
}

function check_workspace_dirty_after_local_change { # @test
	bootstrap_workspace_repo parent workspace

	# Create a new zettel in the workspace
	run_dodder new -edit=false - <<-EOM
		---
		# workspace zettel
		- project-alpha
		! md
		---

		workspace body
	EOM
	assert_success

	# Now workspace should be dirty
	run_dodder check-workspace dirty
	assert_success  # exit 0 = dirty
}

function check_workspace_dirty_clean_after_push { # @test
	bootstrap_workspace_repo parent workspace

	local parent_path
	parent_path="$(realpath ../parent)"

	# Create a new zettel in the workspace
	run_dodder new -edit=false - <<-EOM
		---
		# workspace zettel
		- project-alpha
		! md
		---

		workspace body
	EOM
	assert_success

	# Push to parent
	run_dodder push
	assert_success

	# After push, workspace should be clean again
	run_dodder check-workspace dirty
	assert_failure  # exit 1 = clean
}

function check_workspace_dirty_not_in_workspace { # @test
	# Run in a directory that is not a workspace
	copy_from_version "$DIR"

	run_dodder check-workspace dirty
	# exit 2 = not in a workspace-repo
	assert_failure
	[ "$status" -eq 2 ]
	assert_output --partial 'not in a workspace'
}

function check_workspace_dirty_quiet_output { # @test
	bootstrap_workspace_repo parent workspace

	# Clean workspace should produce no stdout
	run_dodder check-workspace dirty
	assert_output ''
}
```

**Step 2: Run tests**

Run: `just test-bats-targets check_workspace_dirty.bats` from repo root.
Expected: All tests PASS.

**Step 3: Commit**

```
test: add BATS integration tests for check-workspace dirty
```

---

### Task 6: Update FDR-0005 implementation status

**Files:**
- Modify: `docs/features/0005-workspace-as-repo.md`

**Step 1: Update "What's Built" section**

Add divergence detection to the built list. Move it from "What's NOT Built" to
"What's Built". Update the description to reflect the local-only dirty check.

**Step 2: Commit**

```
docs: update FDR-0005 with check-workspace dirty implementation status
```

---

### Task 7: Run full test suite

**Step 1: Run all tests**

Run: `just test` from repo root.
Expected: All tests pass, including existing workspace_repo.bats tests.

**Step 2: If failures, fix and commit**

Existing tests may need adjustments if the sync baseline write changes the
timing or content of the workspace config file. Check `workspace_repo.bats`
assertions against `.dodder-workspace` content.
