# Implicit Parent Transfers Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Push/pull automatically use the workspace config's stored parent path when no remote is explicitly specified, plus error case tests for workspace-as-repo edge cases.

**Architecture:** Add a `ResolveImplicitDirectPath` method on `Remote` that accepts a `local_working_copy.Repo`, checks the workspace config for a parent path, and sets `DirectPath` if no explicit remote was provided. Call it in push.go and pull.go after `MakeLocalWorkingCopy` but before the remote-resolution branch. Error tests exercise init-on-existing-repo, stale parent path, and empty query.

**Tech Stack:** Go, BATS integration tests

**Rollback:** Remove `ResolveImplicitDirectPath` calls from push/pull. Purely additive — explicit `-direct` and stored-remote workflows unchanged.

---

## Critical Files

| Action | File | Purpose |
|--------|------|---------|
| Modify | `go/internal/uniform/command_components_dodder/remote.go` | Add `ResolveImplicitDirectPath` |
| Modify | `go/internal/victor/commands_dodder/pull.go` | Call `ResolveImplicitDirectPath` |
| Modify | `go/internal/victor/commands_dodder/push.go` | Call `ResolveImplicitDirectPath` |
| Modify | `zz-tests_bats/current_version/workspace_repo.bats` | New tests |
| Modify | `docs/features/0005-workspace-as-repo.md` | Update FDR |

## Key Functions to Reuse

- `local.GetEnvWorkspace().GetParentPath()` — returns stored parent path from V1 config, empty string otherwise (`go/internal/november/env_workspace/main.go:304`)
- `cmd.IsDirectTransfer()` — checks if `-direct` flag was provided (`go/internal/uniform/command_components_dodder/remote.go:51`)
- `cmd.DirectPath` — the string field on `Remote` that `MakeDirectRemoteFromPath` reads (`go/internal/uniform/command_components_dodder/remote.go:33`)

---

## Task 1: Add `ResolveImplicitDirectPath` to Remote

### 1a: Add method

**File:** `go/internal/uniform/command_components_dodder/remote.go`

Add after `IsDirectTransfer()` (line 53):

```go
func (cmd *Remote) ResolveImplicitDirectPath(
	local *local_working_copy.Repo,
) {
	if cmd.IsDirectTransfer() {
		return
	}

	parentPath := local.GetEnvWorkspace().GetParentPath()
	if parentPath != "" {
		cmd.DirectPath = parentPath
	}
}
```

This needs an import for `local_working_copy`. Check that `remote.go` already imports `sierra/local_working_copy` — it does (line 18).

The method is a no-op when:
- `-direct` was already provided (explicit override wins)
- The workspace has no parent path (lightweight workspace or V0 config)

### Verify: `cd go && go build ./internal/uniform/command_components_dodder/`

---

## Task 2: Wire into pull.go

**File:** `go/internal/victor/commands_dodder/pull.go`

### 2a: Add implicit resolution

In `Run()`, after `MakeLocalWorkingCopy` (line 33) and before the `if cmd.IsDirectTransfer()` branch (line 37), add:

```go
cmd.ResolveImplicitDirectPath(localWorkingCopy)
```

The full `Run()` becomes:

```go
func (cmd Pull) Run(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	cmd.ResolveImplicitDirectPath(localWorkingCopy)

	var remote repo.Repo
	// ... rest unchanged
```

### Verify: `cd go && go build ./internal/victor/commands_dodder/`

---

## Task 3: Wire into push.go

**File:** `go/internal/victor/commands_dodder/push.go`

### 3a: Add implicit resolution

Same pattern as pull. In `Run()`, after `MakeLocalWorkingCopy` (line 33), add:

```go
cmd.ResolveImplicitDirectPath(local)
```

### Verify: `cd go && go build ./internal/victor/commands_dodder/`

---

## Task 4: BATS test — implicit push/pull

**File:** `zz-tests_bats/current_version/workspace_repo.bats`

### 4a: Add test for implicit parent transfers

Add after `workspace_repo_init_experimental_repo`:

```bash
function workspace_repo_implicit_parent_push_pull { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# --- Create content in workspace ---
	run_dodder new -edit=false - <<-EOM
		---
		# implicit-push zettel
		- project-alpha
		! md
		---

		created for implicit push test
	EOM
	assert_success

	# --- Push WITHOUT -direct (should use stored parent path) ---
	run_dodder push
	assert_success

	# Verify parent received the new zettel
	pushd "$parent_path" || exit 1
	run_dodder show :z
	assert_success
	assert_output --partial 'implicit-push zettel'
	popd || exit 1

	# --- Add content in parent ---
	(
		pushd "$parent_path" || exit 1
		run_dodder new -edit=false - <<-EOM
			---
			# implicit-pull zettel
			- project-alpha
			! md
			---

			created for implicit pull test
		EOM
		assert_success
	)

	# --- Pull WITHOUT -direct (should use stored parent path) ---
	run_dodder pull
	assert_success

	# Verify workspace received the new parent zettel
	run_dodder show :z
	assert_success
	assert_output --partial 'implicit-pull zettel'
}
```

### Verify: `just test-bats-targets current_version/workspace_repo.bats`

---

## Task 5: BATS test — error: init on existing repo

**File:** `zz-tests_bats/current_version/workspace_repo.bats`

### 5a: Add test

```bash
function workspace_repo_init_experimental_repo_existing_repo { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# --- Second init should fail (repo already exists) ---
	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id-2 \
		project-alpha:z

	assert_failure
}
```

### Verify: `just test-bats-targets current_version/workspace_repo.bats`

---

## Task 6: BATS test — error: stale parent path

**File:** `zz-tests_bats/current_version/workspace_repo.bats`

### 6a: Add test

```bash
function workspace_repo_stale_parent_path { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# --- Delete the parent repo ---
	rm -rf "$parent_path"

	# --- Push should fail with meaningful error ---
	run_dodder push
	assert_failure

	# --- Pull should fail with meaningful error ---
	run_dodder pull
	assert_failure
}
```

### Verify: `just test-bats-targets current_version/workspace_repo.bats`

---

## Task 7: BATS test — empty query (no matching objects)

**File:** `zz-tests_bats/current_version/workspace_repo.bats`

### 7a: Add test

```bash
function workspace_repo_init_experimental_repo_empty_query { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	# Query for a tag that doesn't exist — should succeed with empty workspace
	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		nonexistent-tag:z

	assert_success

	# Workspace should have no zettels
	run_dodder show :z
	assert_success
	assert_output ''

	# Workspace config should exist
	assert [ -f .dodder-workspace ]
}
```

### Verify: `just test-bats-targets current_version/workspace_repo.bats`

---

## Task 8: Update FDR-0005

**File:** `docs/features/0005-workspace-as-repo.md`

### 8a: Update CLI Surface section

Replace the push/pull examples that show `-direct <stored-parent-path>` with implicit versions. Update the "Open question" about automatic parent detection to note it's now implemented. Rename "Automatic Parent Push/Pull" in Future Possibilities to "Implicit Parent Transfers" and move it to the main Design section.

### 8b: Update status

If all promotion criteria are met after Task 7, update `status: exploring` to `status: experimental` in the frontmatter.

### Verify: `just test` (full suite, no regressions)

---

## Verification

1. `just build` — compiles
2. `just test-bats-targets current_version/workspace_repo.bats` — all workspace tests pass
3. `just test` — no regressions
4. Existing tests `workspace_repo_clone_pull_push`, `workspace_repo_clone_filtered_by_tag`, `workspace_repo_pull_filtered_by_tag`, `workspace_repo_push_unfiltered`, `workspace_repo_init_experimental_repo` all pass unchanged
