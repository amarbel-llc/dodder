# Migrate Remaining Callers to ExecutePlan --- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Migrate four remaining command-level callers to `ExecutePlan`, then
remove `Commit` from `sku.RepoStore` interface.

**Architecture:** Each caller transforms from manual Lock/CreateOrUpdate/Unlock
to building an `import_plan.Plan` and calling `local.ExecutePlan(plan)`. A new
`MakeLocalBuilder()` constructor separates local-mutation intent from
import-intent, skipping the stream index exists-check and Config genre filter.

**Tech Stack:** Go, LSP rename via lux MCP

**Rollback:** Each task is independently revertable by restoring the manual
Lock/CreateOrUpdate/Unlock pattern. `Commit` interface removal is revertable by
re-adding the method.

--------------------------------------------------------------------------------

### Task 1: Rename `MakeBuilder` to `MakeImportBuilder` + add `MakeLocalBuilder`

**Files:**

- Modify: `go/internal/india/import_plan/builder.go:37` (rename constructor)
- Modify: `go/internal/india/import_plan/builder.go:99-107` (make Config skip
  conditional)
- Modify: `go/internal/victor/commands_dodder/import.go:117` (update call ---
  import path)
- Modify: `go/internal/tango/user_ops/write_new_zettels.go:20` (switch to
  MakeLocalBuilder)
- Modify: `go/internal/tango/user_ops/create_from_paths.go:120` (switch to
  MakeLocalBuilder)
- Modify: `go/internal/tango/user_ops/create_from_shas.go:70` (switch to
  MakeLocalBuilder)
- Modify: `go/internal/sierra/local_working_copy/op_checkin.go:35` (switch to
  MakeLocalBuilder)

**Step 1: LSP rename `MakeBuilder` to `MakeImportBuilder`**

Use lux rename on `import_plan.MakeBuilder` at
`go/internal/india/import_plan/builder.go:37`:

    lux://lsp/rename?uri=file:///home/sasha/eng/repos/dodder/.worktrees/fair-larch/go/internal/india/import_plan/builder.go&line=36&character=5&new_name=MakeImportBuilder

This renames the definition and all 5 call sites (`import.go`,
`write_new_zettels.go`, `create_from_paths.go`, `create_from_shas.go`,
`op_checkin.go`).

**Step 2: Add `MakeLocalBuilder` constructor**

In `go/internal/india/import_plan/builder.go`, add after `MakeImportBuilder`:

``` go
func MakeLocalBuilder() Builder {
    return Builder{
        objectByKey:   make(map[string]int),
        taiByObjectId: make(map[string]ids.Tai),
        typeNameToKey: make(map[string]string),
        dedupLookup:   make(map[string]struct{}),
        abbrIndex:     store_abbr.NewInMemoryIndex(),
    }
}
```

Note: `index` is nil and `dedupFormatId` is empty --- no exists-check, no dedup.

**Step 3: Make Config genre skip conditional on import mode**

In `go/internal/india/import_plan/builder.go`, change `AddObject` lines 105-107
from:

``` go
    if genre == genres.Config {
        return
    }
```

to:

``` go
    if b.index != nil && genre == genres.Config {
        return
    }
```

Local builders (nil index) allow Config objects through. Import builders
preserve existing behavior.

**Step 4: Switch local callers to `MakeLocalBuilder`**

In each of these files, replace the `MakeImportBuilder` call (result of step 1
rename) with `MakeLocalBuilder()`, removing the stream index and dedup
arguments:

- `go/internal/tango/user_ops/write_new_zettels.go:20-23`: change
  `import_plan.MakeImportBuilder(op.GetStore().GetStreamIndex(), "")` to
  `import_plan.MakeLocalBuilder()`
- `go/internal/tango/user_ops/create_from_paths.go:120-123`: same change
- `go/internal/tango/user_ops/create_from_shas.go:70-73`: same change
- `go/internal/sierra/local_working_copy/op_checkin.go:35-38`: same change

After this, only `import.go:117` uses `MakeImportBuilder`.

**Step 5: Build and test**

Run: `just test` from repo root.

Expected: all tests pass --- this is a pure rename + constructor extraction with
no behavior change.

**Step 6: Commit**

    refactor: split MakeBuilder into MakeImportBuilder and MakeLocalBuilder

    Separates import-path intent (stream index exists-check, dedup, Config
    genre skip) from local-mutation intent (no index, no dedup, allows Config
    genre). Prepares for migrating remaining callers to ExecutePlan.

--------------------------------------------------------------------------------

### Task 2: Migrate `remote_add` to `ExecutePlan`

**Files:**

- Modify: `go/internal/victor/commands_dodder/remote_add.go:46-78`

**Step 1: Rewrite `RemoteAdd.Run` to use builder + ExecutePlan**

Replace the Lock/CreateOrUpdateDefaultProto/Unlock block (lines 66-78) with:

``` go
    builder := import_plan.MakeLocalBuilder()
    builder.AddObject(remoteObject, 0)

    plan, buildErr := builder.Build()
    if buildErr != nil {
        req.Cancel(buildErr)
    }

    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto: local.GetStore().GetProtoZettel(),
        StoreOptions: sku.StoreOptions{
            AddToInventoryList: true,
            UpdateTai:          true,
            RunHooks:           true,
            Validate:           true,
            ApplyProto:         true,
        },
    }

    if _, err := local.ExecutePlan(plan); err != nil {
        req.Cancel(err)
    }
```

Add `"code.linenisgreat.com/dodder/go/internal/india/import_plan"` to imports.

Remove the `errors` import if no longer used (check --- `errors` is still used
by the MakeRemoteAndObject pattern via `req.Cancel`).

**Step 2: Build and test**

Run: `just test` from repo root.

Expected: all tests pass.

**Step 3: Commit**

    refactor: migrate remote_add to ExecutePlan

    Replaces manual Lock/CreateOrUpdateDefaultProto/Unlock with builder-based
    plan and ExecutePlan. Part of FDR-0006 promotion work.

--------------------------------------------------------------------------------

### Task 3: Migrate `edit_config` to `ExecutePlan`

**Files:**

- Modify: `go/internal/victor/commands_dodder/edit_config.go:28-66`

**Step 1: Rewrite `EditConfig.Run` to use builder + ExecutePlan**

Replace the Reset/Lock/CreateOrUpdateDefaultProto/Unlock block (lines 48-65)
with:

``` go
    localWorkingCopy.Must(
        errors.MakeFuncContextFromFuncErr(localWorkingCopy.Reset),
    )

    builder := import_plan.MakeLocalBuilder()
    builder.AddObject(sk, 0)

    plan, buildErr := builder.Build()
    if buildErr != nil {
        localWorkingCopy.Cancel(buildErr)
    }

    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto: localWorkingCopy.GetStore().GetProtoZettel(),
        StoreOptions: sku.StoreOptions{
            AddToInventoryList: true,
            UpdateTai:          true,
            RunHooks:           true,
            Validate:           true,
        },
    }

    if _, err := localWorkingCopy.ExecutePlan(plan); err != nil {
        localWorkingCopy.Cancel(err)
    }
```

Add `"code.linenisgreat.com/dodder/go/internal/india/import_plan"` to imports.

Note: This works because Task 1 made the Config genre skip conditional --- local
builders allow Config objects through.

**Step 2: Build and test**

Run: `just test` from repo root.

Expected: all tests pass.

**Step 3: Commit**

    refactor: migrate edit_config to ExecutePlan

    Replaces manual Lock/CreateOrUpdateDefaultProto/Unlock with builder-based
    plan and ExecutePlan. Config genre objects are now allowed through
    MakeLocalBuilder. Part of FDR-0006 promotion work.

--------------------------------------------------------------------------------

### Task 4: Migrate `checkin_blob` to `ExecutePlan`

**Files:**

- Modify: `go/internal/victor/commands_dodder/checkin_blob.go:53-122`

**Step 1: Rewrite `CheckinBlob.Run` to use builder + ExecutePlan**

Replace the Lock/loop/Unlock block (lines 108-121) with:

``` go
    builder := import_plan.MakeLocalBuilder()

    for _, pair := range pairs {
        builder.AddObject(pair.object, 0)
    }

    plan, buildErr := builder.Build()
    if buildErr != nil {
        req.Cancel(buildErr)
    }

    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto: localWorkingCopy.GetStore().GetProtoZettel(),
        StoreOptions: sku.StoreOptions{
            AddToInventoryList: true,
            UpdateTai:          true,
            RunHooks:           true,
            Validate:           true,
            MergeCheckedOut:    true,
        },
    }

    if _, err := localWorkingCopy.ExecutePlan(plan); err != nil {
        req.Cancel(err)
    }
```

Add `"code.linenisgreat.com/dodder/go/internal/india/import_plan"` to imports.

**Step 2: Build and test**

Run: `just test` from repo root.

Expected: all tests pass.

**Step 3: Commit**

    refactor: migrate checkin_blob to ExecutePlan

    Replaces manual Lock/CreateOrUpdateDefaultProto/Unlock loop with
    builder-based plan and ExecutePlan. Part of FDR-0006 promotion work.

--------------------------------------------------------------------------------

### Task 5: Migrate `LockAndCommitOrganizeResults` to `ExecutePlan`

**Files:**

- Modify: `go/internal/sierra/local_working_copy/organize.go:42-96`

**Step 1: Rewrite `LockAndCommitOrganizeResults` to use builder + ExecutePlan**

Replace the entire method body (lines 44-95) with:

``` go
func (local *Repo) LockAndCommitOrganizeResults(
    results orgie.OrganizeResults,
) (changeResults orgie.Changes, err error) {
    if changeResults, err = orgie.ChangesFromResults(
        local.GetConfig().GetPrintOptions(),
        results,
    ); err != nil {
        err = errors.Wrap(err)
        return changeResults, err
    }

    count := changeResults.Changed.Len()

    if count > 30 {
        if !local.Confirm(
            fmt.Sprintf(
                "a large number (%d) of objects are being changed. continue to commit?",
                count,
            ),
            "",
        ) {
            // TODO output organize file used
            errors.ContextCancelWith499ClientClosedRequest(local)
            return changeResults, err
        }
    }

    var proto sku.Proto

    workspace := local.GetEnvWorkspace()
    workspaceType := workspace.GetDefaults().GetDefaultType()

    proto.Metadata.GetTypeMutable().ResetWithType(workspaceType)

    builder := import_plan.MakeLocalBuilder()

    for _, changed := range changeResults.Changed.AllSkuAndIndex() {
        builder.AddObject(changed.GetSkuExternal(), 0)
    }

    plan, buildErr := builder.Build()
    if buildErr != nil {
        err = errors.Wrap(buildErr)
        return changeResults, err
    }

    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto: proto,
        StoreOptions: sku.StoreOptions{
            AddToInventoryList: true,
            UpdateTai:          true,
            RunHooks:           true,
            Validate:           true,
            MergeCheckedOut:    true,
        },
    }

    if _, err = local.ExecutePlan(plan); err != nil {
        err = errors.Wrap(err)
        return changeResults, err
    }

    return changeResults, err
}
```

Add `"code.linenisgreat.com/dodder/go/internal/india/import_plan"` to imports.

Key changes:

- Confirmation prompt moves before the lock (before `ExecutePlan`)
- Manual Lock/Unlock replaced by `ExecutePlan` (which handles Lock/Unlock)
- `CreateOrUpdate` options are explicitly set on `DefaultCommitOptions` instead
  of relying on the wrapper

**Step 2: Build and test**

Run: `just test` from repo root.

Expected: all tests pass.

**Step 3: Commit**

    refactor: migrate LockAndCommitOrganizeResults to ExecutePlan

    Replaces manual Lock/CreateOrUpdate/Unlock loop with builder-based plan
    and ExecutePlan. Confirmation prompt now runs before lock acquisition.
    Part of FDR-0006 promotion work.

--------------------------------------------------------------------------------

### Task 6: Remove `Commit` from `sku.RepoStore` interface

**Files:**

- Modify: `go/internal/golf/sku/store.go:50-57`

**Step 1: Remove `Commit` from `RepoStore` interface**

Change lines 50-57 from:

``` go
    RepoStore interface {
        Commit(*Transacted, CommitOptions) (err error)
        ReadOneInto(domain_interfaces.ObjectId, *Transacted) (err error)
        ReadPrimitiveQuery(
            qg PrimitiveQueryGroup,
            w interfaces.FuncIter[*Transacted],
        ) (err error)
    }
```

to:

``` go
    RepoStore interface {
        ReadOneInto(domain_interfaces.ObjectId, *Transacted) (err error)
        ReadPrimitiveQuery(
            qg PrimitiveQueryGroup,
            w interfaces.FuncIter[*Transacted],
        ) (err error)
    }
```

**Step 2: Fix any compilation errors**

The concrete `*Store` type still has a `Commit` method --- it just no longer
satisfies this interface method. Check for any code that calls `Commit` through
the `RepoStore` interface rather than the concrete type:

- `execute_plan.go:34` calls `local.GetStore().Commit(...)` --- if `GetStore()`
  returns `RepoStore`, this will fail. Check the return type. If it returns the
  concrete `*Store`, no change needed. If it returns `RepoStore`, change it to
  go through the concrete type.
- `store_fs/primitives.go:86` calls `store.storeSupplies.Commit(...)` --- check
  if `storeSupplies` is typed as `RepoStore`. If so, this needs a different
  interface or concrete type reference.
- `remote_transfer/committer.go:35` --- check if `storeObject` is typed as
  `RepoStore`.

For each broken call site: if the caller has access to the concrete `*Store`,
use it. If it goes through an interface, add a narrower `Committer` interface
where needed:

``` go
type Committer interface {
    Commit(*Transacted, CommitOptions) (err error)
}
```

**Step 3: Build and test**

Run: `just test` from repo root.

Expected: all tests pass.

**Step 4: Commit**

    refactor: remove Commit from sku.RepoStore interface

    All external callers now use ExecutePlan. Commit remains as a concrete
    method on *Store for internal use (ExecutePlan, RevertTo,
    CreateOrUpdateBlobDigest, CreateOrUpdateCheckedOut, store_fs, and
    remote_transfer). Completes FDR-0006 promotion criteria.

--------------------------------------------------------------------------------

### Task 7: Update FDR-0006 status

**Files:**

- Modify: `docs/features/0006-two-stage-commit.md`

**Step 1: Update status and implementation sections**

- Change `status: experimental` to `status: testing` in the YAML front matter
- Update the Phase 3 "Out of Scope" section to note these items are now complete
- Add a note to the Implementation Status section listing the four new
  migrations

**Step 2: Commit**

    docs: promote FDR-0006 to testing status

    All local mutation callers now use ExecutePlan. Commit removed from
    sku.RepoStore interface. Promotion criteria met.
