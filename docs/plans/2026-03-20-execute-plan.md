# ExecutePlan Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `ExecutePlan` to `repo.LocalRepo` interface so all local mutations
commit through plans, then migrate all existing callers including Checkin.

**Architecture:** `import_plan.Plan` gains `DefaultCommitOptions`;
`import_plan.Entry` gains optional per-entry `*CommitOptions`.
`local_working_copy.Repo` implements `ExecutePlan` which acquires the lock,
iterates committable entries, calls `store.Commit` directly with resolved
options, releases the lock. Existing `tango/user_ops.CommitPlan` is deleted
after all callers migrate.

**Tech Stack:** Go, dodder internal packages (import_plan, sku, repo,
local_working_copy, user_ops)

**Spec:** `docs/features/0006-two-stage-commit.md` (Phase 3 section)

--------------------------------------------------------------------------------

## File Map

  -------------------------------------------------------------------------------------------------------------
  File                                                      Action   Responsibility
  --------------------------------------------------------- -------- ------------------------------------------
  `go/internal/india/import_plan/plan.go`                   Modify   Add
                                                                     `DefaultCommitOptions sku.CommitOptions`
                                                                     field

  `go/internal/india/import_plan/entry.go`                  Modify   Add `Options *sku.CommitOptions` field

  `go/internal/quebec/repo/main.go`                         Modify   Add `ExecutePlan` to `LocalRepo` interface

  `go/internal/sierra/local_working_copy/execute_plan.go`   Create   Implement `ExecutePlan` on `Repo`

  `go/internal/tango/user_ops/write_new_zettels.go`         Modify   Replace `CommitPlan` with
                                                                     `local.ExecutePlan`

  `go/internal/tango/user_ops/create_from_shas.go`          Modify   Replace `CommitPlan` with
                                                                     `local.ExecutePlan`

  `go/internal/tango/user_ops/create_from_paths.go`         Modify   Replace `CommitPlan` with
                                                                     `local.ExecutePlan`

  `go/internal/tango/user_ops/commit_plan.go`               Delete   Replaced by
                                                                     `local_working_copy.Repo.ExecutePlan`

  `go/internal/sierra/local_working_copy/op_checkin.go`     Modify   Migrate to plan-based (pre-plan / execute
                                                                     / post-plan)
  -------------------------------------------------------------------------------------------------------------

--------------------------------------------------------------------------------

## Conventions

- **Build:** `just build` from repo root (builds debug + release binaries)
- **Unit tests:** `just test-go` from repo root
- **Integration tests:** `just test-bats` from repo root (builds first)
- **Specific bats file:** `just test-bats-targets <file>.bats` from repo root
- **Commit:** Use `grit try_commit` MCP tool, never `git commit` directly
- **Never dereference `sku.Transacted` pointers** --- use `ResetWith` patterns
- **Error wrapping:** Always use `errors.Wrap(err)` or `errors.Wrapf(err, ...)`

--------------------------------------------------------------------------------

### Task 1: Add `DefaultCommitOptions` to Plan and `Options` to Entry

**Files:** - Modify: `go/internal/india/import_plan/plan.go:8-13` - Modify:
`go/internal/india/import_plan/entry.go:8-15`

- [ ] **Step 1: Add `DefaultCommitOptions` field to `Plan` struct**

In `go/internal/india/import_plan/plan.go`, add the import and field:

``` go
import (
    "code.linenisgreat.com/dodder/go/internal/alfa/genres"
    "code.linenisgreat.com/dodder/go/internal/bravo/ids"
    "code.linenisgreat.com/dodder/go/internal/golf/sku"
)

type Plan struct {
    Entries              []Entry
    SourcePaths          []string
    HasErrors            bool
    Abbr                 ids.Abbr
    DefaultCommitOptions sku.CommitOptions
}
```

- [ ] **Step 2: Add `Options` field to `Entry` struct**

In `go/internal/india/import_plan/entry.go`:

``` go
type Entry struct {
    object         sku.Transacted
    Classification Classification
    SourceIndex    int
    Height         int
    OriginalTai    ids.Tai
    ErrorCause     string
    Options        *sku.CommitOptions
}
```

- [ ] **Step 3: Build to verify compilation**

Run: `just build` from repo root. Expected: Clean build, no errors. The new
fields have zero values so no existing code breaks.

- [ ] **Step 4: Run unit tests**

Run: `just test-go` Expected: All tests pass (zero-value fields don't change
behavior).

- [ ] **Step 5: Commit**

Message:
`feat: add DefaultCommitOptions to Plan, Options to Entry (FDR-0006 Phase 3)`
Files: `go/internal/india/import_plan/plan.go`,
`go/internal/india/import_plan/entry.go`

--------------------------------------------------------------------------------

### Task 2: Add `ExecutePlan` to `LocalRepo` interface and implement it

**Files:** - Modify: `go/internal/quebec/repo/main.go:59-69` - Create:
`go/internal/sierra/local_working_copy/execute_plan.go`

- [ ] **Step 1: Add `ExecutePlan` to the `LocalRepo` interface**

In `go/internal/quebec/repo/main.go`, add the import and method:

``` go
import (
    // ... existing imports ...
    "code.linenisgreat.com/dodder/go/internal/india/import_plan"
)

type LocalRepo interface {
    Repo

    GetEnvRepo() env_repo.Env
    GetImmutableConfigPrivate() genesis_configs.TypedConfigPrivate

    Lock() error
    Unlock() error

    GetEnvWorkspace() env_workspace.Env

    ExecutePlan(plan *import_plan.Plan) (sku.TransactedMutableSet, error)
}
```

- [ ] **Step 2: Create `execute_plan.go` with the implementation**

Create `go/internal/sierra/local_working_copy/execute_plan.go`:

``` go
package local_working_copy

import (
    "code.linenisgreat.com/dodder/go/internal/golf/sku"
    "code.linenisgreat.com/dodder/go/internal/india/import_plan"
    "code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func (local *Repo) ExecutePlan(
    plan *import_plan.Plan,
) (results sku.TransactedMutableSet, err error) {
    if err = local.Lock(); err != nil {
        err = errors.Wrap(err)
        return results, err
    }

    results = sku.MakeTransactedMutableSet()

    for i := range plan.Entries {
        entry := &plan.Entries[i]

        if !entry.Classification.IsCommittable() {
            continue
        }

        options := plan.DefaultCommitOptions

        if entry.Options != nil {
            options = *entry.Options
        }

        object := entry.GetObject()

        if err = local.GetStore().Commit(object, options); err != nil {
            err = errors.Wrap(err)
            return results, err
        }

        if err = results.Add(object); err != nil {
            err = errors.Wrap(err)
            return results, err
        }
    }

    if err = local.Unlock(); err != nil {
        err = errors.Wrap(err)
        return results, err
    }

    return results, err
}
```

- [ ] **Step 3: Build to verify compilation**

Run: `just build` from repo root. Expected: Clean build.
`local_working_copy.Repo` already satisfies `LocalRepo` --- this adds one more
method.

- [ ] **Step 4: Run unit tests**

Run: `just test-go` Expected: All pass.

- [ ] **Step 5: Commit**

Message: `feat: add ExecutePlan to LocalRepo interface (FDR-0006 Phase 3)`
Files: `go/internal/quebec/repo/main.go`,
`go/internal/sierra/local_working_copy/execute_plan.go`

--------------------------------------------------------------------------------

### Task 3: Migrate `WriteNewZettels` to `ExecutePlan`

**Files:** - Modify: `go/internal/tango/user_ops/write_new_zettels.go`

- [ ] **Step 1: Replace `CommitPlan` call with plan options + `ExecutePlan`**

The current code calls
`CommitPlan(op.Repo, plan, sku.StoreOptions{ApplyProto: true})`.

`CommitPlan` internally calls `store.CreateOrUpdateDefaultProto` which sets:
`AddToInventoryList=true, UpdateTai=true, RunHooks=true, Validate=true` plus
`Proto=store.protoZettel`. Since `ExecutePlan` calls `store.Commit` directly, we
must set all these on the plan's `DefaultCommitOptions`.

**Proto source:** The current code uses `store.protoZettel` (workspace defaults)
via `CreateOrUpdateDefaultProto`. The caller's `proto` parameter is used only to
*create* objects (e.g. `proto.Make()` or `proto.Apply()`), not as the commit
proto. Use `op.GetStore().GetProtoZettel()` (the public accessor for
`store.protoZettel`) to match current behavior exactly.

Replace `CommitPlan` call in `RunMany`:

``` go
func (op WriteNewZettels) RunMany(
    proto sku.Proto,
    count int,
) (results sku.TransactedMutableSet, err error) {
    zettelIdIndex := op.GetStore().GetZettelIdIndex()

    builder := import_plan.MakeBuilder(
        op.GetStore().GetStreamIndex(),
        "",
    )

    builder.AddTransform(
        import_plan.MakeAllocateZettelIdTransform(zettelIdIndex),
    )

    for range count {
        object, _ := proto.Make() //repool:owned

        builder.AddObject(object, 0)
    }

    plan, buildErr := builder.Build()
    if buildErr != nil {
        err = errors.Wrap(buildErr)
        return results, err
    }

    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto: op.GetStore().GetProtoZettel(),
        StoreOptions: sku.StoreOptions{
            AddToInventoryList: true,
            UpdateTai:          true,
            RunHooks:           true,
            Validate:           true,
            ApplyProto:         true,
        },
    }

    results, err = op.Repo.ExecutePlan(plan)

    return results, err
}
```

Remove the `sku.StoreOptions` import if it was only used for `CommitPlan`. The
`"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"` import
can also be removed since `op.Repo` satisfies `LocalRepo`.

**Important:** Check whether `op.Repo` (which is `*local_working_copy.Repo`)
satisfies the `ExecutePlan` method. It does --- we added it in Task 2. But the
call must go through the concrete type, not the interface, since `op.Repo` is
`*local_working_copy.Repo`. This is fine.

- [ ] **Step 2: Build to verify compilation**

Run: `just build` from repo root. Expected: Clean build.

- [ ] **Step 3: Run integration tests for `new` command**

Run: `just test-bats-targets new.bats` from repo root. Expected: All pass. The
behavior should be identical --- same objects committed with the same options,
just through `ExecutePlan` instead of `CommitPlan`.

- [ ] **Step 4: Commit**

Message: `refactor: migrate WriteNewZettels to ExecutePlan (FDR-0006 Phase 3)`
Files: `go/internal/tango/user_ops/write_new_zettels.go`

--------------------------------------------------------------------------------

### Task 4: Migrate `CreateFromShas` to `ExecutePlan`

**Files:** - Modify: `go/internal/tango/user_ops/create_from_shas.go`

- [ ] **Step 1: Replace `CommitPlan` call with plan options + `ExecutePlan`**

Same pattern as WriteNewZettels. Use `op.GetStore().GetProtoZettel()` for the
proto (not `op.Proto`, which is used only to create objects before planning).
Replace lines 91-95:

``` go
    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto: op.GetStore().GetProtoZettel(),
        StoreOptions: sku.StoreOptions{
            AddToInventoryList: true,
            UpdateTai:          true,
            RunHooks:           true,
            Validate:           true,
            ApplyProto:         true,
        },
    }

    results, err = op.Repo.ExecutePlan(plan)
```

Remove unused imports (`local_working_copy` if present).

- [ ] **Step 2: Build and run unit tests**

Run: `just build && just test-go` Expected: Clean build, all unit tests pass.

- [ ] **Step 3: Commit**

Message: `refactor: migrate CreateFromShas to ExecutePlan (FDR-0006 Phase 3)`
Files: `go/internal/tango/user_ops/create_from_shas.go`

--------------------------------------------------------------------------------

### Task 5: Migrate `CreateFromPaths` to `ExecutePlan`

**Files:** - Modify: `go/internal/tango/user_ops/create_from_paths.go`

- [ ] **Step 1: Replace `CommitPlan` call with plan options + `ExecutePlan`**

This caller passes different `StoreOptions` (includes `AllowTagFailure`). Use
`op.GetStore().GetProtoZettel()` for the proto (not `op.Proto`, which is used
only to create objects before planning). Replace lines 149-158:

``` go
    plan.DefaultCommitOptions = sku.CommitOptions{
        Proto: op.GetStore().GetProtoZettel(),
        StoreOptions: sku.StoreOptions{
            LockfileOptions: sku.LockfileOptions{
                AllowTagFailure: true,
            },
            AddToInventoryList: true,
            UpdateTai:          true,
            RunHooks:           true,
            Validate:           true,
            ApplyProto:         true,
        },
    }

    results, err = op.Repo.ExecutePlan(plan)
```

Remove unused imports.

**Important:** Preserve the `toDelete` file-deletion loop (current lines
160-175) after the `ExecutePlan` call. This loop iterates `toDelete.All()`,
deletes files via `op.GetEnvRepo().Delete()`, and prints deletion messages. It
must remain unchanged --- it runs *after* `ExecutePlan` completes and the lock
is released.

- [ ] **Step 2: Build and run integration tests for `new` with paths**

Run: `just build && just test-bats-targets new.bats` Expected: All pass.

- [ ] **Step 3: Commit**

Message: `refactor: migrate CreateFromPaths to ExecutePlan (FDR-0006 Phase 3)`
Files: `go/internal/tango/user_ops/create_from_paths.go`

--------------------------------------------------------------------------------

### Task 6: Delete `tango/user_ops.CommitPlan`

**Files:** - Delete: `go/internal/tango/user_ops/commit_plan.go`

- [ ] **Step 1: Verify no remaining callers**

Run from repo root:

    grep -r 'CommitPlan(' go/internal/tango/user_ops/ --include='*.go'

Expected: Only the definition in `commit_plan.go` remains (no callers).

- [ ] **Step 2: Delete the file**

Delete `go/internal/tango/user_ops/commit_plan.go`.

- [ ] **Step 3: Build and run full test suite**

Run: `just test` from repo root. Expected: Clean build, all unit and integration
tests pass. No code references `CommitPlan` anymore.

- [ ] **Step 4: Commit**

Message:
`refactor: delete user_ops.CommitPlan, replaced by LocalRepo.ExecutePlan (FDR-0006 Phase 3)`
Files: `go/internal/tango/user_ops/commit_plan.go` (deleted)

--------------------------------------------------------------------------------

### Task 7: Migrate Checkin to plan-based ExecutePlan

**Files:** - Modify: `go/internal/sierra/local_working_copy/op_checkin.go`

This is the most complex migration. Read the current file carefully before
making changes. The current code has two phases: pre-lock ID allocation and a
mixed commit loop under lock. We decompose into three phases.

**Current Checkin signature and behavior to preserve:** - Takes
`skus sku.SkuTypeSetMutable`, `proto sku.Proto`, `delete bool`,
`refreshCheckout bool` - Returns committed objects as
`sku.TransactedMutableSet` - For untracked Zettel/Blob with non-empty metadata:
allocates ID, applies proto, commits via `CreateOrUpdate` - For tracked objects:
commits via `CreateOrUpdateCheckedOut` (which calls `Commit` with
`StoreOptionsCreate` then `UpdateCheckoutFromCheckedOut`) - Optionally deletes
checked-out files after commit

- [ ] **Step 1: Rewrite Checkin with three-phase structure**

``` go
package local_working_copy

import (
    "code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
    "code.linenisgreat.com/dodder/go/internal/alfa/genres"
    "code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
    "code.linenisgreat.com/dodder/go/internal/golf/sku"
    "code.linenisgreat.com/dodder/go/internal/india/import_plan"
    "code.linenisgreat.com/dodder/go/internal/november/env_workspace"
    "code.linenisgreat.com/dodder/go/lib/bravo/errors"
    "code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func (local *Repo) Checkin(
    skus sku.SkuTypeSetMutable,
    proto sku.Proto,
    delete bool,
    refreshCheckout bool,
) (processed sku.TransactedMutableSet, err error) {
    processed = sku.MakeTransactedMutableSet()
    sortedResults := quiter.ElementsSorted(
        skus,
        func(left, right sku.SkuType) bool {
            return left.String() < right.String()
        },
    )

    // Side map for post-plan correlation: object ID string -> CheckedOut
    checkedOutByObjectId := make(map[string]sku.SkuType)

    // Pre-plan phase (no lock)

    zettelIdIndex := local.GetStore().GetZettelIdIndex()

    builder := import_plan.MakeBuilder(
        local.GetStore().GetStreamIndex(),
        "",
    )

    for _, co := range sortedResults {
        if refreshCheckout {
            if err = local.GetEnvWorkspace().GetStoreFS().RefreshCheckedOut(
                co,
            ); err != nil {
                err = errors.Wrap(err)
                return processed, err
            }
        }

        external := co.GetSkuExternal()

        if co.GetState() == checked_out_state.Untracked &&
            (external.GetGenre() == genres.Zettel ||
                external.GetGenre() == genres.Blob) {
            if external.GetMetadata().IsEmpty() {
                continue
            }

            external.GetObjectIdMutable().Reset()

            zettelId, idErr := zettelIdIndex.CreateZettelId()
            if idErr != nil {
                err = errors.Wrap(idErr)
                return processed, err
            }

            if err = external.GetObjectIdMutable().SetWithSeq(
                zettelId.ToSeq(),
            ); err != nil {
                err = errors.Wrap(err)
                return processed, err
            }

            if err = local.GetStore().UpdateTransactedFromBlobs(
                co,
            ); err != nil {
                if errors.Is(err, env_workspace.ErrUnsupportedOperation{}) {
                    err = nil
                } else {
                    err = errors.Wrap(err)
                    return processed, err
                }
            }

            proto.Apply(external, genres.Zettel)

            untrackedOptions := sku.CommitOptions{
                Proto: proto,
                StoreOptions: sku.GetStoreOptionsCreate(),
            }

            builder.AddObject(external, 0)

            // Set per-entry options after AddObject appends the entry
            entries := builder.PeekEntries()
            entries[len(entries)-1].Options = &untrackedOptions
        } else {
            trackedOptions := sku.CommitOptions{
                StoreOptions: sku.GetStoreOptionsCreate(),
            }

            builder.AddObject(external, 0)

            entries := builder.PeekEntries()
            entries[len(entries)-1].Options = &trackedOptions

            // Track for post-plan checkout update
            checkedOutByObjectId[external.GetObjectId().String()] = co
        }
    }

    plan, buildErr := builder.Build()
    if buildErr != nil {
        err = errors.Wrap(buildErr)
        return processed, err
    }

    // Execute phase
    results, execErr := local.ExecutePlan(plan)
    if execErr != nil {
        err = errors.Wrap(execErr)
        return processed, err
    }

    // Post-plan phase (no lock): checkout updates and deletes

    for committedObject := range results.All() {
        objectIdStr := committedObject.GetObjectId().String()

        if co, ok := checkedOutByObjectId[objectIdStr]; ok && !delete {
            if err = local.GetStore().UpdateCheckoutFromCheckedOut(
                checkout_options.OptionsWithoutMode{Force: true},
                co,
            ); err != nil {
                err = errors.Wrap(err)
                return processed, err
            }
        }

        if delete {
            // Find the original CheckedOut for deletion
            for _, co := range sortedResults {
                if co.GetSkuExternal().GetObjectId().String() == objectIdStr {
                    if err = local.GetStore().DeleteCheckedOut(co); err != nil {
                        err = errors.Wrap(err)
                        return processed, err
                    }

                    cloned, _ := co.GetSkuExternal().CloneTransacted() //repool:owned
                    if err = processed.Add(cloned); err != nil {
                        err = errors.Wrap(err)
                        return processed, err
                    }

                    break
                }
            }
        }
    }

    return processed, err
}
```

**Important notes for the implementor:**

1.  The `builder.PeekEntries()` method may not exist yet. Check if `Builder`
    exposes its entries slice. If not, you'll need to add a `PeekEntries` method
    to the builder, OR set per-entry options on the plan after `Build()` by
    matching objects. The simpler approach is to add a `PeekEntries() []Entry`
    method to `Builder`.

2.  `UpdateCheckoutFromCheckedOut` takes
    `(checkout_options.OptionsWithoutMode, sku.SkuType)` --- verify this
    signature by reading `go/internal/papa/store/create.go:175`. The
    `&& !delete` guard matches current behavior:
    `CreateOrUpdateCheckedOut(co, !delete)` passes `!delete` as the
    `updateCheckout` bool, so when deleting, checkout update is skipped.

3.  The `delete` path needs to correlate committed objects back to the original
    `CheckedOut` objects. The side map approach uses object ID strings as keys.

4.  The current code adds to `processed` only when `delete` is true. Preserve
    this behavior.

- [ ] **Step 2: Check if `Builder.PeekEntries` exists, add if needed**

Search for `PeekEntries` in `go/internal/india/import_plan/builder.go`. If it
doesn't exist, add:

``` go
func (b *Builder) PeekEntries() []Entry {
    return b.entries
}
```

Alternatively, if setting per-entry options after `Build()` is simpler, iterate
`plan.Entries` and match by object ID to set `Options`. This avoids modifying
the builder.

- [ ] **Step 3: Build to verify compilation**

Run: `just build` from repo root. Expected: Clean build.

- [ ] **Step 4: Run checkin integration tests**

Run: `just test-bats-targets checkin.bats` from repo root. Expected: All pass.
The behavior must be identical to the old mixed loop.

Also run: `just test-bats-targets organize.bats` (organize calls checkin
internally in some paths).

- [ ] **Step 5: Run the full test suite**

Run: `just test` from repo root. Expected: All unit and integration tests pass.

- [ ] **Step 6: Commit**

Message:
`refactor: migrate Checkin to plan-based ExecutePlan (FDR-0006 Phase 3)` Files:
`go/internal/sierra/local_working_copy/op_checkin.go` (and builder.go if
PeekEntries was added)

--------------------------------------------------------------------------------

### Task 8: Update FDR-0006 implementation status

**Files:** - Modify: `docs/features/0006-two-stage-commit.md`

- [ ] **Step 1: Update Phase 2 checklist**

Mark the Checkin line as complete:

    - [x] Migrate `Checkin` (deferred to Phase 3 — see Checkin decomposition)

- [ ] **Step 2: Update Phase 3 status**

Change the Phase 3 heading from:

    ### Phase 3: Repo Executes Plan (Not Started)

to:

    ### Phase 3: Repo Executes Plan (Complete)

- [ ] **Step 3: Commit**

Message: `docs: mark FDR-0006 Phase 3 complete` Files:
`docs/features/0006-two-stage-commit.md`
