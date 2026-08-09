---
date: 2026-08-09
status: experimental
---

# Inventory-List Transform Plugin API

## Abstract

This document specifies the concrete interface for the list-in/list-out
Lua transform mechanism proposed in FDR-0024: the `transform` CLI command,
a Lua global exposing a mutable object list, a Lua global exposing raw blob
read/write, a write-back path (distinct from RFC-0006's hook write-back)
that additionally supports type mutation, and the validation/commit
pipeline tying them together. It reuses the existing Lua VM pool
(`go/lib/alfa/lua/vm_pool.go`), the existing `sku_lua` object projection
(read side only), the existing two-stage-commit plan/execute machinery
(`import_plan`, `ExecutePlan`), and `fsck`'s existing verification core. No
new VM infrastructure, no new commit machinery, no new consistency-checking
logic — only new bindings and a new command wiring existing pieces together
in a new order.

Implemented 2026-08-09; this revision describes the shipped surface. The
original draft's open choices (command name, handle shape, exact validation
scope) are resolved below, with deviations from the draft called out where
the implementation taught us the draft was wrong.

## Command

```
dodder transform -script <path> [query args...]
dodder transform -script-digest <markl-id> [query args...]
```

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `-script <path>` | — | Load the Lua script from a local file. Mutually exclusive with `-script-digest`. |
| `-script-digest <markl-id>` | — | Load the Lua script from a stored blob, addressed by markl id via the multi-store read fallback (`GetReadBlobStore`). No dedicated script type object is required; how the blob got stored (e.g. `dodder new` with any type, `madder write`) is out of scope. Mutually exclusive with `-script`. |
| `-dry_run` | `false` | Build and validate the output plan; report it; do not call `ExecutePlan`. |
| `-skip_validation` | `false` | Skip the fsck-style validation pass on the transform's output. For staged, intentionally-inconsistent intermediate migration passes. |
| `-no_new_objects` | `false` | Reject any output object whose object id is not present in the input list. |

Query args select the starting object set exactly as in `export`/`show`
(reuses the existing query builder — no new query syntax). Defaults: genre
zettels, sigils latest + hidden (dormant objects are included so cleanup
passes reach them; history is opt-in via `+`).

## Pipeline

### 1. Build the expanded input list

`local_working_copy.Repo.MakeExpandedInventoryList(query)`
(`go/internal/romeo/local_working_copy/op_make_expanded_inventory_list.go`)
builds the query-matched set via the existing `MakeInventoryList` and then
runs the previously pull-path-only `expandEdges`
(`go/internal/romeo/local_working_copy/expand_edges.go`) against it, driven
by `store.MakeEdgeExplorer` over the local object and blob stores, with the
same `sku.EdgeExplorer`/`maxEdgeExpansionDepth` (5) semantics already in
place for pull. This was the only structural change to existing non-plugin
code required — everything else is additive.

### 2. Load and invoke the script

The script source (`-script` file or `-script-digest` blob) is compiled
into a bare `lua.VMPoolBuilder` — deliberately without the module searcher
the tag-filter VMs get, so a transform script has no `require()` at all:
the strictest sandbox variant (no file I/O, no network, no module loading).

Two globals are registered via the builder's apply hook before the script
chunk executes:

- `dodder` — carries `list()`, the object-list binding (§3).
- `blobs` — the blob FFI (§4).

The script chunk executes during VM preparation (the pool's `PrepareVM`
compiles and `PCall`s it); its `return` value is read back on the Go side
from `vm.Top` and MUST be the handle produced by `dodder.list()` (§3.4).

### 3. The `dodder` list binding

Backed by `sku_lua.ListTransformV1`
(`go/internal/golf/sku_lua/lua_list_transform_v1.go`).

#### 3.1 The list handle

`dodder.list()` returns the list handle — a Lua table with `each`,
`remove`, and `add` methods. Every call returns the same handle; the list
identity is the invocation's input set, not a fresh copy per call.

Each element is projected via the *existing* read-side projection,
unchanged: `sku_lua.ToLuaTableV1`
(`go/internal/golf/sku_lua/lua_transacted_v1.go`). V1, not V2, because only
V1 projects the metadata index fields (`Fields`), which transform scripts
need for field rewriting. An object handle therefore has the established V1
shape: `Gattung` (genre), `Kennung` (object id), `Typ` (type), `Etiketten`
(tags table: name → true), `EtikettenImplicit`, `Fields` (name → value) —
plus one transform-only addition: `Blob`, the object's blob digest as a
markl id string. `Blob` is projected by the list binding, not by
`ToLuaTableV1` itself, so hook scripts never see a blob mutation surface
(the RFC-0006 Phase 2 gate, issue #319, stays intact). Assigning
`object.Blob = blobs.write(...)` is how a script points an object at a blob
it rewrote — the composition the FDR's hash-migration story depends on. An
empty string clears the digest.

The draft's illustrative lowercase surface (`object.type`,
`object.tags:remove(...)`) was NOT adopted: the normative rule "reuse the
existing projection unchanged" won, keeping transform scripts and RFC-0006
hook scripts in one dialect.

#### 3.2 Iteration and mutation

```lua
local list = dodder.list()

for object in list:each() do
  if object.Typ == "!task-legacy" then
    object.Typ = "!task"
  end

  object.Etiketten["newsblur"] = nil     -- remove a tag
  object.Etiketten["migrated"] = true    -- add a tag
  object.Fields.status = "done"          -- rewrite a projected field
end

return list
```

Mutation is in-place on the handle. Objects added mid-iteration (§3.3) are
visited by the running iterator; removed objects are skipped.

#### 3.3 List membership

`list:remove(object)` drops an object from the output list (equivalent to
`ObjectTransform` returning `keep = false`; the object simply gets no new
revision — nothing is deleted from the store). Passing a table that is not
an object handle from this list raises a Lua error.

`list:add()` creates a new zettel object not present in the input and
returns its handle for mutation. The draft's `prototype` argument was
dropped: the returned handle is itself the mutation surface. The object id
is left empty; allocation happens Go-side at plan build via the existing
`import_plan.MakeAllocateZettelIdTransform`
(`go/internal/hotel/import_plan/transform_allocate_zettel_id.go`) rather
than new allocation logic. New objects without a scripted `Typ` receive the
repo's proto-zettel default type at commit.

#### 3.4 Return value

The script MUST `return` the handle produced by `dodder.list()`; anything
else aborts the command before any plan is built. The Go side then reads
the output set back off the retained per-object projections — membership
from the remove/add bookkeeping, mutations via §3.5.

#### 3.5 Write-back: `FromLuaTableTransformV1`

`FromLuaTableV1`/its V2 equivalent remain exactly as they are today —
hook-scoped, restricted, unchanged. The transform write-back is
`sku_lua.FromLuaTableTransformV1`
(`go/internal/golf/sku_lua/lua_transacted_transform_v1.go`): the same
genre/id/tags/fields write-back plus, additionally, `Typ` write-back via
`object.GetMetadataMutable().GetTypeMutable().ResetWithType(...)` (the
existing mutator) and `Blob` write-back (§3.1) via the metadata's mutable
blob digest. Blob *content* still moves exclusively through the FFI (§4);
the `Blob` field carries only the digest.

### 4. The `blobs` FFI

Two functions, registered as Lua globals under a `blobs` table:

- `blobs.read(markl_id_string) -> bytes` — wraps
  `GetReadBlobStore().MakeBlobReader(digest)` + `io.ReadAll` (multi-store
  read fallback, per FDR-0015), returning the raw bytes as a Lua string.
  Errors (blob not found, read failure) raise a Lua error via
  `RaiseError`.
- `blobs.write(bytes) -> markl_id_string` — wraps
  `GetDefaultBlobStore().MakeBlobWriter(nil)` + `Write` + `GetMarklId()`.
  Returns the digest as a string in the same format markl `.String()`
  produces, so it round-trips into blob-reference fields and future
  digest-bearing assignments.

Both resolve their stores the same way other commands do — no new
store-resolution logic.

### 5. Validation

Unless `-skip_validation` is set, after the script returns and the
resulting objects are built into an `import_plan.Plan` via `Build()`, the
plan's committable entries are wrapped as an
`interfaces.SeqError[*sku.Transacted]` and run through fsck's verification
core, extracted as `runSeqVerification`
(`go/internal/uniform/commands_dodder/fsck.go`) and shared by both
commands.

**Deviation from the draft.** The draft listed object digest presence,
signature integrity, and stream-index probe verification among the checks.
Those describe *committed* state and are wrong for candidates: the commit
path resets the object digest and the inventory-list flush re-signs every
object (`FinalizeAndSignOverwrite`), so a mutated candidate's stale
digest/sig pair would verify trivially, a freshly added object's null
digest/sig would fail spuriously, and mutated metadata's probes are
legitimately absent from the stream index. The transform therefore invokes
the shared core with those checks disabled, leaving the candidate-relevant
safety net the FDR actually motivated: **blob presence** for every blob
digest a script assigned, and **dangling blob-reference detection** (#330
semantics) across the multi-store read view. `fsck` itself is unchanged and
still runs the full check set. Any validation failure is reported (TAP
not-ok lines) and the command exits without calling `ExecutePlan`, whether
or not `-dry_run` was also set.

### 6. Dry run and commit

`-dry_run` means: perform steps 1–5, print the plan summary (entry count
plus counts by `Classification`, per FDR-0002/FDR-0006's existing
classification vocabulary), then `dry run: not committed`. Without
`-dry_run`, proceed to `local.ExecutePlan(plan)`
(`go/internal/romeo/local_working_copy/execute_plan.go`) exactly as it
exists today — lock, commit per entry, unlock — with
`GetStoreOptionsUpdate` commit options (tai update, hooks, validation; no
proto application onto existing objects) and the proto zettel supplied for
mother-less (added) objects only. No changes to `ExecutePlan` itself;
dry-run is purely "don't call it."

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
- **A dedicated stored-script type** (the draft's `!lua-transform-v1`
  sketch) was not introduced; `-script-digest` addresses blobs directly by
  markl id. A convention type can be layered on later without changing this
  API.

## References

- [FDR-0024 — Inventory-List Transform Plugins](../features/0024-inventory-list-transform-plugins.md)
- [RFC-0006 — Hook Commit-Time Mutation](0006-hook-commit-time-mutation.md)
- [FDR-0006 — Two-Stage Commit](../features/0006-two-stage-commit.md)
- [FDR-0002 — Two-Phase Import](../features/0002-two-phase-import.md)
- [FDR-0015 — Multi-Store Blob Lookup](../features/0015-multi-store-blob-lookup.md)
