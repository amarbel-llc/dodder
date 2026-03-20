# Migrate CreateFromShas and CreateFromPaths to Builder + CommitPlan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Migrate `CreateFromShas` and `CreateFromPaths` from inline Phase 1/Phase 2 loops to `import_plan.Builder` + `MakeAllocateZettelIdTransform` + local `CommitPlan`, continuing FDR-0006 Phase 2.

**Architecture:** Replace the inline ID pre-allocation loops and lock/commit/unlock loops with Builder construction + transform + CommitPlan. Pre-processing (arg parsing, digest dedup, already-checked-in checks, file reading) stays before Builder. Post-commit work (file deletion in CreateFromPaths) moves after CommitPlan.

**Tech Stack:** Go, dodder internal packages (india/import_plan, tango/user_ops)

**Rollback:** Revert to the current inline Phase 1/Phase 2 loops. Both are independently correct under LockSmith.

---

### Task 1: Migrate CreateFromShas to Builder + CommitPlan

**Promotion criteria:** N/A — replaces inline plan directly.

**Files:**
- Modify: `go/internal/tango/user_ops/create_from_shas.go:69-113`

**Step 1: Rewrite Phase 1 + Phase 2 to use Builder + CommitPlan**

The pre-processing (lines 19-67) stays unchanged — it builds the `toCreate` map
with deduplication and already-checked-in checks, applies proto, and sets blob
digests. Only the Phase 1 (lines 69-85) and Phase 2 (lines 87-113) change.

Replace everything from line 69 (`results = sku.MakeTransactedMutableSet()`)
through the end of the function with:

```go
	builder := import_plan.MakeBuilder(
		op.GetStore().GetStreamIndex(),
		"",
	)

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(
			op.GetStore().GetZettelIdIndex(),
		),
	)

	for _, object := range toCreate {
		builder.AddObject(object, 0)
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return results, err
	}

	results, err = CommitPlan(
		op.Repo,
		plan,
		sku.StoreOptions{ApplyProto: true},
	)

	return results, err
```

**Step 2: Update imports**

Add `"code.linenisgreat.com/dodder/go/internal/india/import_plan"`. Remove
`"code.linenisgreat.com/dodder/go/lib/charlie/ui"` only if it's no longer used
— check that `ui.Err().Printf` calls on lines 43 and 51 still need it. (They
do — `ui` is used for the "already checked in" and "duplicate" warnings in
pre-processing.) So keep `ui`.

**Step 3: Verify it compiles**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/smooth-rowan/go && go build ./internal/tango/user_ops/...`
Expected: success

**Step 4: Run relevant tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/smooth-rowan && just test-bats-targets current_version/new.bats`
Expected: all tests pass. The `use_blob_digests` test exercises the `-shas` path.

**Step 5: Commit**

```
feat: migrate CreateFromShas to Builder + CommitPlan (FDR-0006)
```

---

### Task 2: Migrate CreateFromPaths to Builder + CommitPlan

**Promotion criteria:** N/A — replaces inline plan directly.

**Files:**
- Modify: `go/internal/tango/user_ops/create_from_paths.go:121-192`

**Step 1: Rewrite Phase 1 + Phase 2 to use Builder + CommitPlan**

The pre-processing (lines 26-119) stays unchanged — it reads objects from files,
calculates digests, deduplicates, and collects files to delete. Only the Phase 1
(lines 121-141) and Phase 2 (lines 143-191) change.

There are two complications vs CreateFromShas:

1. **Empty metadata skip**: The current Phase 1 skips objects with
   `object.GetMetadata().IsEmpty()` (line 127). The transform handles this
   naturally — objects with empty metadata will have empty IDs, and the transform
   allocates IDs for all objects with empty IDs. But we need to skip empty-metadata
   objects entirely, not allocate IDs for them. Add a filtering transform before
   the ID allocation transform.

2. **Proto apply + LockfileOptions**: The current Phase 2 calls
   `op.Proto.Apply(object, genres.Zettel)` and passes
   `LockfileOptions{AllowTagFailure: true}`. The proto apply happens before
   `CreateOrUpdateDefaultProto` — since Builder copies the object, we should
   apply proto before `AddObject`. The LockfileOptions pass through CommitPlan's
   `storeOptions` parameter.

3. **Delete loop**: The delete loop (lines 173-184) runs inside the lock. Move
   it after CommitPlan returns — files can be deleted after commits succeed. This
   requires the lock to still be held. Since CommitPlan unlocks, the delete loop
   must happen after unlock. File deletion after commit success is safe — the
   objects are persisted.

Replace everything from line 120 (`results = sku.MakeTransactedMutableSet()`)
through the end of the function with:

```go
	builder := import_plan.MakeBuilder(
		op.GetStore().GetStreamIndex(),
		"",
	)

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(
			op.GetStore().GetZettelIdIndex(),
		),
	)

	for _, object := range toCreate {
		if object.GetMetadata().IsEmpty() {
			continue
		}

		op.Proto.Apply(object, genres.Zettel)

		builder.AddObject(object, 0)
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return results, err
	}

	results, err = CommitPlan(
		op.Repo,
		plan,
		sku.StoreOptions{
			LockfileOptions: sku.LockfileOptions{
				AllowTagFailure: true,
			},
			ApplyProto: true,
		},
	)

	if err != nil {
		return results, err
	}

	for fdToDelete := range toDelete.All() {
		// TODO-P2 move to checkout store
		if err = op.GetEnvRepo().Delete(fdToDelete.GetPath()); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		pathRel := op.GetEnvRepo().RelToCwdOrSame(fdToDelete.GetPath())

		// TODO-P2 move to printer
		op.GetUI().Printf("[%s] (deleted)", pathRel)
	}

	return results, err
```

**Step 2: Update imports**

Add `"code.linenisgreat.com/dodder/go/internal/india/import_plan"`.
Remove `"code.linenisgreat.com/dodder/go/internal/bravo/markl"` only if no
longer used — check that `markl.GetId()` and `markl.PurposeV5MetadataDigestWithoutTai`
and `markl.AssertIdIsNotNull` are still used in pre-processing. (They are.)
So keep all existing imports, just add `import_plan`.

**Step 3: Verify it compiles**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/smooth-rowan/go && go build ./internal/tango/user_ops/...`
Expected: success

**Step 4: Run relevant tests**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/smooth-rowan && just test-bats-targets current_version/new.bats`
Expected: all tests pass. The `new_zettel_file`, `new_zettel_stdin`, and
`can_duplicate_zettel_content` tests exercise the file-path code path.

**Step 5: Run full test suite**

Run: `cd /home/sasha/eng/repos/dodder/.worktrees/smooth-rowan && just test`
Expected: all tests pass.

**Step 6: Commit**

```
feat: migrate CreateFromPaths to Builder + CommitPlan (FDR-0006)
```

---

### Task 3: Update FDR-0006 Phase 2 checklist

**Promotion criteria:** N/A

**Files:**
- Modify: `docs/features/0006-two-stage-commit.md`

**Step 1: Update Phase 2 checklist**

Change the checklist to:

```markdown
### Phase 2: Builder Unification (In Progress)

- [x] Write `MakeAllocateZettelIdTransform` for zettel ID allocation
- [x] Write local `CommitPlan` function in `tango/user_ops`
- [x] Migrate `WriteNewZettels` (`new` zero-arg) to Builder + CommitPlan
- [x] Migrate `CreateFromShas` (`new -shas`) to Builder + CommitPlan
- [x] Migrate `CreateFromPaths` (`new` with file args) to Builder + CommitPlan
- [ ] Migrate `Checkin` (deferred — mixed update/create loop needs richer commit executor)
```

**Step 2: Commit**

```
docs: update FDR-0006 Phase 2 checklist with completed migrations
```

---

## Considerations

### Proto application timing for CreateFromPaths

The current code applies `op.Proto.Apply(object, genres.Zettel)` inside the
locked Phase 2 loop (line 154). In the migrated version, proto is applied
before `builder.AddObject` — before the lock. This is safe because proto
application is a pure in-memory mutation (sets type, tags from proto). It doesn't
touch the store or require the lock.

### Delete loop moved after CommitPlan

The current code deletes source files inside the locked section. The migrated
version deletes after `CommitPlan` returns (after unlock). This is safe:
- If CommitPlan succeeds, the objects are persisted — deleting source files is
  the correct outcome
- If CommitPlan fails, we return early before the delete loop — source files
  are preserved

### Why Checkin is deferred

Checkin's Phase 2 loop dispatches to two different commit methods based on
checked-out state (`CreateOrUpdate` for untracked, `CreateOrUpdateCheckedOut`
for tracked), interleaves `RefreshCheckedOut` and `UpdateTransactedFromBlobs`
per-object, and handles delete conditionally. This mixed dispatch doesn't map
to CommitPlan's uniform "iterate committable entries" model. Migrating it
requires either a richer commit executor or splitting the loop into separate
Builder-backed and direct paths, which would increase complexity rather than
reduce it.
