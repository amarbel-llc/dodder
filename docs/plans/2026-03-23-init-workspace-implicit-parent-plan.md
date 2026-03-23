# init-workspace: Implicit Parent Discovery --- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Remove `-direct` from `init-workspace`, add `-parent` flag, flip
`-experimental-repo` default to true, and auto-discover the home repo as parent.

**Architecture:** `init-workspace` resolves the parent repo (home XDG or
CWD-scoped via `-parent`), validates it exists, links zettel ID providers, pulls
from it, and stores the parent type in workspace config. Push/pull resolve the
parent at runtime via `ResolveImplicitParent`, which returns a `repo_blobs.Blob`
(either `TomlXDGV0` for home repos or `TomlLocalOverridePathV0` for CWD-scoped).

**Tech Stack:** Go, BATS

**Rollback:** `-experimental-repo=false` opts out. N/A for additive flag
changes.

--------------------------------------------------------------------------------

### Task 1: Add `-parent` flag and remove `-direct` requirement from init-workspace

**Files:** - Modify: `go/internal/victor/commands_dodder/init_workspace.go`

**Step 1: Update struct and flag definitions**

Add `ParentPath string` field to `InitWorkspace`. Replace the `-direct`
requirement check with `-parent` flag registration. Flip `-experimental-repo`
default to `true`. Remove `RemoteTransfer` embedding (it brings `-direct` which
we don't want for init-workspace). Keep `Remote` available through a local field
for `MakeDirectRemoteFromPath` usage with the resolved parent path.

``` go
type InitWorkspace struct {
    command_components_dodder.Genesis
    command_components_dodder.Query

    complete command_components_dodder.Complete

    ExperimentalRepo  bool
    ParentPath        string
    DefaultQueryGroup values.String
    Proto             sku.Proto
}
```

In `SetFlagDefinitions`:

``` go
flagSet.BoolVar(
    &cmd.ExperimentalRepo,
    "experimental-repo",
    true, // flipped from false
    "create a repo-backed workspace with independent store and commit history",
)

flagSet.StringVar(
    &cmd.ParentPath,
    "parent",
    "",
    "path to a CWD-scoped parent dodder repository (omit for home repo)",
)
```

Remove `cmd.RemoteTransfer.SetFlagDefinitions(flagSet)` --- this removes the
`-direct` flag from init-workspace.

**Step 2: Update `runExperimentalRepo` --- validation and parent resolution**

Replace the `-direct` requirement and `config.RepoId` override with:

``` go
func (cmd InitWorkspace) runExperimentalRepo(req command.Request) {
    config := req.Utility.GetConfigAny().(*repo_config_cli.Config)

    // Reject -repo_id with -experimental-repo
    if !config.RepoId.IsEmpty() {
        req.Cancel(
            errors.BadRequestf(
                "-repo_id cannot be used with -experimental-repo (workspace repos are always CWD-rooted)",
            ),
        )
        return
    }

    // Force CWD routing
    if err := config.RepoId.Set("."); err != nil {
        req.Cancel(err)
        return
    }

    // Resolve and validate parent
    absParentPath, parentIsHomeRepo := cmd.resolveParentPath(req)

    // Validate parent repo exists
    cmd.validateParentRepo(req, absParentPath, parentIsHomeRepo)

    cmd.Genesis.BigBang.ExcludeDefaultType = true
    cmd.linkParentZettelIdProviders(absParentPath, parentIsHomeRepo)

    local := cmd.OnTheFirstDay(req, req.PopArg("workspace repo id"))

    // Create remote from parent
    remote := cmd.makeParentRemote(req, local, absParentPath, parentIsHomeRepo)

    queryArgs := req.PopArgs()

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
        queryArgs,
    )

    if err := local.PullQueryGroupFromRemote(
        remote,
        queryGroup,
        cmd.WithPrintCopies(true),
    ); err != nil {
        req.Cancel(err)
        return
    }

    var parentPathForConfig string
    if !parentIsHomeRepo {
        parentPathForConfig = absParentPath
    }

    blob := &workspace_config_blobs.V1{
        V0: workspace_config_blobs.V0{
            Query: cmd.DefaultQueryGroup.String(),
            Defaults: repo_configs.DefaultsV1OmitEmpty{
                Type: cmd.Proto.Metadata.GetType().ToType(),
                Tags: slices.Collect(
                    ids.ITagSeqToTagStructSeq(cmd.Proto.Metadata.AllTags()),
                ),
            },
        },
        ParentPath: parentPathForConfig,
    }

    if err := local.GetEnvWorkspace().CreateWorkspace(blob); err != nil {
        req.Cancel(err)
    }

    if err := local.GetEnvWorkspace().UpdateSyncBaseline(
        local.GetInventoryListStore(),
    ); err != nil {
        req.Cancel(err)
    }
}
```

**Step 3: Add helper methods**

``` go
// resolveParentPath returns the absolute parent path and whether it's the
// home repo. When -parent is omitted, resolves the home XDG data directory.
func (cmd InitWorkspace) resolveParentPath(
    req command.Request,
) (absPath string, isHomeRepo bool) {
    if cmd.ParentPath != "" {
        absPath = cmd.ParentPath
        if !filepath.IsAbs(absPath) {
            var err error
            if absPath, err = filepath.Abs(absPath); err != nil {
                req.Cancel(err)
            }
        }
        return absPath, false
    }

    // Home repo — resolve XDG data dir to find inventory_lists_log
    home, err := os.UserHomeDir()
    if err != nil {
        req.Cancel(err)
    }

    // Check XDG_DATA_HOME env var first, fall back to default
    dataHome := os.Getenv("XDG_DATA_HOME")
    if dataHome == "" {
        dataHome = filepath.Join(home, ".local", "share")
    }

    absPath = filepath.Join(dataHome, env_dir.XDGUtilityNameDodder)
    return absPath, true
}

// validateParentRepo checks that the parent path contains a valid dodder repo.
func (cmd InitWorkspace) validateParentRepo(
    req command.Request,
    absPath string,
    isHomeRepo bool,
) {
    // Check for inventory_lists_log as proof of a valid repo
    inventoryListLog := filepath.Join(absPath, "inventory_lists_log")
    if !files.Exists(inventoryListLog) {
        if isHomeRepo {
            req.Cancel(
                errors.BadRequestf(
                    "no dodder repo found at %s; run `dodder init` first",
                    absPath,
                ),
            )
        } else {
            req.Cancel(
                errors.BadRequestf(
                    "no dodder repo found at -parent path %s",
                    absPath,
                ),
            )
        }
    }
}
```

**Step 4: Update `linkParentZettelIdProviders`**

Add `isHomeRepo` parameter to select the right directory layout:

``` go
func (cmd *InitWorkspace) linkParentZettelIdProviders(
    absParentPath string,
    isHomeRepo bool,
) {
    if cmd.Genesis.BigBang.Yin != "" || cmd.Genesis.BigBang.Yang != "" {
        return
    }

    var parentObjectIdDir string
    if isHomeRepo {
        // Home repo: XDG layout at <dataHome>/object_ids/
        parentObjectIdDir = filepath.Join(absParentPath, "object_ids")
    } else {
        // CWD-scoped repo: override layout at <path>/.dodder/local/share/object_ids/
        parentObjectIdDir = filepath.Join(
            absParentPath,
            "."+env_dir.XDGUtilityNameDodder,
            "local", "share",
            "object_ids",
        )
    }

    parentYin := filepath.Join(
        parentObjectIdDir,
        zettel_id_provider.FilePathZettelIdYin,
    )

    parentYang := filepath.Join(
        parentObjectIdDir,
        zettel_id_provider.FilePathZettelIdYang,
    )

    if files.Exists(parentYin) {
        cmd.Genesis.BigBang.Yin = parentYin
    }

    if files.Exists(parentYang) {
        cmd.Genesis.BigBang.Yang = parentYang
    }
}
```

**Step 5: Add `makeParentRemote`**

Create the remote from parent path, handling both home and CWD-scoped repos:

``` go
func (cmd InitWorkspace) makeParentRemote(
    req command.Request,
    local *local_working_copy.Repo,
    absParentPath string,
    isHomeRepo bool,
) repo.Repo {
    var remote command_components_dodder.Remote

    if isHomeRepo {
        // Home repo parent — use standard XDG home initialization
        config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

        home, err := os.UserHomeDir()
        if err != nil {
            req.Cancel(err)
        }

        envDir := env_dir.MakeWithHomeAndInitialize(
            req,
            env_dir.XDGUtilityNameDodder,
            home,
            config.Debug,
        )

        envUI := env_ui.Make(
            req,
            config,
            config.Debug,
            env_ui.Options{},
        )

        return local_working_copy.Make(
            env_local.Make(envUI, envDir),
            local_working_copy.OptionsEmpty,
        )
    }

    // CWD-scoped parent — use override path
    remote.DirectPath = absParentPath
    return remote.MakeDirectRemoteFromPath(req, local)
}
```

**Step 6: Run `just build` to verify compilation**

Run: `just build` Expected: compilation succeeds

**Step 7: Commit**

    feat: add -parent flag to init-workspace, flip -experimental-repo default

    - Add -parent flag for CWD-scoped parent repos
    - Omitting -parent uses home XDG repo as parent
    - Flip -experimental-repo default from false to true
    - Reject -repo_id with -experimental-repo
    - Validate parent repo exists before proceeding
    - Update linkParentZettelIdProviders for both parent types

--------------------------------------------------------------------------------

### Task 2: Update `ResolveImplicitDirectPath` for empty `ParentPath`

When workspace config has empty `ParentPath` (home repo parent), push/pull need
to create a home-repo remote. The current `ResolveImplicitDirectPath` only sets
`DirectPath` which feeds into `TomlLocalOverridePathV0` --- wrong layout for
home repos.

**Files:** - Modify: `go/internal/uniform/command_components_dodder/remote.go` -
Modify: `go/internal/victor/commands_dodder/push.go` - Modify:
`go/internal/victor/commands_dodder/pull.go`

**Step 1: Add `parentIsHomeRepo` field and `MakeHomeRepoRemote` to Remote**

``` go
type Remote struct {
    RemoteRepoBlobs

    InventoryLists
    LocalWorkingCopy

    DirectPath           string
    parentIsHomeRepo     bool
    RemoteConnectionType remote_connection_types.Type
}
```

Update `ResolveImplicitDirectPath`:

``` go
func (cmd *Remote) ResolveImplicitDirectPath(
    local *local_working_copy.Repo,
) {
    if cmd.IsDirectTransfer() {
        return
    }

    parentPath := local.GetEnvWorkspace().GetParentPath()
    if parentPath != "" {
        cmd.DirectPath = parentPath
        return
    }

    // Empty ParentPath means home repo is the parent.
    // Check that a workspace config exists at all (non-workspace repos
    // won't have GetParentPath return anything meaningful).
    if local.GetEnvWorkspace().IsWorkspace() {
        cmd.parentIsHomeRepo = true
    }
}

func (cmd Remote) IsHomeRepoParent() bool {
    return cmd.parentIsHomeRepo
}

func (cmd Remote) MakeHomeRepoRemote(
    req command.Request,
) repo.Repo {
    config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

    home, err := os.UserHomeDir()
    if err != nil {
        req.Cancel(err)
    }

    envDir := env_dir.MakeWithHomeAndInitialize(
        req,
        env_dir.XDGUtilityNameDodder,
        home,
        config.Debug,
    )

    envUI := env_ui.Make(
        req,
        config,
        config.Debug,
        env_ui.Options{},
    )

    return local_working_copy.Make(
        env_local.Make(envUI, envDir),
        local_working_copy.OptionsEmpty,
    )
}
```

**Step 2: Check if `IsWorkspace` exists on env_workspace**

Look up `IsWorkspace` or equivalent. If it doesn't exist, use a different check
--- e.g., the presence of any workspace config blob. Adapt as needed based on
what's available.

**Step 3: Update push.go**

``` go
func (cmd Push) Run(req command.Request) {
    local := cmd.MakeLocalWorkingCopy(req)

    cmd.ResolveImplicitDirectPath(local)

    var remote repo.Repo

    if cmd.IsHomeRepoParent() {
        remote = cmd.MakeHomeRepoRemote(req)
    } else if cmd.IsDirectTransfer() {
        remote = cmd.MakeDirectRemoteFromPath(req, local)
    } else {
        // ... existing named remote logic unchanged ...
    }

    // ... rest unchanged ...
}
```

**Step 4: Update pull.go**

Same pattern as push.go --- add `cmd.IsHomeRepoParent()` branch.

**Step 5: Run `just build` to verify compilation**

Run: `just build` Expected: compilation succeeds

**Step 6: Commit**

    feat: support home repo parent in push/pull via ResolveImplicitDirectPath

--------------------------------------------------------------------------------

### Task 3: Update BATS tests for init-workspace changes

**Files:** - Modify: `zz-tests_bats/current_version/workspace_repo.bats`

**Step 1: Update all `init-workspace` invocations in tests**

For every test that uses `init-workspace -experimental-repo`: - Remove
`-experimental-repo` flag (now default) - Remove `-repo_id .` flag (now
implicit, and rejected if passed) - Replace `-direct "$parent_path"` with
`-parent "$parent_path"`

Example transformation for `workspace_repo_init_experimental_repo`:

``` bash
# Before:
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

# After:
run_dodder init-workspace \
    -encryption none \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -lock-internal-files=false \
    -parent "$parent_path" \
    workspace-repo-id \
    project-alpha:z
```

Apply this transformation to ALL tests: -
`workspace_repo_init_experimental_repo` -
`workspace_repo_linked_zettel_ids_from_parent` -
`workspace_repo_implicit_parent_push_pull` -
`workspace_repo_init_experimental_repo_existing_repo` -
`workspace_repo_stale_parent_path` -
`workspace_repo_experimental_repo_implies_cwd` -
`workspace_repo_init_experimental_repo_empty_query`

**Step 2: Update lightweight tests**

For `workspace_repo_clone_pull_push`, `workspace_repo_clone_filtered_by_tag`,
`workspace_repo_pull_filtered_by_tag`, and `workspace_repo_push_unfiltered` ---
these use `clone` + `init-workspace` (lightweight). The lightweight
`init-workspace` call needs `-experimental-repo=false` since the default
flipped:

``` bash
run_dodder init-workspace -experimental-repo=false
```

**Step 3: Update `workspace_repo_experimental_repo_implies_cwd`**

This test now validates that `-repo_id` is rejected with `-experimental-repo`.
Rewrite to test the error case:

``` bash
function workspace_repo_experimental_repo_implies_cwd { # @test
    parent="parent"
    bootstrap_parent "$parent"
    parent_path="$(realpath "$parent")"

    mkdir -p workspace
    pushd workspace || exit 1

    # -repo_id should be rejected with -experimental-repo (default)
    run_dodder init-workspace \
        -encryption none \
        -yin <(cat_yin) \
        -yang <(cat_yang) \
        -lock-internal-files=false \
        -repo_id . \
        -parent "$parent_path" \
        workspace-repo-id \
        project-alpha:z

    assert_failure
    assert_output --partial 'cannot be used with'
}
```

**Step 4: Add validation test for missing parent repo**

``` bash
function workspace_repo_init_missing_parent_fails { # @test
    mkdir -p workspace
    pushd workspace || exit 1

    run_dodder init-workspace \
        -encryption none \
        -yin <(cat_yin) \
        -yang <(cat_yang) \
        -lock-internal-files=false \
        -parent /nonexistent/path \
        workspace-repo-id

    assert_failure
    assert_output --partial 'no dodder repo found'
}
```

**Step 5: Run the workspace tests**

Run: `just test-bats-targets workspace_repo.bats` Expected: all tests pass

**Step 6: Commit**

    test: update workspace_repo.bats for implicit parent discovery

--------------------------------------------------------------------------------

### Task 4: Run full test suite

**Step 1: Run all tests**

Run: `just test` Expected: all tests pass

**Step 2: Fix any failures**

If tests fail, investigate and fix. Common issues: - Other test files that
invoke `init-workspace` may need updating - Completion tests in `complete.bats`
may need flag updates

**Step 3: Commit fixes if any**

    fix: address test failures from init-workspace CLI changes
