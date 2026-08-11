---
status: experimental
date: 2026-08-09
promotion-criteria: a list-in/list-out Lua transform command exists and can
  (a) mutate an object's type, not just its tags/fields, (b) read and write
  raw blob content by digest, returning a usable markl id, and (c) validate
  its output against fsck's verification logic before any commit, with a
  working dry-run mode; at least one real bulk-migration script (a tag/type
  cleanup pass over a personal repo) has been run successfully end to end.
promotion-status: the `transform` command (implemented 2026-08-09 per
  RFC-0008) satisfies (a), (b), and (c); the real bulk-migration run over a
  personal repo is still outstanding and rides the personal-data program
  (dodder#16), which is this feature's first consumer.
---

# Inventory-List Transform Plugins

## Problem Statement

dodder has no `git filter-repo` equivalent. There is no way to write a
script that rewrites the object graph in bulk — canonicalizing a type across
thousands of objects, dropping a deprecated tag everywhere it appears,
merging duplicate import artifacts. The closest existing mechanisms are both
the wrong shape for this:

- **`import_plan.ObjectTransform`** (`go/internal/hotel/import_plan/builder.go:18`)
  is a per-object `func(*sku.Transacted) (keep bool, err error)` callback
  already wired into `checkin`, `organize`, and `remote-add` via
  `MakeLocalBuilder()` + `AddObject`. It mutates one object at a time with
  no visibility into the rest of the graph — fine for a regex tag filter
  (`MakeOmitTagsTransform`), useless for a transform that needs to reason
  about relationships between objects (deduplication, cross-reference
  rewrites).
- **RFC-0006's commit-time Lua hooks** (`on_pre_commit`, `on_commit_fields`)
  run *inside* a live commit, one object at a time, and deliberately
  withhold type and blob write-back (`lua_transacted_v1.go:114-117`, tracked
  as issue #319, gated on RFC-0006 Phase 2) because that binding is guarded
  by a re-entrancy counter (`hookDepth`,
  `go/internal/oscar/store/hooks.go:413-420`) protecting against nested
  commits triggered from within a hook. Hooks are designed for per-commit
  policy (auto-archive a done task, advance a recurring due date), not mass
  graph rewriting.

Neither mechanism can express "read this whole selected subgraph, rewrite
it however the transform needs to, hand back a new subgraph" — which is
what a real migration script needs.

## Interface

A new **list-in/list-out Lua transform** mechanism, complementing (not
replacing) the existing per-object `ObjectTransform`:

```
build expanded inventory list (query + transitive closure)
        │
        ▼
run Lua plugin: script receives the full list, returns a new list
        │
        ▼
validate output against fsck's verification logic (default: on)
        │
        ▼
dry-run? ──yes──▶ report and stop
        │no
        ▼
commit via the existing two-stage-commit plan/ExecutePlan path
```

### Why list-in/list-out, not per-object streaming

A script needs the *whole* selected graph in view to do things like
deduplicate near-identical objects, rewrite every reference to a tag being
renamed, or decide that dropping one object means three others become
orphaned and should be dropped too. None of that is expressible as an
isolated per-object callback. The list-transform script receives a complete
list (the query's matches plus their full transitive closure — types, tags,
and other referenced objects, via the previously pull-path-only
`expandEdges` traversal, now also driven locally by
`MakeExpandedInventoryList`,
`go/internal/romeo/local_working_copy/op_make_expanded_inventory_list.go`)
and returns
a complete list. What comes back can differ arbitrarily from what went in:
objects can be mutated, dropped, or newly created.

### Why type mutation is safe here but not in a commit hook

The commit-hook restriction on type/blob write-back exists specifically
because that code runs inside a live, potentially-nested commit — the
`hookDepth` guard exists to prevent a hook from triggering a commit that
triggers another hook. The list-transform's Lua invocation runs in a
completely different context: a single batch pass over a plan that is being
*built*, before any commit has begun. There is no nesting and no
re-entrancy risk. This means type mutation for list-transform scripts is
new, additive write-back logic scoped to this context — it does not touch,
weaken, or require finishing RFC-0006 Phase 2's hook-side gate.

### Why blob rewriting doesn't need its own "rehash" feature

Exposing raw blob read/write to Lua, backed by the same
`mad_domain_interfaces.BlobStore` interface this session's throwaway
`reconcile_blob_to_store.go`/`repair_config_blob.go` commands already used
directly, means a script can read a blob's bytes and write them back
through a store configured with a different hash algorithm — the returned
markl id is naturally in the new format. Hash-algorithm migration (the
planned move to blake3) falls out of this for free: it's a *consumer* of
the blob FFI, not a feature the FFI itself needs to know about. This is why
the blake3 migration is explicitly **not** part of this plugin API design —
it will be its own specialized transform script, built later, on top of
this infrastructure, exactly as the tag/type cleanup pass will be.

### Why fsck is the safety net, not a new validator

`fsck`'s core verification (`go/internal/uniform/commands_dodder/fsck.go`,
`runVerification`) already takes an `interfaces.SeqError[*sku.Transacted]`
and does not require that sequence to come from the live committed store —
it already supports reading from an inventory-list file
(`cmd.MakeSeqFromPath`). A list-transform's output, before commit, is just
another such sequence. Validating it before allowing a commit reuses
existing, already-trusted verification logic rather than inventing a
second consistency checker. Because the list-transform can do arbitrary
rewriting (including creating dangling references, if a script has a bug or
is mid-multi-pass-migration), this validation runs by default; a flag
disables it for a script author who is deliberately staging an intermediate,
temporarily-inconsistent state across multiple passes.

### Why Lua, and why not shell out to an external process

The simpler alternative — `export | external-transform | import`, letting a
script in any language operate on the serialized inventory-list text format
— was considered and rejected. It would work, but every transform author
would have to hand-roll inventory-list parsing and blob access themselves,
in whatever language they chose, for every script. The explicit goal is
that a transform script should focus on the transform, not on plumbing. An
embedded Lua plugin, using the existing Lua VM machinery
(`go/lib/alfa/lua/`; the transform holds one VM for the batch pass rather
than borrowing from the pool — see RFC-0008 and #390) and native Go-backed
bindings for the object list and blob store, gives scripts that plumbing
for free. It also inherits that machinery's existing sandboxing — a real
security property worth keeping
for a plugin that gets raw blob-write power. The sandbox allowlist
(see `go/lib/alfa/lua/CLAUDE.md`) admits only: `base` (minus
`dofile`/`loadfile`/`load`/`loadstring`), `package` (preload searcher only;
filesystem searcher blocked), `string`, `table`, and `math`. `io`, `os`,
`coroutine`, `channel`, and `debug` are never opened; `require()` outside
the preloaded `der`/`dodder`/`zit` module aliases fails with an actionable
error.

## Examples

In the shipped surface (see RFC-0008 for the concrete API; object handles
use the existing V1 projection's field names):

```lua
-- tag/type cleanup pass (no blob access needed)
local list = dodder.list()
for object in list:each() do
  if object.Typ == "!task-legacy" then
    object.Typ = "!task"
  end
  object.Etiketten["newsblur"] = nil
end
return list
```

```lua
-- blake3 migration pass (a later, separate script — not designed here,
-- shown only to demonstrate the blob FFI composes for this without any
-- rehash-specific API)
local list = dodder.list()
for object in list:each() do
  local bytes = blobs.read(object.Blob)
  object.Blob = blobs.write(bytes)  -- store configured for blake3
end
return list
```

## Pluggable source: one transform, three sources (dodder#392)

The list-in/list-out shape is indifferent to where the input list comes
from. Once the `transform` command existed, the same machinery generalized —
with no change to the script contract — to two more sources, because
"rewrite this graph in bulk" is the same operation whether the graph comes
from a query, a pile of archived inventory-list files, or another repo's
history:

- **`transform`** — the query source: rewrite this repo's own objects in
  place.
- **`init-from-lists`** — the inventory-list-union source: consolidate N
  archived inventory-list files into a FRESH repo through one transform.
  This is `git filter-branch` into a fresh repo — the history is born
  already rewritten (tag cleanup, fork resolution, hash migration in a
  single pass) and re-signed under the newborn's key, instead of carrying
  legacy mess plus correction commits. Distinct from `init-from`
  (copy-migration: same keypair, single source, signatures preserved).
- **`clone -script`** — the pull-stream source: clone another repo and apply
  the transform in the same pass, so the clone is born rewritten rather than
  pulled verbatim and corrected after.

The two fresh-repo consumers share a re-signing commit (foreign objects are
re-signed under the new repo's key, since a transformed object cannot keep
its source signature) and make the result self-contained (every referenced
source blob is copied into the new repo, so it survives deleting the
sources). RFC-0008 §7 gives the concrete surface. `clone -script` over a
networked transport is deferred (dodder#393); it is direct-transfer only
today.

## Open Questions

- ~~Exact CLI command name and flag surface~~ — resolved: `transform`; see
  RFC-0008 for the shipped surface.
- Whether the strict "no new objects" mode (rejecting output objects absent
  from the input) needs finer granularity than a single on/off flag (e.g.
  allowed for one genre but not another).
- Long-term: whether this mechanism should eventually be exposed as a
  general `ObjectTransform`-compatible adapter, letting a list-transform
  script also be usable in the simpler per-object contexts. Not needed for
  the initial migration use case; noted for later.

## References

- [RFC-0008 — Inventory-List Transform Plugin API](../rfcs/0008-inventory-list-transform-plugin-api.md)
- [RFC-0006 — Hook Commit-Time Mutation](../rfcs/0006-hook-commit-time-mutation.md)
- [FDR-0002 — Two-Phase Import](0002-two-phase-import.md)
- [FDR-0006 — Two-Stage Commit](0006-two-stage-commit.md)
