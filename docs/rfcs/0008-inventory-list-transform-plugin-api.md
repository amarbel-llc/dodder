---
date: 2026-07-12
status: draft
---

# Inventory-List Transform Plugin API

## Abstract

This document specifies the concrete interface for the list-in/list-out
Lua transform mechanism proposed in FDR-0024: a new CLI command, a new Lua
global exposing a mutable object list, a new Lua global exposing raw blob
read/write, a new write-back path (distinct from RFC-0006's hook write-back)
that additionally supports type mutation, and the validation/commit
pipeline tying them together. It reuses the existing Lua VM pool
(`go/lib/alfa/lua/vm_pool.go`), the existing `sku_lua` object projection
(read side only), the existing two-stage-commit plan/execute machinery
(`import_plan`, `ExecutePlan`), and `fsck`'s existing verification core. No
new VM infrastructure, no new commit machinery, no new consistency-checking
logic — only new bindings and a new command wiring existing pieces together
in a new order.

## Command

A new subcommand, name TBD at implementation time (candidates:
`transform`, `rewrite` — avoid `filter-repo`/`filter-branch` as literal
names since dodder's content-addressed, append-only model means nothing is
destructively rewritten in place the way git's history is; original objects
remain reachable after a transform produces new revisions).

```
dodder <cmd> -script <path> [query args...]
dodder <cmd> -script-digest <markl-id> [query args...]
```

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `-script <path>` | — | Load the Lua script from a local file. Mutually exclusive with `-script-digest`. |
| `-script-digest <markl-id>` | — | Load the Lua script from a stored blob (a new `!lua-transform-v1` type object, mirroring `exec-lua`'s existing `!lua-tag-v1/v2` model). Mutually exclusive with `-script`. |
| `-dry_run` | `false` | Build and validate the output plan; report it; do not call `ExecutePlan`. |
| `-skip_validation` | `false` | Skip the fsck-style validation pass on the transform's output. For staged, intentionally-inconsistent intermediate migration passes. |
| `-no_new_objects` | `false` | Reject any output object whose object id is not present in the input list. |

Query args select the starting object set exactly as in `export`/`show`
(reuses the existing query builder — no new query syntax).

## Pipeline

### 1. Build the expanded input list

`local_working_copy.Repo.MakeInventoryList(query)`
(`go/internal/romeo/local_working_copy/op_make_inventory_list.go:10`) builds
the query-matched set but does not include the transitive closure. A new
entry point, `MakeExpandedInventoryList(query)`, additionally runs the
existing (currently private, pull-path-only) `expandEdges`
(`go/internal/romeo/local_working_copy/expand_edges.go:11`) against it,
using the same `sku.EdgeExplorer`/`maxEdgeExpansionDepth` (5) semantics
already in place for pull. This is the *only* structural change to existing
non-plugin code required by this design — everything else is additive.

### 2. Load and invoke the script

The script source (`-script` file or `-script-digest` blob) is compiled
into the existing `lua.VMPool`
(`go/lib/alfa/lua/vm_pool.go`, `SetReader`/`SetCompiled`), exactly as
`exec-lua` already does (`go/internal/sierra/repo_actions/exec_lua.go:14-46`)
for its own script source. A VM is obtained via `PoolPtr.GetWithRepool()`.

Before invocation, two new Lua globals are registered via
`vm.SetGlobal(name, vm.NewFunction(...))` — the same registration pattern
`dodder_advance_date` already uses
(`go/internal/hotel/tag_blobs/lua_v1.go:36-53`):

- `dodder` — the object-list binding (below).
- `blobs` — the blob FFI (below).

The script is expected to `return` a list value (see §3.4); the VM's
return value is read back on the Go side after `PCall` completes.

### 3. The `dodder` list binding

#### 3.1 Constructing the input list handle

`dodder.list()` returns a Lua table wrapping the Go-side
`sku.HeapTransacted` built in step 1. Each element is projected via the
*existing* read-side projection, unchanged:
`sku_lua.ToLuaTableV1`/`ToLuaTableV2`
(`go/internal/golf/sku_lua/lua_transacted_v1.go:18-61`,
`lua_transacted_v2.go`) — `Gattung`/genre, `Kennung`/id, `Typ`/`Type`,
`Etiketten`/tags, `EtikettenImplicit`/implicit tags, `Fields`.

#### 3.2 Iteration

`list:each()` returns an iterator over per-object Lua table handles (same
shape as an individual object's projection above), suitable for a `for
object in list:each() do ... end` loop. Mutation is in-place on the handle
(`object.type = "..."`, `object.tags:remove(...)`, `object.tags:add(...)`,
`object.fields.<name> = "..."` — matching the existing field-write pattern
in `FromLuaTableV1`/`writeFieldsBack`).

#### 3.3 List membership

`list:remove(object)` drops an object from the list (equivalent to
`ObjectTransform` returning `keep = false`).
`list:add(prototype)` creates a new object not present in the input,
returning a handle for further mutation. New objects need id allocation —
this reuses `import_plan.MakeAllocateZettelIdTransform`'s existing pattern
(`go/internal/hotel/import_plan/transform_allocate_zettel_id.go`) rather
than inventing new allocation logic.

#### 3.4 Return value

The script's `return`ed value must be a `dodder.list()`-produced handle
(the same one passed in, or a fresh one built via repeated `list:add`) —
the Go side reads it back into a `sku.HeapTransacted` (or directly into a
new `import_plan.MakeLocalBuilder()` + `AddObject` sequence, mirroring the
established local-plan pattern already used by
`checkin_haustoria.go:44-49`, `remote_add.go:86-91`).

#### 3.5 Write-back: a new function, not `FromLuaTableV1`

`FromLuaTableV1`/its V2 equivalent remain exactly as they are today
(`lua_transacted_v1.go:68-121`) — hook-scoped, restricted, unchanged. A
**new** function (name TBD, e.g. `FromLuaTableTransformV1`, living in a new
file in `sku_lua` or a new package if keeping hook and transform write-back
cleanly separated is preferred at implementation time) performs the same
tags/fields write-back plus, additionally, `Typ`/`Type` write-back via
`object.GetMetadataMutable().GetTypeMutable().ResetWithType(...)` (the
existing mutator, already used elsewhere, e.g.
`go/internal/india/config_log/main.go`'s `Append`). Blob digest write-back
(§4) is handled separately, not through this table projection, since blob
content and blob digest are handled via the FFI directly.

### 4. The `blobs` FFI

Two functions, registered as Lua globals under a `blobs` table:

- `blobs.read(markl_id_string) -> bytes` — wraps
  `mad_domain_interfaces.BlobStore.MakeBlobReader(digest)` +
  `io.ReadAll`, returning the raw bytes as a Lua string. Errors (blob not
  found, read failure) raise a Lua error via `luaState.RaiseError()`
  (matching `lua_v1.go:46`'s existing error-raising pattern).
- `blobs.write(bytes) -> markl_id_string` — wraps
  `MakeBlobWriter(nil)` + `Write` + `GetMarklId()` (the exact sequence this
  session's `reconcile_blob_to_store.go` already used directly against the
  real repo, including the established defer-only-close pattern —
  `defer errors.DeferredCloser(&err, writer)`, no separate explicit
  `Close()` call, per the double-close bug found and fixed earlier this
  session in that same throwaway command). Returns the digest as a string
  in the same format `object.blob_digest`/markl `.String()` already
  produces, so it round-trips directly into a Lua-side assignment like
  `object.blob_digest = new_digest`.

Both operate against the target blob store resolved the same way other
write-capable commands resolve it today (the repo's default blob store for
writes; read side uses the existing multi-store fallback,
`GetReadBlobStore()`) — no new store-resolution logic.

### 5. Validation

Unless `-skip_validation` is set, after the script returns and the
resulting objects are built into an `import_plan.Plan` via `Build()`, the
plan's entries are wrapped as an `interfaces.SeqError[*sku.Transacted]` —
the same wrapping `export.go` already does via
`quiter.MakeSeqErrorFromSeq(list.All())` — and run through fsck's existing
`runVerification` logic
(`go/internal/uniform/commands_dodder/fsck.go`): object digest presence,
signature integrity, stream-index probe verification, blob presence, and
dangling blob-reference detection. Any verification failure is reported
and the command exits without calling `ExecutePlan`, whether or not
`-dry_run` was also set.

### 6. Dry run and commit

`-dry_run` means: perform steps 1–5, print a summary of the resulting plan
(counts by `Classification`, per FDR-0002/FDR-0006's existing
classification vocabulary — `ClassificationImport`,
`ClassificationResolveTaiReassign`, etc.), and stop. Without `-dry_run`,
proceed to `local.ExecutePlan(plan)`
(`go/internal/romeo/local_working_copy/execute_plan.go:9-51`) exactly as
it exists today — lock, all-or-nothing commit per entry, unlock. No
changes to `ExecutePlan` itself; dry-run is purely "don't call it," since
it has no dry-run mode of its own to thread through.

## Non-goals

- **Hash-algorithm migration (blake3)** is explicitly not designed here.
  The blob FFI in §4 gives a future migration script everything it needs
  (read bytes, write to a store configured with a different hash format,
  get back the new digest) without any blake3-specific API. That script,
  and how a store-wide rehash is sequenced safely, is future, separate
  work.
- **Modifying RFC-0006's hook write-back** is explicitly out of scope —
  `FromLuaTableV1`/V2 and the `hookDepth` guard are unchanged by this RFC.
- **A general `ObjectTransform`-compatible adapter** making list-transform
  scripts reusable in per-object contexts (noted as an open question in
  FDR-0024) is not designed here.

## References

- [FDR-0024 — Inventory-List Transform Plugins](../features/0024-inventory-list-transform-plugins.md)
- [RFC-0006 — Hook Commit-Time Mutation](0006-hook-commit-time-mutation.md)
- [FDR-0006 — Two-Stage Commit](../features/0006-two-stage-commit.md)
- [FDR-0002 — Two-Phase Import](../features/0002-two-phase-import.md)
