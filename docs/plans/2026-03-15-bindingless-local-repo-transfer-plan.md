# Bindingless Local Repo Transfer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `-direct <path>` flag to push, pull, and clone to enable ad-hoc local repo-to-repo transfer without a stored remote object.

**Architecture:** Add a `DirectPath` string field to the `Remote` command component. When set, construct a `TomlLocalOverridePathV0` blob inline and call `MakeRemoteFromBlob`, bypassing the stored-object lookup. Wrap `ErrNotInDodderDir` with direct-remote context on failure.

**Tech Stack:** Go, BATS integration tests

**Rollback:** Remove the `-direct` flag, `DirectPath` field, and `MakeDirectRemoteFromPath` method. Purely additive — no persistent state changes.

---

### Task 1: Add `-direct` flag and `MakeDirectRemoteFromPath` to Remote component

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/uniform/command_components_dodder/remote.go:25-44`

**Step 1: Add `DirectPath` field and flag registration**

In `remote.go`, add a `DirectPath` string field to `Remote` and register it in `SetFlagDefinitions`:

```go
type Remote struct {
	RemoteRepoBlobs

	InventoryLists
	LocalWorkingCopy

	RemoteConnectionType remote_connection_types.Type
	DirectPath           string
}

func (cmd *Remote) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cli.FlagSetVarWithCompletion(
		flagSet,
		&cmd.RemoteConnectionType,
		"remote-connection-type",
	)

	flagSet.StringVar(
		&cmd.DirectPath,
		"direct",
		"",
		"path to a local dodder repository for direct transfer without a stored remote",
	)
}
```

**Step 2: Add `MakeDirectRemoteFromPath` method**

Add this method to `remote.go`, below `MakeRemoteFromBlob`:

```go
func (cmd Remote) MakeDirectRemoteFromPath(
	req command.Request,
	local *local_working_copy.Repo,
) repo.Repo {
	absPath := cmd.DirectPath

	if !filepath.IsAbs(absPath) {
		var err error

		if absPath, err = filepath.Abs(absPath); err != nil {
			req.Cancel(err)
		}
	}

	blob := &repo_blobs.TomlLocalOverridePathV0{
		OverridePath: absPath,
	}

	return cmd.MakeRemoteFromBlob(req, local, blob)
}
```

Add `"path/filepath"` to the import block.

Note: `MakeRemoteFromBlob` already handles the `BlobOverridePath` case via
`MakeWithXDGRootOverrideHomeAndInitialize`. If the path doesn't contain a dodder
repo, `env_repo.Make` returns `ErrNotInDodderDir` which cancels the context with
a message like "not in a dodder directory. Looking for <path>/.local/share/dodder/Konfig".
This is sufficient error context — the path itself tells the user which
`-direct` target failed.

**Step 3: Add `IsDirectTransfer` helper**

Add a convenience method:

```go
func (cmd Remote) IsDirectTransfer() bool {
	return cmd.DirectPath != ""
}
```

**Step 4: Build to verify compilation**

Run: `just build` from repo root.
Expected: builds successfully.

**Step 5: Commit**

```
feat: add -direct flag and MakeDirectRemoteFromPath to Remote component
```

---

### Task 2: Wire `-direct` into pull command

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/victor/commands_dodder/pull.go:31-68`

**Step 1: Branch on `-direct` in `Run`**

Replace the `Run` method body:

```go
func (cmd Pull) Run(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	var remote repo.Repo

	if cmd.IsDirectTransfer() {
		remote = cmd.MakeDirectRemoteFromPath(req, localWorkingCopy)
	} else {
		var object *sku.Transacted

		{
			var err error

			if object, err = localWorkingCopy.GetObjectFromObjectId(
				req.PopArg("repo-id"),
			); err != nil {
				localWorkingCopy.Cancel(err)
			}
		}

		remote = cmd.MakeRemote(req, localWorkingCopy, object)
	}

	qg := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		localWorkingCopy,
		req.PopArgs(),
	)

	if err := localWorkingCopy.PullQueryGroupFromRemote(
		remote,
		qg,
		cmd.WithPrintCopies(true),
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
```

Add `"code.linenisgreat.com/dodder/go/internal/quebec/repo"` to the import
block.

**Step 2: Build to verify compilation**

Run: `just build` from repo root.
Expected: builds successfully.

**Step 3: Commit**

```
feat: wire -direct flag into pull command
```

---

### Task 3: Wire `-direct` into push command

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/victor/commands_dodder/push.go:31-68`

**Step 1: Branch on `-direct` in `Run`**

Replace the `Run` method body with the same pattern as pull, but calling
`remote.PullQueryGroupFromRemote(local, ...)` instead of
`local.PullQueryGroupFromRemote(remote, ...)`:

```go
func (cmd Push) Run(req command.Request) {
	local := cmd.MakeLocalWorkingCopy(req)

	var remote repo.Repo

	if cmd.IsDirectTransfer() {
		remote = cmd.MakeDirectRemoteFromPath(req, local)
	} else {
		var remoteObject *sku.Transacted

		{
			var err error

			if remoteObject, err = local.GetObjectFromObjectId(
				req.PopArg("repo-id"),
			); err != nil {
				local.Cancel(err)
			}
		}

		remote = cmd.MakeRemote(req, local, remoteObject)
	}

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		local,
		req.PopArgs(),
	)

	if err := remote.PullQueryGroupFromRemote(
		local,
		queryGroup,
		cmd.WithPrintCopies(true),
	); err != nil {
		local.Cancel(err)
	}
}
```

Add `"code.linenisgreat.com/dodder/go/internal/quebec/repo"` to the import
block.

**Step 2: Build to verify compilation**

Run: `just build` from repo root.
Expected: builds successfully.

**Step 3: Commit**

```
feat: wire -direct flag into push command
```

---

### Task 4: Wire `-direct` into clone command

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/victor/commands_dodder/clone.go:41-67`

**Step 1: Branch on `-direct` in `Run`**

Replace the `Run` method body. Clone creates a new repo first, then either uses
`-direct` path or the existing `MakeRemoteAndObject` path:

```go
func (cmd Clone) Run(req command.Request) {
	local := cmd.OnTheFirstDay(req, req.PopArg("new repo id"))

	var remote repo.Repo

	if cmd.IsDirectTransfer() {
		remote = cmd.MakeDirectRemoteFromPath(req, local)
	} else {
		// TODO offer option to persist remote object, if supported
		remote, _ = cmd.MakeRemoteAndObject(req, local)
	}

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		local,
		req.PopArgs(),
	)

	if err := local.PullQueryGroupFromRemote(
		remote,
		queryGroup,
		cmd.WithPrintCopies(true),
	); err != nil {
		req.Cancel(err)
	}
}
```

Add `"code.linenisgreat.com/dodder/go/internal/quebec/repo"` to the import
block.

**Step 2: Build to verify compilation**

Run: `just build` from repo root.
Expected: builds successfully.

**Step 3: Commit**

```
feat: wire -direct flag into clone command
```

---

### Task 5: Write BATS tests for direct clone

**Promotion criteria:** N/A

**Files:**
- Modify: `zz-tests_bats/current_version/clone.bats`

**Step 1: Add test for direct clone**

Add after the last existing test function in `clone.bats`:

```bash
function clone_direct_local_path { # @test
	them="them"
	bootstrap "$them"

	run_clone_default_with \
		-direct "$(realpath ./them)" \
		test-repo-id-us \
		+zettel,typ,etikett

	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		\[konfig @blake2b256-.* !toml-config-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		\[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc \(5 B)
		copied Blob blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m \(51 B)
	EOM

	try_add_new_after_clone
}

function clone_direct_nonexistent_path { # @test
	run_clone_default_with \
		-direct "/nonexistent/path" \
		test-repo-id-us

	assert_failure
	assert_output --partial 'not in a dodder directory'
}
```

Note: The `run_clone_default_with` helper passes args through to `run_dodder
clone`. The `-direct` flag replaces the `toml-repo-local_override_path-v0
<path>` positional args (remote-type + path). The `test-repo-id-us` positional
arg is still needed (it's the new repo id for `OnTheFirstDay`).

**Step 2: Run the new tests**

Run: `just test-bats-targets clone.bats` from repo root.
Expected: both new tests pass.

**Step 3: Commit**

```
test: add BATS tests for direct clone
```

---

### Task 6: Write BATS tests for direct pull

**Promotion criteria:** N/A

**Files:**
- Modify: `zz-tests_bats/current_version/pull.bats`

**Step 1: Add test for direct pull**

Add after the last existing test function in `pull.bats`:

```bash
function pull_direct_local_path_no_conflicts { # @test
	them="them"
	bootstrap_repo "$them"

	pushd "$BATS_TEST_TMPDIR" || exit 1

	run_dodder_init_disable_age

	run_dodder pull -direct "$(realpath them)" +zettel,typ,etikett

	assert_success
	assert_output_unsorted - <<-EOM
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 (36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc (5 B)
		copied Blob blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e (15 B)
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
	EOM

	try_add_new_after_pull
}

function pull_direct_nonexistent_path { # @test
	pushd "$BATS_TEST_TMPDIR" || exit 1
	run_dodder_init_disable_age

	run_dodder pull -direct "/nonexistent/path" +zettel

	assert_failure
	assert_output --partial 'not in a dodder directory'
}
```

**Step 2: Run the new tests**

Run: `just test-bats-targets pull.bats` from repo root.
Expected: both new tests pass.

**Step 3: Commit**

```
test: add BATS tests for direct pull
```

---

### Task 7: Write BATS tests for direct push

**Promotion criteria:** N/A

**Files:**
- Modify: `zz-tests_bats/current_version/push.bats`

**Step 1: Add test for direct push**

Add after the last existing test function in `push.bats`. The push tests use
`copy_from_version "$DIR"` in setup to get a pre-populated repo, so we bootstrap
an empty target to push into:

```bash
function push_direct_local_path_no_conflicts { # @test
	(
		mkdir -p them
		pushd them || exit 1
		run_dodder_init
	)

	run_dodder push -direct "$(realpath them)" +zettel,typ,etikett

	assert_success
	assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
	EOM

	(
		pushd them || exit 1
		run_dodder show +zettel,typ,konfig,etikett
		assert_output_unsorted - <<-EOM
			[konfig @$(get_konfig_sha) !toml-config-v2]
			[!md @$(get_type_blob_sha) !toml-type-v1]
			[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
			[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
			[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		EOM
	)
}

function push_direct_nonexistent_path { # @test
	run_dodder push -direct "/nonexistent/path" +zettel

	assert_failure
	assert_output --partial 'not in a dodder directory'
}
```

**Step 2: Run the new tests**

Run: `just test-bats-targets push.bats` from repo root.
Expected: both new tests pass.

**Step 3: Commit**

```
test: add BATS tests for direct push
```

---

### Task 8: Run full test suite

**Files:** None (verification only)

**Step 1: Run all tests**

Run: `just test` from repo root.
Expected: all existing + new tests pass. No regressions.

**Step 2: Commit FDR and plan**

```
docs: add FDR and implementation plan for bindingless local repo transfer
```
