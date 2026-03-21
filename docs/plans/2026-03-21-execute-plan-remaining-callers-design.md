# Migrate Remaining Callers to ExecutePlan

Date: 2026-03-21

## Context

FDR-0006 (Two-Stage Commit) Phase 3 added `ExecutePlan` to `LocalRepo` and
migrated `WriteNewZettels`, `CreateFromPaths`, `CreateFromShas`, and `Checkin`.
Four command-level callers still use manual Lock → `CreateOrUpdate` → Unlock:

1.  `remote_add.go` --- single object via `CreateOrUpdateDefaultProto`
2.  `edit_config.go` --- single object via `CreateOrUpdateDefaultProto`
3.  `checkin_blob.go` --- loop via `CreateOrUpdateDefaultProto`
4.  `LockAndCommitOrganizeResults` --- loop via `CreateOrUpdate`

All four must migrate before `Commit` can be removed from `sku.RepoStore`.

## Design

### Builder Constructor Split

`MakeBuilder(index, dedupFormatId)` serves two distinct intents that should be
separate constructors:

- **`MakeImportBuilder(index, dedupFormatId)`** --- existing behavior: stream
  index for exists-check, dedup by format. Used by remote transfer path.
- **`MakeLocalBuilder()`** --- nil index, no dedup. Everything classifies as
  committable. Used by all local mutation commands.

Rename existing `MakeBuilder` → `MakeImportBuilder` via LSP rename. Add
`MakeLocalBuilder` calling the same internal init with nil/empty. Migrate
existing local callers (`WriteNewZettels`, `CreateFromPaths`, `CreateFromShas`,
`Checkin`) from `MakeBuilder(store.GetStreamIndex(), "")` to
`MakeLocalBuilder()`.

### Per-Caller Migration

Each caller transforms from manual Lock/Unlock to `ExecutePlan`:

**`remote_add.go`:** Build plan with single remote object.
`DefaultCommitOptions`: `store.protoZettel` +
`AddToInventoryList, UpdateTai, RunHooks, Validate, ApplyProto`.

**`edit_config.go`:** Build plan with single config object.
`DefaultCommitOptions`: `store.protoZettel` +
`AddToInventoryList, UpdateTai, RunHooks, Validate`. Note: `Builder.AddObject`
skips Config genre --- this needs handling (either set classification directly
or bypass the genre check for local builders).

**`checkin_blob.go`:** Build plan with all pairs. `DefaultCommitOptions`:
`store.protoZettel` +
`AddToInventoryList, UpdateTai, RunHooks, Validate, MergeCheckedOut`.

**`LockAndCommitOrganizeResults`:** Build plan with all changed objects.
`DefaultCommitOptions`: explicit proto (workspace default type) +
`AddToInventoryList, UpdateTai, RunHooks, Validate, MergeCheckedOut`. The \>30
object confirmation prompt moves before `ExecutePlan` (before the lock), which
is better.

### Interface Cleanup

After all four migrations: remove `Commit` from `sku.RepoStore` interface.
Internal store methods (`RevertTo`, `CreateOrUpdateBlobDigest`,
`CreateOrUpdateCheckedOut`) continue calling `Commit` as a concrete method on
`*Store`.

### Migration Order

1.  `MakeBuilder` → `MakeImportBuilder` rename + add `MakeLocalBuilder`
2.  `remote_add` (single object, simple)
3.  `edit_config` (single object, config genre consideration)
4.  `checkin_blob` (loop, pre-existing objects)
5.  `LockAndCommitOrganizeResults` (loop, explicit proto, confirmation prompt)
6.  Remove `Commit` from `sku.RepoStore`

## Rollback

Each migration is independently revertable --- restore manual Lock →
`CreateOrUpdate` → Unlock. Convenience wrappers remain on `*Store` throughout.
`Commit` interface removal is independently revertable by re-adding the method.

## Testing

Existing integration suite (`just test`) covers all four commands. No new tests
needed --- transformation is behavior-preserving.
