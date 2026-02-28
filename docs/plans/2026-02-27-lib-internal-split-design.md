# lib/internal Split Design

## Problem

All Go packages live under `go/internal/` regardless of whether they are
domain-agnostic utilities or dodder/madder-specific code. This makes it hard to
identify reusable infrastructure and creates no structural boundary between
generic libraries and application logic.

## Goal

Split `go/internal/` into two trees:

- `go/lib/` — domain-agnostic packages with no dodder/madder concepts. Maintains
  the same NATO phonetic hierarchy (`_`, alfa, bravo, ...).
- `go/internal/` — dodder/madder-specific packages. Same NATO hierarchy,
  everything that remains after lib extraction.

Both directories stay within the same Go module
(`code.linenisgreat.com/dodder/go`).

## Approach: Two-Phase Execution

### Phase 1: Split `_/interfaces`

Move domain-specific interfaces out of `_/interfaces` into
`alfa/domain_interfaces` (which already exists):

| Interface | Source File | Reason |
|-----------|------------|--------|
| `StoreVersion` | `store_version.go` | Dodder schema versioning |

Note: `ObjectId`, `MarklIdGetter`, and blob-related interfaces are already in
`alfa/domain_interfaces`. No `BlobId` interface exists — the design uses
`MarklId` instead.

Everything else stays in `_/interfaces` — including `Value`, `ValuePtr`,
`DirectoryLayout*`, `ContextState`, all generic contracts (coders, collections,
iterators, pools, errors, I/O, stringers, keyers, elements, locks, printers, CLI
flags).

Update all importers that reference the moved interfaces.

### Phase 2: Rename and Move

1. `git mv go/src go/internal`
2. Create `go/lib/{_,alfa,bravo,charlie,delta,echo}`
3. `git mv` each lib-bound package from `go/internal/` to `go/lib/`
4. Update all import paths:
   - `code.linenisgreat.com/dodder/go/internal/` →
     `code.linenisgreat.com/dodder/go/lib/` (for lib packages)
   - `code.linenisgreat.com/dodder/go/internal/` →
     `code.linenisgreat.com/dodder/go/internal/` (for internal packages)
5. Fix compilation, run tests

## Package Classification

### Packages Moving to `go/lib/` (62 packages)

#### lib/_/ (15 packages)

- `bech32` — BIP173 encoding
- `box_chars` — box drawing characters
- `equality` — equality utilities
- `exec` — process execution wrapper
- `flag_policy` — flag parsing policy
- `hecks` — hex utilities
- `http_statuses` — HTTP status codes
- `interfaces` — generic interface contracts (after domain interface extraction)
- `mcp` — generic MCP protocol utilities
- `ohio_buffer` — buffer utilities
- `primordial` — terminal I/O
- `reflexive_interface_generator` — code generation
- `reset` — reset constraints
- `stack_frame` — stack frame inspection
- `vim_cli_options_builder` — CLI option builder

#### lib/alfa/ (11 packages)

- `analyzers` — static analysis (repool, seqerror)
- `cmp` — comparison utilities
- `collections_coding` — collection encoding
- `collections_map` — map utilities
- `equals` — equality checking
- `errors` — error handling system
- `pool` — object pooling
- `quiter_collection` — collection iteration
- `quiter_seq` — sequence iteration
- `reflexive_interface_generator` — interface generation
- `unicorn` — unicode utilities

#### lib/bravo/ (12 packages)

- `collections_slice` — slice utilities
- `comments` — comment parsing
- `delim_reader` — delimiter reader
- `env_vars` — environment variables
- `flags` — flag parsing
- `lua` — Lua integration wrapper
- `ohio` — binary codec format
- `quiter` — iterator operations
- `quiter_set` — set iteration
- `string_builder_joined` — string builder
- `ui` — UI rendering (borders, tables, colors)
- `values` — value type utilities

#### lib/charlie/ (15 packages)

- `catgut` — string manipulation
- `cli` — CLI framework
- `collections` — collection abstractions
- `collections_ptr` — pointer collections
- `collections_value` — value collections
- `compression_type` — compression wrappers (gzip/zlib/zstd)
- `delim_io` — delimiter I/O
- `expansion` — hierarchical string expansion
- `files` — file utilities
- `heap` — heap data structure
- `script_config` — script execution config
- `toml` — TOML codec
- `tridex` — three-way index
- `trie` — trie data structure
- `xdg_defaults` — XDG base directory templates

#### lib/delta/ (8 packages)

- `age` — age encryption wrapper
- `alfred` — Alfred workflow JSON
- `collections_delta` — set delta calculation
- `debug` — debug utilities
- `editor` — external editor integration
- `ohio_files` — file copy utilities
- `script_value` — script execution value
- `thyme` — date/time utilities

#### lib/echo/ (1 package)

- `config_cli` — CLI configuration

### Packages Staying in `go/internal/` (everything else)

All packages from echo through yankee (except `config_cli`), plus these
lower-tier packages:

- `_/coordinates`, `_/external_state`, `_/object_change_type`, `_/token_types`
- `alfa/domain_interfaces`
- `bravo/blob_store_id`, `bravo/checkout_mode`, `bravo/markl_io`,
  `bravo/options_tools`, `bravo/organize_text_mode`,
  `bravo/remote_connection_types`
- `charlie/checkout_options`, `charlie/doddish`, `charlie/genres`,
  `charlie/key_bytes`, `charlie/options_print`, `charlie/store_version`
- `delta/file_extensions`, `delta/key_strings`, `delta/string_format_writer`
- All of echo through yankee (except `echo/config_cli`)

### Refactoring Candidates (internal/ now, lib/ later)

These packages are mostly domain-agnostic but have small domain-specific
dependencies that could be extracted:

- `echo/directory_layout` — extract generic path generation from
  `blob_store_id.Id` dependency
- `delta/string_format_writer` — evaluate for future extraction

### Already Extracted

- `echo/xdg` — moved to `lib/echo/xdg` (extracted `GetLocationType`, made
  override env var configurable)
- `india/env_dir` TemporaryFS + hash bucket utils — moved to `lib/delta/files`
  (aliases left in `env_dir`)
- `foxtrot/format` — moved to `lib/delta/format` (replaced
  `string_format_writer.LenStringMax` with local constant)

## Import Path Schema

```
# Before
code.linenisgreat.com/dodder/go/lib/alfa/errors

# After (lib package)
code.linenisgreat.com/dodder/go/lib/alfa/errors

# After (internal package)
code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces
```

## Classification Criteria

A package qualifies for `lib/` if and only if:

1. It has no imports from dodder-specific packages (genres, blob_store_id,
   markl_io, key_bytes, doddish, sku, etc.)
2. Its transitive dodder dependencies all resolve to other lib-eligible packages
3. It defines no dodder-specific domain types (object IDs, store versions,
   genres, serialization keys)
