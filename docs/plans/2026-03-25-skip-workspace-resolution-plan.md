# Skip Workspace Resolution Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add opt-out builder option so `show` (and later other query-only
commands) skip the workspace store filename resolution loop in
`build_state.build()`.

**Architecture:** New `BuilderOptionSkipWorkspaceResolution` sets a flag on
`Builder` → propagated to `buildState` → when set, the workspace store loop in
`build_state.build()` is skipped entirely (all values go straight to the query
parser). The `.` sigil is already handled by the parser at line 452, so
`dotOperatorActive` is set correctly without the build loop.

**Tech Stack:** Go, dodder query system (`internal/kilo/queries`)

**Rollback:** Remove `BuilderOptionSkipWorkspaceResolution()` from `show.go`.
The builder option type is inert when unused.

**Ref:** https://github.com/amarbel-llc/dodder/issues/55

--------------------------------------------------------------------------------

### Task 1: Add BuilderOptionSkipWorkspaceResolution

**Files:** - Modify: `go/internal/kilo/queries/builder_options.go` (append after
line 176)

**Step 1: Add the builder option type**

Append to `builder_options.go` after the `builderOptionDebug` block (line 176):

``` go
type builderOptionSkipWorkspaceResolution struct{}

func BuilderOptionSkipWorkspaceResolution() builderOptionSkipWorkspaceResolution {
    return builderOptionSkipWorkspaceResolution{}
}

func (option builderOptionSkipWorkspaceResolution) Apply(builder *Builder) *Builder {
    builder.skipWorkspaceResolution = true
    return builder
}
```

**Step 2: Add field to Builder**

Modify `go/internal/kilo/queries/builder.go`. Add `skipWorkspaceResolution` to
the `Builder` struct (after line 50, the `workspaceEnabled` field):

``` go
skipWorkspaceResolution bool
```

**Step 3: Propagate flag to buildState**

Modify `go/internal/kilo/queries/build_state.go`. Add `skipWorkspaceResolution`
field to `buildState` struct (after line 34):

``` go
skipWorkspaceResolution bool
```

Wire it in `go/internal/kilo/queries/builder.go` inside
`BuildQueryGroupWithRepoId` (line 155, inside the `if builder.workspaceEnabled`
block). After the workspace store is set on `state` (line 158-160), add:

``` go
state.skipWorkspaceResolution = builder.skipWorkspaceResolution
```

Also propagate in `buildState.copy()` (`build_state.go` line 39-62) --- add
inside the copy function:

``` go
dst.skipWorkspaceResolution = src.skipWorkspaceResolution
```

**Step 4: Guard the workspace store loop**

Modify `go/internal/kilo/queries/build_state.go` line 81-119. Change the
condition from:

``` go
if buildState.workspaceStore == nil {
    remaining = values
} else {
```

to:

``` go
if buildState.workspaceStore == nil || buildState.skipWorkspaceResolution {
    remaining = values
} else {
```

**Step 5: Build to verify compilation**

Run: `just build` Expected: clean build, exit 0

**Step 6: Run tests to verify no regressions**

Run: `just test 2>&1 > /tmp/dodder-test-skip-ws.txt; echo "Exit: $?"` Expected:
exit 0, all tests pass (no command uses the option yet)

**Step 7: Commit**

    feat(queries): add BuilderOptionSkipWorkspaceResolution

    Adds a builder option that skips the workspace store filename resolution
    loop in build_state.build(). When set, all query values go directly to
    the query parser. The `.` sigil is still handled by the parser natively.

    No commands use this option yet — this is infrastructure for migrating
    commands away from unconditional workspace store consultation.

    Ref: https://github.com/amarbel-llc/dodder/issues/55

--------------------------------------------------------------------------------

### Task 2: Wire option into show command

**Files:** - Modify: `go/internal/victor/commands_dodder/show.go:85-93`

**Step 1: Add the option to show's builder options**

In `show.go`, the `Run` method builds options at lines 87-90:

``` go
query := cmd.MakeQueryIncludingWorkspace(
    req,
    pkg_query.BuilderOptions(
        pkg_query.BuilderOptionWorkspace(repo),
        pkg_query.BuilderOptionDefaultGenres(genres.Zettel),
    ),
    repo,
    args,
)
```

Add `BuilderOptionSkipWorkspaceResolution()` to the options:

``` go
query := cmd.MakeQueryIncludingWorkspace(
    req,
    pkg_query.BuilderOptions(
        pkg_query.BuilderOptionWorkspace(repo),
        pkg_query.BuilderOptionDefaultGenres(genres.Zettel),
        pkg_query.BuilderOptionSkipWorkspaceResolution(),
    ),
    repo,
    args,
)
```

**Step 2: Build**

Run: `just build` Expected: clean build

**Step 3: Run full test suite**

Run: `just test 2>&1 > /tmp/dodder-test-show-skip.txt; echo "Exit: $?"`
Expected: exit 0. The `show.bats` tests run with a workspace active (via
`setup_repo` → `run_dodder_init_workspace`), so they exercise the new code path.

**Step 4: Run show tests in isolation to confirm**

Run: `just test-bats-targets show.bats` Expected: all pass

**Step 5: Commit**

    feat(show): skip workspace store resolution for queries

    The show command now uses BuilderOptionSkipWorkspaceResolution, so query
    values like :z, !md, one/uno go directly to the query parser without
    first being tried as filesystem paths against the workspace store.

    This is the first command migrated as part of the fix for workspace
    store being consulted unconditionally for all queries.

    Ref: https://github.com/amarbel-llc/dodder/issues/55
