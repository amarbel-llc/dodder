---
name: Dodder Development
description: >
  Helps contributors work on the dodder Go codebase. Covers the build and test
  cycle, NATO phonetic module hierarchy, pool management rules, the repool
  static analyzer, codebase navigation, and critical coding conventions.
  Activated by requests to work on dodder, add commands or types, register
  coders, fix dodder code, add NATO modules, manage pools, or explore dodder
  architecture and development workflows.
triggers:
  - work on dodder
  - add a new command
  - add a type
  - register a coder
  - fix dodder code
  - add a NATO module
  - repool
  - pool management
  - dodder architecture
  - dodder development
---

# Dodder Development

## Overview

Dodder is a distributed zettelkasten written in Go. It provides content-addressable
blob storage with automatic two-part IDs, version-controlled metadata via inventory
lists, and sophisticated querying including tag, type, boolean, full-text, and Lua
filters.

Two user-facing binaries exist: `dodder` and `der` (same tool, shorter name). A
third binary, `madder`, is low-level and intended for internal or advanced
repository operations. All three share the same Go module and NATO-layered source
tree.

The project follows strict architectural rules: a NATO phonetic module hierarchy
enforces dependency direction, pool management requires disciplined repool
semantics, and `sku.Transacted` pointers must never be dereferenced.

## Build and Test Cycle

### Building

Run `just build` to compile debug and release Go binaries. Debug binaries land in
`go/build/debug/` and release binaries in `go/build/release/`. The debug build
includes the `debug` build tag, which enables runtime pool poisoning and additional
assertions. Build before running integration tests if binaries are stale.

### Testing

| Command | Scope |
|---------|-------|
| `just test` | All tests (unit + integration) |
| `just test-go` | Unit tests only (`go test -tags test,debug ./...`) |
| `just test-bats` | Integration tests (builds first, generates fixtures, runs BATS) |
| `just test-bats-targets clone.bats` | Specific BATS test files |
| `just test-bats-tags migration` | Filter integration tests by tag |

### Code Quality

| Command | Purpose |
|---------|---------|
| `just check` | Run all checks (vuln + vet + repool analyzer) |
| `just codemod-go-fmt` | Format code (goimports + gofumpt) |

`just check` chains three checks: `govulncheck ./...` for known vulnerabilities,
`go vet ./...` for standard static analysis, and the custom repool analyzer that
detects leaked or discarded pool return functions.

### Fixture Workflow

Integration tests use committed fixtures in `zz-tests_bats/migration/`. When code
changes alter the store format or binary layout, regenerate fixtures:

1. Run `just test-bats-update-fixtures` to rebuild fixture data.
2. Review the diff with `git diff -- zz-tests_bats/migration/`.
3. Stage and commit the regenerated fixtures before pushing.

Fixture generation requires a working debug binary, so `just build` runs
automatically as a prerequisite. See `references/testing-workflow.md` for the full
fixture lifecycle and how to write new integration tests.

## NATO Phonetic Module Hierarchy

The source tree in `go/src/` uses NATO phonetic alphabet names to enforce a strict
dependency DAG. Each layer may only import from layers below it alphabetically.
This prevents circular dependencies and makes the dependency direction visible in
the directory name alone.

| Layer | Key Packages | Purpose |
|-------|-------------|---------|
| **alfa** | pool, errors, interfaces, collections, analyzers | Foundational primitives (pools, errors, interfaces, collections, analyzers) |
| **bravo** | markl_io, blech32, ui, flags | Low-level I/O and utilities |
| **charlie** | catgut, doddish, checkout_options, store_version | Core domain types (string interning, tokenized IDs, checkout options) |
| **delta** | alfred, age, editor | Development and debug tools (age encryption, editor integration) |
| **echo** | ids, format, markl, checked_out_state, blob_store_configs | Object ID and format system |
| **foxtrot** | triple_hyphen_io, tag_paths | Repository and page management (format versioning, tag paths) |
| **golf** | command, env_ui, objects | Command and object handling |
| **hotel** | env_dir, file_lock, genesis_configs | File and directory management |
| **india** | blob_stores, zettel_id_index | Storage and indexing |
| **juliett** | sku, command, env_repo | SKU and core objects (`sku.Transacted`, command interface) |
| **kilo** | blob_library, box_format, dormant_index | Intermediate store and queries |
| **lima** | checked_out_sku, merge_state, remote_inventory_lists | Working copy and collaboration |
| **mike** | inventory_list_store, sku_fmt | Store format and serialization |
| **november** | queries | Query execution (tag/type/boolean, full-text, Lua) |
| **oscar** | organize_text | Workspace and organization |
| **papa** | store_fs | Filesystem store (SHA bucketing) |
| **quebec** | env_workspace | Workspace environment |
| **romeo** | store_config | Configuration store (config-seed text, config-mutable gob) |
| **sierra** | store_browser | UI and browsing |
| **tango** | repo, store | Repository operations and store interfaces |
| **uniform** | push, pull | Remote transfer |
| **yankee** | commands_dodder | Top-level commands |

Refer to `references/nato-hierarchy.md` for the full breakdown of every package in
each layer, dependency rules, and correct versus incorrect import examples.

### Dependency Rule

A package in layer N may import any package from layers 1 through N-1 but never
from layer N or above. For example, `echo/ids` (layer 5) may import
`charlie/catgut` (layer 3) and `alfa/errors` (layer 1), but it must never import
`foxtrot/` or higher.

When adding a new package, place it in the lowest layer whose dependencies it
requires. When in doubt, prefer a lower layer and promote later if needed.

## Critical Rules

### Never Dereference sku.Transacted Pointers

Never use `*object` to copy an `sku.Transacted` value. This violates pool
management and produces subtle corruption. Instead, use
`sku.TransactedResetter.ResetWith(&dst, src)` to copy field data safely:

```go
// CORRECT: reset target from source without dereferencing
var local sku.Transacted
sku.TransactedResetter.ResetWith(&local, sourcePointer)
```

```go
// WRONG: direct dereference — never do this
value := *sourcePointer
```

For persistent objects, clone from the pool and defer the return:

```go
cloned := original.CloneTransacted()
defer sku.GetTransactedPool().Put(cloned)
```

### Always Repool After GetWithRepool

`GetWithRepool()` returns `(element, FuncRepool)`. Call the repool function exactly
once when the caller is done with the element. Three enforcement layers exist:

1. **Static analyzer** (`just check-go-repool`): A CFG-based `go vet` checker in
   `src/alfa/analyzers/repool/`. It detects discarded repool functions (blank `_`
   without `//repool:owned`) and repool variables not called on all code paths.

2. **Runtime debug poisoning** (build tag `debug`): Wraps every repool function
   with an `atomic.Bool` guard that panics on double-repool. Tracks outstanding
   borrows via `pool.OutstandingBorrows()`. Zero overhead in release builds.

3. **Lint check** (`bin/lint.bash`): Grep-based check for discarded repool
   functions missing the `//repool:owned` annotation.

Use the `//repool:owned` comment to suppress the analyzer when intentionally
discarding a repool function, such as when a hash's lifetime extends beyond the
scope (see `references/pool-management.md` for the hash-lifetime pattern).

```go
hash, _ := config.hashFormat.GetHash() //repool:owned
writer.digester = markl_io.MakeWriter(hash, nil)
```

Refer to `references/pool-management.md` for detailed repool semantics, common
mistakes, and debugging pool leaks.

### Read MIGRATION_LEARNINGS.md Before Touching Critical Code

Before modifying ObjectId, the stream index, store version code, or gob
serialization paths, read `MIGRATION_LEARNINGS.md` at the repository root. It
documents hard-won lessons from previous migration attempts, including hidden
dependencies, binary format details, and the correct store initialization order.

### Format Before Committing

Run `just codemod-go-fmt` before committing. This chains `goimports` (to organize
imports) and `gofumpt` (to enforce stricter formatting). The CI pipeline expects
code formatted by these tools.

## Codebase Navigation

### Binary Entry Points

| Path | Binary | Purpose |
|------|--------|---------|
| `go/cmd/dodder/` | dodder | Primary user-facing CLI |
| `go/cmd/der/` | der | Short alias for dodder (same tool) |
| `go/cmd/madder/` | madder | Low-level repository operations |

### Source Tree

All Go source lives under `go/src/`, organized by NATO layer:

- `go/src/alfa/` through `go/src/yankee/` contain the module packages.
- Each NATO directory holds one or more Go packages (e.g., `go/src/echo/ids/`,
  `go/src/echo/format/`).
- Import paths follow the form
  `code.linenisgreat.com/dodder/go/src/{layer}/{package}`.

### Tests

- **Unit tests:** `*_test.go` files alongside source throughout `go/src/`.
- **Integration tests:** `zz-tests_bats/` contains 40+ `.bats` files using the
  BATS framework.
- **Versioned fixtures:** `zz-tests_bats/migration/` holds committed test data
  organized by store version.

### Documentation

| File | Purpose |
|------|---------|
| `CLAUDE.md` | Development conventions and quick-reference commands |
| `MIGRATION_LEARNINGS.md` | Critical migration do-not-repeat mistakes |
| `docs/plans/` | Design and implementation documents |

## Adding a New Command

Commands live in `go/src/yankee/commands_dodder/`. Each command is a Go struct
implementing the command interface defined in `go/src/juliett/command/`. Follow
these steps:

1. Create a new file in `go/src/yankee/commands_dodder/` following the naming
   pattern of existing commands.
2. Implement the `command.Command` interface: define flags via `flag.FlagSet`,
   implement the `Run` method with request context and configuration.
3. Register the command in the command map (check existing registrations for the
   pattern).
4. Use `alfa/errors` for error handling — wrap errors with `errors.Wrap()` or
   `errors.Wrapf()`.
5. Run `just codemod-go-fmt` and `just check` before committing.
6. Add integration tests in `zz-tests_bats/` if the command has user-visible
   behavior.

## Adding a New Type or Coder

When adding blob store types or registering new coders:

1. **Type constant:** Add the type constant to `go/src/echo/ids/types_builtin.go`.
2. **Init registration:** Register the type in the `init()` function of the same
   file.
3. **Coder registration:** Add the type-to-coder mapping in the appropriate IO
   file (e.g., `go/src/echo/blob_store_configs/io.go`).
4. **Interface implementation:** Implement the relevant interface from
   `go/src/alfa/interfaces/` or `go/src/alfa/domain_interfaces/`. Use existing
   implementations as templates.
5. **Triple-hyphen format:** If the type is serialized through the triple-hyphen
   IO system (`foxtrot/triple_hyphen_io`), add the format version mapping.

## Key Architectural Patterns

### Content-Addressable Storage

Blob content is stored by SHA hash using Git-like bucketing (first two hex
characters as the directory name). The `papa/store_fs` package implements the
filesystem store. SHA writers are created via `sha.MakeWriter()` (not
`sha.NewWriter()`), and `interfaces.Sha` is already a pointer type — never use
`*interfaces.Sha`.

### Inventory Lists as Source of Truth

Inventory lists (text format, stored in
`.dodder/local/share/inventory_lists_log`) are the authoritative record of all
objects. The stream index (binary pages in `.dodder/local/share/objects_index/`)
and config-mutable (gob cache) are derived and can be rebuilt. When debugging data
issues, start by examining inventory lists.

### Store Initialization Order

1. Initialize inventory list store.
2. Build working list.
3. Create zettel ID index.
4. Create stream index via `stream_index.MakeIndex` (lazy — pages read on first
   query).

### Versioned Structs with Typed Blob Store

The system uses generic typed blob stores (`typed_blob_store.BlobStore[T, TPtr]`)
for compile-time type safety. Multiple struct versions implement the same
interface, enabling backward-compatible format evolution. Common interfaces live in
`go/src/alfa/interfaces/` (e.g., `BlobStoreConfigImmutable`), with versioned
implementations in the appropriate NATO layer.

## Reference Documents

- `references/nato-hierarchy.md` — Full layer breakdown with package contents,
  dependency rules, correct and incorrect import examples.
- `references/pool-management.md` — Repool semantics, the static analyzer, runtime
  poisoning, common mistakes, and debugging pool leaks.
- `references/testing-workflow.md` — BATS fixture lifecycle, writing new integration
  tests, fixture regeneration, and tag-based test filtering.
