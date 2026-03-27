# Pandoc Internal Formatting: Blob-Backed Type Configuration

**Date:** 2026-03-27 **FDR:** 0010 (Core Types) --- pathfinding experiment
**Scope:** Store pandoc Lua filters and defaults files as blobs in the object
graph, wire them into the `!md` type formatter, validate with end-to-end
markdown formatting

## Problem

The `!md` type has no formatter. Pandoc configuration (Lua filters, defaults
YAML) lives in `zz-pandoc/` and is installed to `~/.local/share/pandoc/` via
symlinks. This creates an external filesystem dependency that violates dodder's
principle: the store is the source of truth.

The existing type blob infrastructure supports formatters (`[formatters]`) and
reference discovery (`[references]`), but neither has been exercised for `!md`.
The formatter and reference discovery scripts can run external commands, and
type objects can carry typed blob references --- but nobody has connected these
capabilities to make tool configuration internal.

## Goals

1.  `dodder init` creates a repo where `!md` formatting works without any
    filesystem pandoc configuration
2.  Pandoc Lua filters and defaults YAML are stored as content-addressed blobs
3.  The `!md` type object references these blobs via typed blob references with
    semantic aliases
4.  Genesis uses `import_plan.Builder` for the full bootstrap (proving the
    builder is loadbearing)
5.  Expose what infrastructure is missing for the broader FDR-0010 vision

## Non-Goals

- FUSE/VFS blob access (future optimization)
- Pandoc binary as blob (stays as nix dependency)
- Reference discovery from markdown (separate experiment)
- Null type (`!`) implementation
- Seed repo distribution (`dodder.net`)

## Design

### Object Graph Structure

Genesis creates five new objects/blobs alongside the existing `!md` type and
repo config:

**New type objects:**

- `!pandoc-defaults` --- type for pandoc defaults YAML files. Type blob:
  `{ file-extension = "yaml" }`
- `!pandoc-lua_filter` --- type for pandoc Lua filter files. Type blob:
  `{ file-extension = "lua" }`. Placeholder `[references]` section for future
  `require()` dependency discovery (not implemented in this experiment).

**New tool blobs** (content `go:embed`'d in the dodder binary):

- `dodder-common.lua` content → blob digest A, typed `!pandoc-lua_filter`
- `dodder-edit.lua` content → blob digest B, typed `!pandoc-lua_filter`
- `dodder-edit.yaml` content → blob digest C, typed `!pandoc-defaults`

**Updated `!md` type object:**

- Type blob gains `[formatters.text]` section invoking pandoc
- Typed blob references with path-convention aliases:
  - `filters/dodder-common.lua = @A !pandoc-lua_filter`
  - `filters/dodder-edit.lua = @B !pandoc-lua_filter`
  - `defaults/dodder-edit.yaml = @C !pandoc-defaults`

### Genesis Changes

The existing `initDefaultTypeAndConfig` in `genesis.go` already uses
`import_plan.MakeLocalBuilder()`. The change adds new `prepare*` functions that
add objects to the same builder before `builder.Build()`:

1.  `prepareToolTypes(builder)` --- creates `!pandoc-defaults` and
    `!pandoc-lua_filter` type objects, adds to builder
2.  `prepareToolBlobs(builder)` --- writes embedded Lua/YAML content to blob
    store, returns digests
3.  `prepareDefaultType` (modified) --- creates `!md` with formatters in type
    blob, adds blob references using digests from step 2, adds to builder

The builder's topographic ordering handles the dependency chain:
`!pandoc-defaults` and `!pandoc-lua_filter` must be committed before `!md`
(because `!md`'s blob references carry type locks referencing those types).

### Blob Materializer

New function, likely in `juliett/typed_blob_store/` or a sibling package:

    materializeBlobTree(blobStore, blobReferences) → (tmpdir, cleanup, err)

1.  Creates a tmpdir
2.  Iterates blob references on the type object
3.  For each: reads blob from store, writes to `tmpdir/<alias>` (creating
    subdirectories from the alias path)
4.  Returns tmpdir path and cleanup function

The alias-as-path convention means `filters/dodder-edit.lua` creates
`$tmpdir/filters/dodder-edit.lua`. This is a convention bet --- if it proves
fragile, migration to explicit mapping in the formatter config is
straightforward (the materializer is one function).

**Interface designed for future FUSE replacement:** a FUSE/VFS implementation
would replace the internals of this function (mount instead of copy) without
changing the signature.

### Formatter Pipeline Changes

`GetBlobFormatter()` in `op_get_blob_formatter.go` currently receives the full
type object but ignores its blob references. Changes:

1.  After selecting a formatter `ScriptConfig`, collect blob references from the
    type object via `typeObject.GetMetadata().AllBlobReferences()`
2.  Call `materializeBlobTree()` to create tmpdir
3.  Add `DODDER_BLOB_TREE=$tmpdir` to the env vars passed to
    `MakeWriterToWithStdin`
4.  Defer cleanup

The formatter script in the `!md` type blob uses bash for env var expansion:

    [formatters.text]
    description = "Normalize markdown with pandoc"
    script = "pandoc --data-dir=\"$DODDER_BLOB_TREE\" --defaults=dodder-edit"
    file-extension = "md"

### Lua `require()` Resolution

`dodder-edit.lua` calls `require("dodder-common")` via a `package.path` hack
pointing at `~/.local/share/pandoc/filters/`. With materialization, we prepend
`$DODDER_BLOB_TREE/filters/` to `LUA_PATH` in the script environment so pandoc's
Lua runtime resolves `require("dodder-common")` from the materialized tree.

**Future:** dodder's native Lua support could provide a custom loader that reads
directly from the blob store, eliminating the materialization for Lua
dependencies entirely.

### Embedded Content

The Lua filter and defaults YAML content from `zz-pandoc/` is embedded via
`go:embed` directives. The embedded files are:

- `zz-pandoc/filters/dodder-common.lua`
- `zz-pandoc/filters/dodder-edit.lua`
- `zz-pandoc/defaults/dodder-edit.yaml`

Note: `dodder-edit.yaml` references `dodder-edit.lua` by filename. With
materialization, this resolves naturally because the defaults file and filter
land in the same `--data-dir` tree.

### `dodder-edit.yaml` Modifications

The defaults file needs one change: the filter path must be relative to the
data-dir rather than relying on `~/.local/share/pandoc/`:

    filters:
      - dodder-edit.lua

This already works because pandoc searches `<data-dir>/filters/` for Lua
filters.

## Flagged for Future

1.  **`!pandoc-lua_filter` reference discovery** --- the type should have a
    `[references]` config that parses `require()` calls to discover Lua
    dependencies. This would make the `dodder-edit.lua → dodder-common.lua` edge
    explicit in the graph.
2.  **FUSE/VFS materialization** --- replace tmpdir with a FUSE mount backed by
    blob store reads. The materializer interface is designed for this.
3.  **Named pipes** --- for simple single-file blobs, `/dev/fd/` or `mkfifo`
    could avoid disk writes entirely.
4.  **Pandoc binary as blob** --- long-term, the `dodder.net` seed repo hosts
    the pandoc binary as a platform-specific blob with lazy evaluation.
5.  **Native Lua blob loader** --- dodder's Lua runtime provides a custom
    `require()` that reads from the blob store, eliminating materialization for
    Lua filter dependencies.

## Rollback

Revert the genesis changes. Existing repos are unaffected --- their `!md` type
blob has no formatters and no blob references. New repos created after revert
get the old minimal `!md`. No dual-architecture period needed because this is
additive (no existing behavior changes).

## Success Criteria

    $ dodder init
    $ echo '# Hello World' | dodder new -
    $ dodder show .z
    # Output is pandoc-normalized markdown
    # No ~/.local/share/pandoc/ dependency required
    $ dodder show :t
    # Shows: !md, !pandoc-defaults, !pandoc-lua_filter

## Implementation Status (2026-03-27)

### Completed (branch `vivid-cherry`)

1.  Builtin type registration for `!pandoc-defaults` and `!pandoc-lua_filter`
2.  Type blob constructors (`DefaultPandocDefaults`, `DefaultPandocLuaFilter`,
    `DefaultWithPandocFormatter`)
3.  Embedded pandoc files via `go:embed` in
    `sierra/local_working_copy/embedded/`
4.  Genesis expansion gated behind `-include-default-pandoc-tools` flag
5.  Blob tree materializer (`MaterializeBlobTree` on `local_working_copy.Repo`)
6.  Formatter pipeline wiring --- `DODDER_BLOB_TREE` env var injected into
    pandoc process in both `format-object` and `format-blob` commands
7.  Context-managed cleanup via `errors.Context.After` (no manual defer at call
    sites)
8.  Fixed `dodder-common.lua` --- replaced `require("url")` with inline
    `url_unescape()` (pandoc's Lua runtime has no third-party modules)
9.  Integration tests: genesis creates tool types + end-to-end
    `format-blob -stdin` with absolute assertions

### Design Issues Identified

The prototype works but the wiring is hardcoded:

- Call sites unconditionally call `MaterializeBlobTree` and inject
  `DODDER_BLOB_TREE` --- materialization should be driven by the type config
- The env var name `DODDER_BLOB_TREE` is baked into Go code in two places

### Next Steps (issues)

- **#66** --- `[formatters.*.fs]` in type blob config. The formatter declares
  which blob reference aliases it needs materialized and at what paths.
  `DODDER_SANDBOX` env var set to tmpdir root only when `fs` entries are
  present. Replaces hardcoded `MaterializeBlobTree` + `DODDER_BLOB_TREE`.

  ``` toml
  [formatters.text]
  script = 'pandoc --data-dir="$DODDER_SANDBOX" --defaults=dodder-edit'

  [formatters.text.fs]
  "filters/dodder-common.lua" = "filters/dodder-common.lua"
  "filters/dodder-edit.lua" = "filters/dodder-edit.lua"
  "defaults/dodder-edit.yaml" = "defaults/dodder-edit.yaml"
  ```

- **#67** --- Type type (`!toml-type-v1`) validation: `ParseTypedBlob` should
  validate that `fs` alias values exist as blob reference aliases on the type
  object's metadata.

- Lua library import mechanism --- filters must inline third-party Lua until we
  add blob-tree-aware `require()` resolution (TODO in `dodder-common.lua`).
