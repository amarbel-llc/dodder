# Dodder Core Concepts

This document provides a deep reference for the foundational concepts underlying
dodder. It expands on the brief descriptions in the onboarding skill and serves
as the authoritative explanation of each concept.

## Object ID System

Every zettel in dodder receives a unique two-part identifier. Rather than
generating opaque UUIDs or sequential numbers, dodder combines entries from two
user-supplied word lists -- called "yin" and "yang" -- to produce human-friendly
IDs like `one/uno`, `red/apple`, or `summer/breeze`.

### Yin and Yang Lists

During `dodder init`, the `-yin` and `-yang` flags accept file paths (or process
substitutions) containing one word per line. Dodder stores these lists in the
repository at `.dodder/` under `Yin` and `Yang` filenames within the object ID
provider directory. The lists are immutable after initialization -- extending
them requires care, as the index tracks assignments by integer coordinates into
the cross-product.

Example initialization:

```bash
dodder init -yin <(echo -e "one\ntwo\nthree") \
  -yang <(echo -e "uno\ndos\ntres") my-repo
```

This creates a 3x3 ID space with nine possible IDs: `one/uno`, `one/dos`,
`one/tres`, `two/uno`, `two/dos`, `two/tres`, `three/uno`, `three/dos`,
`three/tres`.

### ID Assignment

Dodder assigns IDs from the cross-product in a configurable order. By default,
assignment is random (selecting an unoccupied coordinate pair at random). Setting
`UsePredictableZettelIds` in the repository configuration switches to sequential
assignment, which is useful for testing and deterministic workflows.

The zettel ID index (implemented in `india/zettel_id_index`) tracks which
coordinate pairs are occupied using a bitset. Two implementations exist:

- **v0 (map-based)**: Original implementation using a Go map for coordinate
  tracking.
- **v1 (bitset-based)**: Current implementation using a compact bitset for
  efficient storage and lookup. Persisted via gob encoding with mutex-protected
  concurrent access.

### Peeking at Upcoming IDs

The `peek-zettel-ids` command previews the next N available IDs without creating
any zettels. This is useful for scripting and for understanding the ID space:

```bash
dodder peek-zettel-ids
```

### ID Structure

Each zettel ID consists of a left part (from the yin list) and a right part
(from the yang list), separated by a forward slash. Internally, dodder
represents this as a pair of integer coordinates into the yin and yang arrays.
The string representation is the human-readable form; the coordinate pair is the
storage-efficient form used in binary indexes.

## ID Genres

Dodder uses a genre system to categorize all objects in the store. Every
identifier carries an implicit or explicit genre that determines how the object
is stored, indexed, and displayed. The genres are defined in the
`charlie/genres` package.

### Zettel (`z`)

The primary content type. Zettels are user-created notes or blobs with
automatically assigned two-part IDs from the yin/yang system. Query zettels with
the `:z` genre filter:

```bash
dodder show :z
```

### Tag (`t`)

Organizational labels attached to zettels. Tags appear in output with their
plain name. Query all tags:

```bash
dodder show :t
```

### Type (`e`)

Content format definitions prefixed with `!`. Types define how blob content is
interpreted and serialized. The `e` genre identifier comes from the internal
name. Query all types:

```bash
dodder show :e
```

### Blob

Raw content identified by its SHA hash. Blob identifiers use the `@` prefix
followed by the hash algorithm and digest, for example
`@blake2b256-abc123...`. Blobs are the lowest-level storage unit -- every
zettel, tag definition, type definition, and configuration object has an
associated blob.

### Repo (`r`)

Remote repository identifiers. Repos represent connection configurations for
push/pull synchronization. Repo IDs use a `/` prefix in some contexts (e.g.,
`/my-remote`).

### Config

The reserved system configuration object. Each repository has a single config
object (identified as `konfig`) that stores repository-wide settings. The config
object is versioned like any other object, with its own type
(`!toml-config-v2`).

### Inventory List

Append-only version history records. Inventory lists are identified by TAI
timestamps (e.g., `sec.asec` format). They are internal to the store and not
typically queried directly by users, but they are the authoritative source of
truth for all object state.

### Genre Summary Table

| Genre | Prefix/Syntax | Example | Purpose |
|-------|---------------|---------|---------|
| Zettel | `left/right` | `one/uno` | User notes and content |
| Tag | plain name | `project` | Organizational labels |
| Type | `!` prefix | `!md` | Content format definitions |
| Blob | `@` prefix | `@blake2b256-abc...` | Content-addressed raw data |
| Repo | `/` prefix | `/my-remote` | Remote repository configs |
| Config | `konfig` | `konfig` | Repository configuration |
| Inventory List | TAI timestamp | `1234567890.001` | Version history records |

## Type System

Types in dodder define the content format of objects. Every object has exactly
one type. Types are themselves versioned objects stored in the repository.

### Built-in Types

Dodder ships with a set of built-in types registered at startup. The
authoritative list lives in `echo/ids/types_builtin.go`. Key categories include:

**User content types** -- used for zettel content:

- `!md` -- Markdown (when supported by the configuration)
- `!txt` -- Plain text

**Configuration types** -- internal to dodder:

- `!toml-config-v2` -- Repository configuration (current version)
- `!toml-config-v1` -- Repository configuration (previous version)

**Tag types** -- for tag definitions with behavior:

- `!toml-tag-v1` -- Tag definition in TOML format
- `!lua-tag-v2` -- Tag with Lua scripting behavior

**Type definition types** -- metadata about types themselves:

- `!toml-type-v1` -- Type definition in TOML format

**Blob store configuration types** -- for remote blob storage:

- `!toml-blob_store_config-v2` -- Blob store configuration (current)
- `!toml-blob_store_config_sftp-explicit-v0` -- SFTP blob store (explicit)
- `!toml-blob_store_config_sftp-ssh_config-v0` -- SFTP blob store (via SSH config)
- `!toml-blob_store_config-pointer-v0` -- Blob store pointer
- `!toml-blob_store_config-inventory_archive-v0` -- Inventory archive blob store

**Inventory list types** -- internal version tracking:

- `!inventory_list-v2` -- Current inventory list format
- `!inventory_list-v1` -- Previous inventory list format

**Other internal types:**

- `!toml-repo-uri-v0` -- Remote repository URI
- `!toml-workspace_config-v0` -- Workspace configuration
- `!zettel_id_list-v0` -- Zettel ID list (reserved)

### Version Convention

Types follow a strict versioning convention with a `-vN` suffix. When the
format of a type changes, a new version is created (e.g., `v0` to `v1` to
`v2`). Old versions remain registered for backward compatibility but are marked
as deprecated internally. The `VCurrent` constants (e.g.,
`TypeTomlConfigVCurrent = TypeTomlConfigV2`) always point to the latest version.

### Default Types by Genre

Each genre can have a default type. When creating a new object without
specifying a type, dodder uses the genre's default:

| Genre | Default Type |
|-------|-------------|
| Config | `!toml-config-v2` |
| Tag | `!toml-tag-v1` |
| Type | `!toml-type-v1` |
| Repo | `!toml-repo-uri-v0` |
| Inventory List | `!inventory_list-v2` |

### Custom Types

Define custom types by creating type objects in the repository. Custom types
allow defining new content formats with specific serialization rules.

## Tags

Tags are the primary organizational mechanism in dodder. They provide flexible,
non-hierarchical labeling that replaces the need for directory structures.

### Plain Tags

Standard labels attached to zettels. Plain tags are simple strings:

```
project
important
reading-list
```

Attach tags during creation:

```bash
dodder new -edit=false -tags project,important
```

### Dependent Tags

Tags prefixed with `-` indicate a dependency relationship. A dependent tag
implies that the tagged zettel depends on or is subordinate to something:

```
-blocked
-dependency
-waiting-on
```

Dependent tags carry semantic meaning in queries and organization. They signal
that the zettel has an outstanding relationship or constraint.

### Virtual Tags

Tags prefixed with `%` are computed or system-generated. Virtual tags are not
stored directly but are derived at query time based on object state or rules:

```
%computed
%archived
```

Virtual tags enable dynamic categorization without manual tagging.

### Tag Hierarchy

Tags can be hierarchical, forming a tree structure. Child tags inherit from
parent tags, enabling broad-to-specific categorization. The tag hierarchy is
managed through tag definition objects (`!toml-tag-v1`) that specify parent-child
relationships.

### Lua-Scripted Tags

Tags defined with the `!lua-tag-v2` type contain Lua scripts that execute when
the tag is evaluated. This enables dynamic behavior -- a scripted tag can
compute whether it applies to a zettel based on the zettel's content, other
tags, or external state.

## Content-Addressable Storage

All blob content in dodder is stored by its cryptographic hash. This provides
deduplication, integrity verification, and immutability.

### Hash Algorithm

Dodder uses blake2b-256 as its primary hash algorithm (with sha256 also
supported). The hash format is defined in `echo/markl/format_hash.go`. Blob
identifiers encode the algorithm and digest, e.g., `@blake2b256-abc123...`.

### Storage Layout

Blobs are stored in the filesystem using Git-like bucketing. The first two
hexadecimal characters of the hash serve as a directory name, and the remaining
characters form the filename. This distributes blobs across 256 subdirectories to
avoid filesystem performance degradation from large flat directories.

Example: A blob with hash `a1b2c3d4...` is stored at:

```
.dodder/local/share/blobs/a1/b2c3d4...
```

The `papa/store_fs` package implements this filesystem store.

### Deduplication

Because storage is content-addressed, identical content always produces the same
hash and is stored exactly once. Creating two zettels with identical content
results in both pointing to the same blob. This is automatic and transparent.

### Immutability

Once a blob is written to the store, its content never changes. The hash
guarantees this -- any modification would produce a different hash. When a
zettel's content is edited, a new blob is created with the new content, and the
zettel's metadata is updated to reference the new blob hash. The old blob remains
in the store until garbage collection removes unreferenced blobs.

### Integrity Verification

The hash serves as a built-in integrity check. `dodder fsck` can verify that
every stored blob matches its expected hash, detecting corruption or tampering.

## Versioning

Every change to every object in dodder is recorded in the version history. There
is no "save without tracking" -- all mutations flow through the inventory list
system.

### Inventory Lists

Inventory lists are the authoritative version record. They are text-format,
append-only logs stored in `.dodder/local/share/inventory_lists_log`. Each
inventory list entry records the complete state of an object at a point in time:
its ID, type, tags, blob hash, and TAI timestamp.

Inventory lists are identified by TAI (Temps Atomique International) timestamps,
providing monotonically increasing, globally unique version identifiers.

### Stream Index

The stream index (binary pages in `.dodder/local/share/objects_index/`) provides
fast random access to object state. Unlike inventory lists which must be scanned
sequentially, the stream index supports direct lookup by object ID.

The stream index is derived from inventory lists and can be rebuilt from scratch.
It uses a lazy-loading page system -- pages are read from disk only on first
query, keeping memory usage low for large repositories. `stream_index.MakeIndex`
creates the index, and `dodder reindex` rebuilds it if corruption or staleness
is suspected.

### Store Initialization Order

When dodder opens a repository, initialization follows a strict order:

1. Initialize inventory list store (load the append-only logs).
2. Build working list (aggregate current object states).
3. Create zettel ID index (load the bitset of occupied coordinate pairs).
4. Create stream index (lazy -- pages read on first query).

This order ensures that derived indexes are built from authoritative data.

## Working Copy vs. Store

Dodder maintains a clear separation between the store (the `.dodder/` directory)
and the working copy (files visible on the filesystem).

### The Store

The `.dodder/` directory contains:

- **Blobs**: Content-addressed binary storage in bucketed directories.
- **Inventory lists**: Append-only version history logs.
- **Stream index**: Binary pages for fast object lookup (derived).
- **Configuration**: Repository settings, blob store configs, zettel ID provider
  lists (Yin/Yang).
- **Zettel ID index**: Bitset tracking occupied ID coordinates (derived).

The store is the source of truth. All queries, version history, and content
retrieval read from the store.

### The Working Copy

Files on the filesystem outside `.dodder/` are the working copy. These are
checked-out representations of store objects, formatted for human editing. The
working copy is a convenience -- dodder can operate entirely without it (all
objects can remain internal to the store).

### Checkout States

Every object has a checkout state that describes its relationship between the
store and the working copy. The states are defined in
`echo/checked_out_state`:

| State | Store | Filesystem | Description |
|-------|-------|------------|-------------|
| **Internal** | Present | Absent | Object exists only in the store. Default state for newly created objects when `-edit=false`. |
| **CheckedOut** | Present | Present | Object is synced to both store and filesystem. The store and filesystem versions match. |
| **JustCheckedOut** | Present | Present | UI variant of CheckedOut, indicating a recent checkout operation. |
| **Untracked** | Absent | Present | File exists on the filesystem but is not tracked by the store. |
| **Recognized** | Present | Present | Object matched between store and filesystem but not yet synced. Intermediate state during checkout/checkin. |
| **Conflicted** | Present | Present | Store version and filesystem version differ. Requires manual resolution. |

### Checkout and Checkin

The `checkout` command syncs objects from the store to the filesystem:

```bash
dodder checkout one/uno
```

This writes the zettel's content to a file, formatted according to its type.
The object transitions from Internal to CheckedOut.

The `checkin` command syncs changes from the filesystem back into the store:

```bash
dodder checkin
```

This reads modified files, creates new blobs for changed content, and updates
the inventory list. Unmodified files are skipped.

### Conflict Resolution

When both the store version and the filesystem version of an object have changed
independently (the object is in the Conflicted state), manual resolution is
required. The `mergetool` command provides an interface for resolving conflicts,
similar to Git's merge conflict workflow.

### The Status Command

The `status` command shows the checkout state of all objects:

```bash
dodder status
```

Output lists objects grouped by their checkout state, making it clear which
objects are internal, checked out, untracked, or conflicted.
