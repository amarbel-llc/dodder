# NATO Phonetic Module Hierarchy

The source tree under `go/src/` enforces a strict dependency DAG using NATO
phonetic alphabet names. Each layer may only import from layers below it. The
directory name alone makes the dependency direction visible.

## Layer Breakdown

### Layer 0: `_` (External/Vendored)

**Packages:** bech32, box_chars, coordinates, equality, exec, external_state,
flag_policy, hecks, http_statuses, interfaces, mcp, object_change_type,
ohio_buffer, primordial, reflexive_interface_generator, reset, stack_frame,
token_types, vim_cli_options_builder

External and vendored packages that do NOT depend on any NATO layer. These
provide fundamental Go interfaces and type definitions used across the entire
codebase. The `interfaces` package here defines core types like `PoolPtr`,
`FuncRepool`, `Sha`, `Ptr`, `ResetablePtr`, and `ShaWriteCloser`.

### Layer 1: alfa

**Packages:** pool, errors, interfaces, domain_interfaces, collections_coding,
collections_map, cmp, equals, quiter_collection, quiter_seq, analyzers,
reflexive_interface_generator, unicorn

Foundational primitives with no dependencies except stdlib and the `_` layer.
Defines the generic pool with `GetWithRepool`/reset semantics, custom error
handling with context and stack traces, generic collection types (maps, coding
helpers), and iterator/sequence protocols (`quiter_*`). The `analyzers` package
contains the repool static analyzer.

**Notable types and interfaces:** `pool.Pool`, `errors.Err`, `FuncRepool` (via
`_/interfaces`), `collections_map.Map`, `quiter_seq.Seq`.

### Layer 2: bravo

**Packages:** markl_io, blech32, blob_store_id, checkout_mode, collections_slice,
comments, delim_reader, env_vars, flags, lua, ohio, options_tools,
organize_text_mode, quiter, quiter_set, remote_connection_types,
string_builder_joined, ui, values

Low-level I/O and utility packages. Provides the MARKL binary format with SHA
checksums (`markl_io`), BLECH32 encoding for human-readable identifiers, Lua
runtime integration, delimiter-based reading, terminal UI utilities, flag parsing
support, and slice-based collection helpers.

**Notable types and interfaces:** `markl_io.Writer`, `markl_io.Reader`,
`blech32.Encoder`, `blob_store_id.BlobStoreId`, `ui.Printer`.

### Layer 3: charlie

**Packages:** catgut, doddish, collections, collections_ptr, collections_value,
checkout_options, cli, compression_type, delim_io, expansion, files, genres,
heap, key_bytes, options_print, script_config, store_version, toml, tridex,
trie, xdg_defaults

Core domain types. String interning and caching (`catgut`) provides efficient
string storage. Tokenized identifier parsing (`doddish`) handles sequences,
parts, and ranges for the two-part ID system. Store version constants, checkout
mode enums, TOML configuration support, and generic heap/trie data structures
live here.

**Notable types and interfaces:** `catgut.String`, `doddish.Sequence`,
`store_version.StoreVersion`, `compression_type.CompressionType`,
`collections.Set`.

### Layer 4: delta

**Packages:** age, alfred, collections_delta, debug, editor, file_extensions,
key_strings, ohio_files, script_value, string_format_writer, thyme, xdg

Development, debug, and encryption tools. Age encryption integration (`age`)
handles public/private key operations. Editor launching (`editor`) opens external
editors. Debug mode features (`debug`) activate with the `debug` build tag. The
`alfred` package provides helper utilities, and `thyme` offers time-related
operations.

**Notable types and interfaces:** `age.Encrypter`, `age.Decrypter`,
`editor.Editor`, `xdg.XDG`.

### Layer 5: echo

**Packages:** ids, format, descriptions, markl, checked_out_state,
config_cli, directory_layout, inventory_archive

Object ID and format system. All object ID types and genres live in `ids`,
including `ObjectId`, `ZettelId`, `Type`, `Tag`, `RepoId`, and genre
definitions. The `format` package provides formatting utilities. MARKL format
helpers, checked-out state tracking, and inventory archive types complete this
layer.

**Notable types and interfaces:** `ids.ObjectId`, `ids.ZettelId`, `ids.Type`,
`ids.Tag`, `ids.Genre`, `checked_out_state.State`, `format.Formatter`.

### Layer 6: foxtrot

**Packages:** triple_hyphen_io, markl_age_id, fd, object_id_provider, page_id,
repo_config_cli, store_workspace, tag_paths

Format versioning and repository page management. The `triple_hyphen_io` package
handles versioned serialization formats (V0, V1, V2, etc.) using the
triple-hyphen delimiter. MARKL+Age ID generation (`markl_age_id`) combines
content hashing with encryption. Tag path management (`tag_paths`) handles
hierarchical tag structures.

**Notable types and interfaces:** `triple_hyphen_io.FormatVersion`,
`page_id.PageId`, `tag_paths.TagPath`, `object_id_provider.Provider`.

### Layer 7: golf

**Packages:** blob_store_configs, env_ui, id_fmts, objects, repo_blobs,
repo_configs

Command and object handling. Blob store configuration types
(`blob_store_configs`) define how blob stores are configured. Environment/UI
integration (`env_ui`) connects the UI layer to the environment. Repository
configuration (`repo_configs`) and blob management (`repo_blobs`) handle
repository-level settings. ID formatting (`id_fmts`) provides display formats
for identifiers.

**Notable types and interfaces:** `blob_store_configs.Config`, `env_ui.Env`,
`objects.Object`, `repo_configs.RepoConfig`.

### Layer 8: hotel

**Packages:** env_dir, file_lock, genesis_configs, object_fmt_digest,
object_metadata_box_builder, object_metadata_fmt, workspace_config_blobs

File and directory management. Environment directory management (`env_dir`)
handles the `.dodder/` directory structure. File-based locking (`file_lock`)
prevents concurrent access. Initial configuration generation
(`genesis_configs`) bootstraps new repositories. Object metadata formatting and
workspace configuration blob storage round out this layer.

**Notable types and interfaces:** `env_dir.EnvDir`, `file_lock.Lock`,
`genesis_configs.GenesisConfig`, `object_metadata_fmt.Formatter`.

### Layer 9: india

**Packages:** blob_stores, env_local, object_metadata_fmt_triple_hyphen,
zettel_id_index

Storage and indexing. Concrete blob store implementations (`blob_stores`)
provide the actual storage backends. Local environment setup (`env_local`)
manages the local dodder environment. Triple-hyphen metadata formatting
(`object_metadata_fmt_triple_hyphen`) serializes metadata in the triple-hyphen
format. Zettel ID indexing (`zettel_id_index`) enables fast zettel lookups.

**Notable types and interfaces:** `blob_stores.BlobStore`, `env_local.EnvLocal`,
`zettel_id_index.Index`.

### Layer 10: juliett

**Packages:** sku, command, env_repo

SKU and core objects. The central `sku.Transacted` type represents all versioned
objects in the system. The `command` package defines the command execution
interface. Repository environment (`env_repo`) manages the full repository
context. Pool management for `Transacted` is accessed via
`sku.GetTransactedPool()`.

**Notable types and interfaces:** `sku.Transacted`, `sku.CheckedOut`,
`sku.Proto`, `sku.TransactedResetter`, `command.Command`, `env_repo.Env`.

### Layer 11: kilo

**Packages:** alfred_sku, blob_library, blob_transfers, box_format,
command_components, command_components_madder, dormant_index, log_remote_inventory_lists,
object_finalizer, object_probe_index, sku_json_fmt, sku_lua, store_abbr

Intermediate store, queries, and command building blocks. Blob library
management (`blob_library`) handles blob collections. The box (text) format
(`box_format`) serializes objects in a human-readable text format. Archived
object indexing (`dormant_index`) tracks inactive objects. Command building
components provide reusable pieces for assembling CLI commands. SKU JSON and Lua
formatting enable alternative output formats.

**Notable types and interfaces:** `blob_library.Library`, `box_format.Writer`,
`dormant_index.Index`, `command_components.Component`, `object_finalizer.Finalizer`.

### Layer 12: lima

**Packages:** commands_madder, env_lua, inventory_list_coders, stream_index,
tag_blobs, type_blobs

Working copy, serialization coders, and stream indexing. Inventory list coders
(`inventory_list_coders`) handle encoding/decoding of inventory list entries.
The stream index (`stream_index`) provides binary page-based indexing for fast
object access. Tag and type blob packages manage blob storage for tags and
types. The `commands_madder` package defines the madder tool's commands.

**Notable types and interfaces:** `stream_index.Index`,
`inventory_list_coders.Coder`, `env_lua.Env`.

### Layer 13: mike

**Packages:** inventory_list_store, sku_fmt, typed_blob_store

Store format and serialization. The inventory list store
(`inventory_list_store`) implements the append-only inventory log that serves as
the source of truth. SKU serialization (`sku_fmt`) handles gob and box format
encoding. The typed blob store (`typed_blob_store`) provides generic,
compile-time type-safe blob storage via `BlobStore[T, TPtr]`.

**Notable types and interfaces:** `inventory_list_store.Store`,
`sku_fmt.Formatter`, `typed_blob_store.BlobStore`.

### Layer 14: november

**Packages:** queries

Query execution. Implements tag queries, type queries, boolean combinators
(AND/OR/NOT), full-text search, and Lua scripting-based filtering. All query
types compose through a common query interface, enabling arbitrarily complex
filter expressions.

**Notable types and interfaces:** `queries.Query`, `queries.Builder`.

### Layer 15: oscar

**Packages:** organize_text, store_workspace

Workspace and organization. The `organize_text` package provides a text-based
bulk organization interface where objects are presented as editable text and
changes are parsed back into operations. The `store_workspace` package manages
workspace-level store interactions.

**Notable types and interfaces:** `organize_text.Organizer`,
`store_workspace.Store`.

### Layer 16: papa

**Packages:** store_fs

Filesystem blob store with SHA bucketing. Stores blob content by SHA hash using
Git-like bucketing where the first two hex characters of the hash serve as the
directory name. Handles atomic file moves, temporary file creation, and
immutable file locking.

**Notable types and interfaces:** `store_fs.Store`.

### Layer 17: quebec

**Packages:** env_workspace

Workspace environment setup. Manages the workspace-level environment, including
checked-out file tracking, workspace configuration, and the relationship between
the repository store and the working directory.

**Notable types and interfaces:** `env_workspace.Env`.

### Layer 18: romeo

**Packages:** store_config

Configuration store. Manages two configuration formats: `config-seed` (text
format, human-readable, editable) and `config-mutable` (gob cache, derived from
seed, rebuilt on demand). Handles configuration migration between store versions.

**Notable types and interfaces:** `store_config.Store`.

### Layer 19: sierra

**Packages:** store_browser, env_box

UI and browsing. The `store_browser` package provides an object browsing
interface for interactive exploration. The `env_box` package manages the box
format environment for text-based object display.

**Notable types and interfaces:** `store_browser.Browser`, `env_box.Env`.

### Layer 20: tango

**Packages:** repo, store

Repository abstraction and main store interface. The `repo` package provides the
top-level repository operations (open, init, close). The `store` package
defines the main store interface that ties together inventory lists, blob stores,
indexes, and queries into a unified API.

**Notable types and interfaces:** `repo.Repo`, `store.Store`.

### Layer 21: uniform

**Packages:** remote_transfer

Remote sync implementation. Handles push and pull operations for synchronizing
inventory lists and blobs between local and remote repositories. Manages
connection setup, transfer negotiation, and conflict detection.

**Notable types and interfaces:** `remote_transfer.Transfer`.

### Layer 22: victor

**Packages:** local_working_copy

Local working copy management. Handles the checked-out files in the working
directory, including checkout, checkin, and status operations for the local
filesystem representation of repository objects.

**Notable types and interfaces:** `local_working_copy.WorkingCopy`.

### Layer 23: whiskey

**Packages:** remote_http, user_ops

Remote HTTP transport and user operations. The `remote_http` package implements
HTTP-based remote repository communication. The `user_ops` package provides
high-level user-facing operations that compose lower-level store and repository
operations.

**Notable types and interfaces:** `remote_http.Client`, `user_ops.Ops`.

### Layer 24: xray

**Packages:** command_components_dodder

Command components specific to the dodder/der binary. Provides reusable command
building blocks that are specific to the primary user-facing tool, composing
lower-level command components from kilo with dodder-specific behavior.

**Notable types and interfaces:** `command_components_dodder.Component`.

### Layer 25: yankee

**Packages:** commands_dodder

Top-level user-facing commands registered for the dodder/der binary. Each
command struct implements the `command.Command` interface from juliett. This is
the highest application layer and may import from any layer below it.

**Notable types and interfaces:** Individual command structs (e.g.,
`commands_dodder.Init`, `commands_dodder.New`, `commands_dodder.Show`).

## Dependency Rules

### The Rule

A package in layer N may only import from layers 0 through N-1. It must never
import from layer N (sibling) or any layer above N.

### Correct Example

`echo/ids` (layer 5) importing `charlie/catgut` (layer 3) and `alfa/errors`
(layer 1):

```go
import (
    "code.linenisgreat.com/dodder/go/src/alfa/errors"
    "code.linenisgreat.com/dodder/go/src/charlie/catgut"
)
```

This is valid because layer 5 may import from layers 1 through 4.

### Wrong Example

`charlie/catgut` (layer 3) importing `echo/ids` (layer 5):

```go
import (
    "code.linenisgreat.com/dodder/go/src/echo/ids" // WRONG: layer 3 cannot import layer 5
)
```

This violates the hierarchy because a lower layer imports from a higher layer.

### Placing a New Package

To decide where a new package belongs:

1. List every NATO layer it must import from.
2. Identify the highest layer in that list.
3. Place the new package one layer above that highest dependency.
4. If no NATO imports are needed, place it in `alfa` (or `_` if it has zero
   dodder dependencies).
5. Prefer the lowest valid layer. Promote to a higher layer later if new
   dependencies arise.
