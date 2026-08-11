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
| `-dry_run` | `false` | Build and validate the output plan; report it (per-entry listing plus summary); do not call `ExecutePlan`. The script still executes, but `blobs.write` is contained (#390): its output goes to a discardable, run-stamped staging store under the repo's cache tree, never the real blob store, and `blobs.read` overlays that staging store over the real read view so read-your-writes holds within the run. The dry-run summary reports the staging location and the digests staged there; that directory is always safe to delete. |
| `-skip_validation` | `false` | Skip the fsck-style validation pass on the transform's output, and tolerate edge-expansion failures (dangling references) when building the input list. For staged, intentionally-inconsistent intermediate migration passes. |
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
the tag-filter VMs get, so a transform script has no stored-module
`require()`. Scripts execute inside the VM pool sandbox (issue #389,
`go/lib/alfa/lua/stdlib.go`): `io`, `os`, `dofile`, `loadfile`, `load`,
and `loadstring` are blocked; `require()` outside preloaded modules fails
with an actionable error. See §"Lua Sandbox Migration Note" for migration
guidance.

Two globals are registered via the builder's apply hook before the script
chunk executes:

- `dodder` — carries `list()`, the object-list binding (§3).
- `blobs` — the blob FFI (§4).

The transform builds a single, explicitly-owned VM
(`VMPoolBuilder.BuildSingleVM`) rather than borrowing from the pool: the
chunk is compiled once and `PCall`ed exactly once during VM preparation,
and its `return` value is read back on the Go side from `vm.Top` and MUST
be the handle produced by `dodder.list()` (§3.4). Single-run execution is
deliberate (#390): the pooled path is `sync.Pool`-backed, so a chunk run in
a trial VM could execute a second time if that VM were evicted before its
first borrow, firing non-idempotent side effects (chiefly `blobs.write`)
twice; one owned VM removes that window structurally. The pool is unchanged
and remains in use for the repeated per-object tag-filter workload it was
built for.

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

Tag removal accepts both natural Lua set idioms: `= nil` and `= false`.
Assigning an empty string to `Kennung` or `Typ` is a no-op — the
projection cannot distinguish "left alone" from "cleared", so clearing an
object's type or blanking its id to request re-allocation is not
expressible (only `list:add` creates allocation candidates). `Blob` is the
one field where `""` explicitly clears, because write-back compares
against the current digest.

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
from the remove/add bookkeeping, mutations via §3.5. An output set naming
the same object id more than once (a script reassigning one handle's
`Kennung` onto another object's id) is always rejected, independent of
`-no_new_objects` — committing it would silently collapse two objects'
updates into last-write-wins revisions of one.

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

On a real run both resolve their stores the same way other commands do — no
new store-resolution logic: `blobs.write` targets the default store and
`blobs.read` the multi-store read view. Under `-dry_run` (§6) the FFI is
rebound to a staging overlay — writes go to a discardable staging store and
reads consult it before the real read view — so the real store is never
written yet read-your-writes still holds within the run.

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
safety net the FDR actually motivated: **full content verification of each
object's blob digest** (the blob must exist and re-hash to its digest) and
**dangling blob-reference detection** (#330 semantics, presence checks)
across the multi-store read view. `fsck` itself is unchanged and
still runs the full check set. Any validation failure is reported (TAP
not-ok lines) and the command exits without calling `ExecutePlan`, whether
or not `-dry_run` was also set.

### 6. Dry run and commit

`-dry_run` means: perform steps 1–5, print the plan summary (entry count
plus counts by `Classification`, per FDR-0002/FDR-0006's existing
classification vocabulary), surface any blobs `blobs.write` staged (their
count and the run-stamped staging directory, which is always safe to
delete — see §4 and `env_repo.MakeDiscardableStagingBlobStore`), then
`dry run: not committed`. Without
`-dry_run`, proceed to `local.ExecutePlan(plan)`
(`go/internal/romeo/local_working_copy/execute_plan.go`) exactly as it
exists today — lock, commit per entry, unlock — with
`GetStoreOptionsUpdate` commit options (tai update, hooks, validation; no
proto application onto existing objects) and the proto zettel supplied for
mother-less (added) objects only. No changes to `ExecutePlan` itself;
dry-run is purely "don't call it."

## 7. Pipeline generalization: three sources (dodder#392)

Everything from the VM onward — script invocation, output read-back,
duplicate check, plan build, validation, dry-run/commit — is indifferent to
where the input objects came from. dodder#392 extracts steps 2–6 as a
source-agnostic `transformPipeline`
(`go/internal/uniform/commands_dodder/transform_pipeline.go`) and hangs
three consumers off it, differing only in how they produce the input list,
which repo they target, and how they commit.

| Consumer | Source of objects | Target | Commit |
|---|---|---|---|
| `transform` | query + `expandEdges` over this repo | this repo | `ExecutePlan` (locally-authored; sealed under this repo's key at the working-list flush) |
| `init-from-lists` | union of N inventory-list files | a FRESH repo | `CommitPlan` + `OverwriteSignatures` (re-sign under the newborn's key) |
| `clone -script` | the source repo's pulled objects | a FRESH repo (the clone) | `CommitPlan` + `OverwriteSignatures` (re-sign under the clone's key) |

### 7.1 The two commit paths

`transform`'s objects are already this repo's own, so `ExecutePlan` seals
them under this repo's key at the inventory-list flush with nothing foreign
to reconcile. The two new consumers import FOREIGN objects (list files,
another repo's stream) into a fresh repo and MUST re-sign every object under
the new repo's key: `remote_transfer.CommitPlan` with
`ImporterOptions.OverwriteSignatures` resets each object's sig/pubkey/digest
and re-signs via `FinalizeAndSignOverwrite`. `ExecutePlan` does NOT re-sign,
so it is wrong for a foreign source.

`clone -script` follows clone's existing signing model unchanged: a plain
clone already re-signs everything under the clone's own key (the same
`CommitPlan`/`OverwriteSignatures` path), so a scripted clone introduces no
new signing policy — and a transformed object cannot keep its source
signature anyway, its content having changed. Remote-signature preservation
stays the standing pre-existing TODO's concern, out of scope here.

### 7.2 Duplicate-object-id handling

`transform`'s query source yields one latest version per id, so two same-id
outputs mean the script merged two objects onto one id (silent
last-write-wins) — rejected (§3.4). The inventory-list consumers leave that
rejection OFF: a history union/clone carries many `(id, tai)` versions per
id BY DESIGN, and ruled fork-resolution is a deliberate same-id merge. The
import builder's within-batch `(id, tai)` reassign guards the genuine
last-write-wins hazard instead. `init-from-lists` additionally collapses
exact `(id, tai, digest)` duplicates across its input lists before the
script sees them, so passing the same list twice equals passing it once.

### 7.3 Source blobs and self-containment

A fresh repo's own store starts empty, but the transform's objects reference
blobs that live only in the source. Both fresh-repo consumers make the
result SELF-CONTAINED — every referenced blob is duplicated into the new
repo so it survives deleting the (often large) sources:

- `init-from-lists` reads source blobs from read-only `-blob-source` stores
  overlaid ahead of the newborn's read view (so `blobs.read` resolves them),
  then copies every referenced blob into the newborn before commit.
- `clone -script` pre-copies every referenced source blob into the clone
  before the transform. The source and clone share an XDG namespace, so an
  overlay would place the clone's own write store in its read list, which
  `MakeReadBlobStoreWithOverlay` rejects; a pre-copy sidesteps that.

A post-copy `fsck` against the new repo alone — with the sources detached —
is the self-containment proof, asserted in both commands' e2e tests.

### 7.4 New command surface

```
dodder init-from-lists -script <path> [-blob-source <store>...] <repo-id> <list-path>...
dodder clone -direct <path> -script <path> <repo-id> [query args...]
```

`init-from-lists` genesises a fresh repo (fresh keypair, fresh instance
identity, end-state config, `ExcludeDefaultType`) and consolidates N
inventory-list files through one transform — git-filter-branch into a fresh
repo. `clone -script` adds the transform to clone's existing genesis+pull; it
is **direct (local-path) transfer only** — the networked receive paths
(drtp, legacy HTTP) commit as they stream and are deferred (dodder#393).
Both reuse the same `-script`/`-script-digest` and validation surface
described above.

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

## Lua Sandbox Migration Note

Lua scripts running in the VM pool (commit hooks and, once implemented,
list-transform scripts) execute inside a sandbox that blocks `io`, `os`,
and several dangerous base-library functions. Scripts written before the
sandbox was hardened (issue #389) may need small updates:

| Removed | Replacement |
|---------|-------------|
| `os.date("!%Y-%m-%d")` | `dodder_today()` — Go-side global, returns current UTC date as `YYYY-MM-DD` |
| Any `os.*` access | Blocked; proxy raises `os is not available in dodder Lua scripts; use dodder_today() for the current date` |
| Any `io.*` access | Blocked; proxy raises `io is not available in dodder Lua scripts` |
| `dofile`, `loadfile`, `load`, `loadstring` | Blocked; no replacement — arbitrary code execution from the filesystem is not permitted |
| `require("anything-not-preloaded")` | Blocked; only the `der`/`dodder`/`zit` module aliases and explicitly preloaded modules are reachable |

`dodder_advance_date(date, duration)` (ISO-8601 duration math) was already
available before the sandbox hardening and is unaffected.

## References

- [FDR-0024 — Inventory-List Transform Plugins](../features/0024-inventory-list-transform-plugins.md)
- [RFC-0006 — Hook Commit-Time Mutation](0006-hook-commit-time-mutation.md)
- [FDR-0006 — Two-Stage Commit](../features/0006-two-stage-commit.md)
- [FDR-0002 — Two-Phase Import](../features/0002-two-phase-import.md)
- [FDR-0015 — Multi-Store Blob Lookup](../features/0015-multi-store-blob-lookup.md)
