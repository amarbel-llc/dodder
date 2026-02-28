# dagnabit per-tree sorting

## Problem

dagnabit computes package heights across the unified `lib/` + `internal/` graph.
When `internal/bravo` is empty (all bravo packages moved to `lib/bravo`),
`internal/charlie` packages still get height 3 because they depend on
`lib/bravo/errors` (height 2). dagnabit sees them as correctly placed and
produces no moves, leaving a gap in the `internal/` directory tree.

## Root cause

`GoListReader` merges edges from all prefixes into one graph. `TopologicalSort`
computes heights across the combined graph. Cross-tree dependencies inflate
`internal/` heights, anchoring packages at levels that match the unified graph
but leave gaps within the `internal/` tree.

## Fix

Sort each tree prefix independently. Within `internal/`, only internal-to-internal
edges determine height. Cross-tree dependencies (e.g., `internal/charlie/blech32`
→ `lib/bravo/errors`) are treated as external and ignored.

### Concrete changes

1. **`GoListReader`**: `ReadDependencies` already iterates per prefix. Change it
   to return `map[string][]Edge` keyed by prefix instead of `[]Edge`. Within
   `readPrefix`, only include edges where both source and target share the same
   prefix. Cross-prefix edges are dropped.

2. **`Repositioner`**: Change `Run` to iterate over each prefix's edges
   independently. For each prefix, run `TopologicalSort` on that prefix's edges,
   then map heights to levels and move packages. The `Mover` receives full paths
   including the tree prefix.

3. **`DependencyReader` interface**: Change return type from `([]Edge, error)` to
   `(map[string][]Edge, error)`.

### Result on current codebase

Per-tree sort of `internal/` produces:

| Current level | New level | Example packages |
|---|---|---|
| `_` | `_` (no change) | coordinates, external_state |
| `alfa` | `_` | domain_interfaces |
| `charlie` | `_` or `alfa` | blech32→alfa, blob_store_id→_ |
| `delta` | `_` or `alfa` | genres→alfa, doddish→_ |
| `echo` | `_`, `alfa`, or `bravo` | file_extensions→bravo |
| `foxtrot` | `bravo` | ids, markl, directory_layout |
| `golf` | `charlie` | fd, triple_hyphen_io |
| `hotel` | `delta` | objects, blob_store_configs |
| `india` | `echo` | env_dir, genesis_configs |
| `juliett` | `foxtrot` | blob_stores, zettel_id_index |
| `kilo` | `golf` | env_repo, sku |
| `lima` | `hotel` | box_format, type_blobs |
| `mike` | `india` | stream_index, sku_fmt |
| `november` | `juliett` | typed_blob_store |
| `oscar` | `kilo` | queries |
| `papa` | `lima` | store_workspace |
| `quebec` | `mike` | store_fs |
| `romeo` | `november` | env_workspace |
| `sierra` | `oscar` | store_config |
| `tango` | `papa` | store, env_box |
| `uniform` | `quebec` | repo |
| `victor` | `romeo` | remote_transfer |
| `whiskey` | `sierra` | local_working_copy |
| `xray` | `tango` | remote_http, user_ops |
| `yankee` | `uniform` | command_components_dodder |
| `zulu` | `victor` | commands_dodder |

The hierarchy compacts from `_` through `zulu` (24 tiers) to `_` through
`victor` (23 tiers). `lib/` keeps its existing levels unchanged (its own
per-tree sort produces the same result as the unified sort since `lib/` doesn't
depend on `internal/`).

## Testing

- Existing `TopologicalSort` tests: unchanged.
- Existing `Repositioner` tests: update stubs for new `map[string][]Edge` return
  type.
- New test: `GoListReader` with edges spanning two prefixes verifies cross-prefix
  edges are filtered out.
- Integration: `dagnabit -n` should show the expected moves listed above.
