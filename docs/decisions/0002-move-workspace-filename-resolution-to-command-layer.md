---
date: 2026-03-25
status: accepted
---

# Move Workspace Filename Resolution to the Command Layer

## Context and Problem Statement

The query builder's `build_state.build()` unconditionally consulted the
workspace store for every query value, even for repo-only queries like `:z` or
`!md:t`. This conflated two concerns: filename resolution (mapping filesystem
paths to external object IDs for commands like `add` and `checkin`) and
query-time filtering (the `.` sigil selecting external/checked-out objects). How
should the query system separate these concerns so that repo-only queries never
touch the workspace store?

## Decision Drivers

- The MCP bridge required an `IgnoreWorkspace` workaround to prevent
  workspace-aware tools from consulting the workspace store on every query
- A zettel ID matching a checked-out file could be silently claimed as an
  external object ID when no `.` sigil was present
- FDR-0009 (External Object Index) envisions the `.` sigil as a fast index
  filter, not a workspace store query at build time

## Considered Options

1.  Gate workspace store consultation on the `.` sigil in `build_state.build()`
2.  Move filename resolution to the command layer

## Decision Outcome

Chosen option: "Move filename resolution to the command layer", because it
cleanly separates the two concerns without trying to infer intent from query
syntax. Commands that accept filenames resolve them upstream via
`MakeQueryResolvingFilenames`; all other commands pass query terms directly to
the parser.

### Consequences

- Good, because repo-only queries (`show :z`, `show !md:t`) never touch the
  workspace store
- Good, because the MCP bridge `IgnoreWorkspace` workaround can be removed
- Good, because `build_state.build()` is simpler --- it only parses query terms,
  never resolves filesystem paths
- Bad, because `MakeQueryResolvingFilenames` duplicates the try-open-fallback
  pattern that was previously in the build loop
- Bad, because the `"."` operator requires special handling: both resolved via
  workspace store (to pin all external IDs) and passed to the parser (to set
  `dotOperatorActive`)

### Confirmation

All 18 dodder commands pass `just test` (unit + integration) after the
migration. The workspace store loop in `build_state.build()` is removed in
commit `fcee7b7a1`.

## More Information

- https://github.com/amarbel-llc/dodder/issues/55
- [FDR-0009: External Object Index](../features/0009-workspaces-indexes.md) ---
  future direction for `.` sigil as an index filter
- Commit `fcee7b7a1` removes the dead workspace store loop from
  `build_state.build()`
