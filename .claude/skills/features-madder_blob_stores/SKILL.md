---
name: features-madder_blob_stores
description: Use when working on the dodder/madder blob-store internals --- blob store ID parsing, the two-phase `MakeBlobStores()` discovery, config interfaces and the `ConfigKeyValues` key-value system, `info-repo` key resolution, archive vs loose blob initialization, or writing BATS tests that create / share / query blob stores.
---

# Blob Store Internals

User-facing CLI behavior (commands, ID prefixes, encryption flag values)
lives in the plugin copy of this skill. This file covers the
implementation: source locations, init phases, config interfaces, and the
BATS testing patterns the dodder/madder bats suite uses.

## Blob Store ID Parsing

A blob store ID has an optional location prefix followed by a name. The
prefix determines where on the filesystem the store lives.

| Prefix | Location | Filesystem root | Example |
|--------|----------|-----------------|---------|
| *(none)* | XDG user | `$XDG_DATA_HOME/madder/blob_stores/` | `shared` |
| `.` | CWD-scoped | `$PWD/.madder/local/share/blob_stores/` | `.default` |
| `/` | XDG system | system data dir | `/system` |
| `~` | *(compat)* | same as unprefixed | `~default` |

**Key behavior:** Unprefixed IDs default to `LocationTypeXDGUser`. The `.`
prefix creates CWD-scoped stores that live inside the repo directory. Two
IDs with the same name but different prefixes (e.g., `default` vs
`.default`) are **different stores** at different filesystem paths.

**Source:** `go/internal/bravo/blob_store_id/` --- `Id` struct with
`location` + `id` fields, `Set()` parses prefix, `String()` omits prefix
for XDG user.

## CWD-Override and User-Scoped Stores Together

When `-repo_id .` is active (which remaps XDG dirs under `.dodder/` in
CWD), `MakeBlobStores()` performs a two-phase discovery:

1. Load CWD-scoped blob store configs from the override directory.
2. If in override mode, **also** load user-scoped stores via
   `CloneWithoutOverride()` (reads real XDG env vars).
3. Merge both maps --- user-scoped stores are accessible alongside
   CWD-scoped ones.

This is how cross-repo blob sharing works: a user-scoped store (e.g.,
`shared`) is visible to any repo that shares the same XDG environment,
regardless of whether those repos use `-repo_id .`.

**Source:** `go/internal/india/blob_stores/main.go:75-95`.

## Blob Store Types

| Type string | Config struct | Description |
|-------------|---------------|-------------|
| `local` | `TomlV0`, `TomlV2` | Hash-bucketed local filesystem storage |
| `local-inventory-archive` | `TomlInventoryArchiveV0/V1/V2` | Packed archive format with optional delta compression |
| *(sftp)* | `TomlSFTPV0`, `TomlSFTPViaSSHConfigV0` | Remote SFTP storage |
| *(pointer)* | `TomlPointerV0` | Indirection to another store's config |

### Local Hash-Bucketed (default)

The default store type. Blobs stored as individual files in hash-bucketed
directories (e.g., `blake2b256/3k/j7xgch6...`). Configurable hash buckets
(default `[2]` = 2-char directory prefix), compression (default zstd),
encryption, and file locking.

### Inventory Archive

Packs loose blobs into archive files with an index for fast lookup.
Requires a loose blob store for unpacked blobs --- either embedded
(created automatically under `blobs/` subdirectory) or referenced via
`loose-blob-store-id`. Supports delta compression (bsdiff algorithm),
configurable max pack size, and independent encryption settings.

**Two-pass initialization:** `MakeBlobStores()` initializes non-archive
stores first, then archives, because archives may reference other stores
via `loose-blob-store-id`.

## Config Key-Value System

`blob_store_configs.ConfigKeyValues(config)` returns a `map[string]string`
of queryable keys. Available keys depend on which interfaces the config
implements:

- **All stores:** `blob-store-type`
- **ConfigHashType:** `hash_type-id`, `supports-multi-hash`
- **BlobIOWrapper:** `encryption`, `compression-type`
- **ConfigLocalHashBucketed:** `hash_buckets`, `lock-internal-files`
- **ConfigInventoryArchive:** `loose-blob-store-id`, `max-pack-size`
- **DeltaConfigImmutable:** `delta.enabled`, `delta.algorithm`,
  `delta.min-blob-size`, `delta.max-blob-size`, `delta.size-ratio`
- **SFTP configs:** `host`, `port`, `user`, `private-key-path`,
  `remote-path`

`ConfigKeyNames(config)` returns sorted key names. Both are used by
`info-repo` for dynamic key lookup.

**Source:** `go/internal/golf/blob_store_configs/key_values.go`.

## info-repo Key Resolution

`info-repo` handles keys in two layers:

1. **Special cases** (hardcoded in switch): `config-immutable`,
   `config-path`, `dir-blob_stores`, `xdg`.
2. **Dynamic lookup** (via `ConfigKeyValues`): All other keys are looked
   up in the config's key-value map. Unknown keys produce an error
   listing available keys.

With 0 args: defaults to `config-immutable` on the default store. With 1
arg: key on default store. With 2 args: store ID + key. With 3+ args:
store ID + multiple keys.

## Encryption Flag Plumbing

The `-encryption` flag (shared via `setEncryptionFlagDefinition`) accepts
`none`, `generate` (or empty), a file path, or a key string. Used by
`blob_store-init`, `blob_store-init-inventory-archive`, and repo `init`.

## BATS Testing Patterns

### Creating a User-Scoped Shared Store

To share blobs across repos in tests (e.g., for import tests):

```bash
setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"
  export output

  set_xdg "$BATS_TEST_TMPDIR"

  # Create user-scoped store (unprefixed = XDG user location)
  run_dodder blob_store-init shared
  assert_success

  # Init repo using the shared store
  run_dodder init \
    -yin <(cat_yin) -yang <(cat_yang) \
    -lock-internal-files=false \
    -repo_id . \
    -encryption none \
    -blob_store-id shared \
    test
  assert_success
}
```

`set_xdg` ensures both the outer and inner repos share the same XDG
namespace. The unprefixed `shared` ID resolves via XDG env vars, making
the store visible to any repo initialized with the same XDG environment.

### Querying Store Config in Tests

```bash
run_dodder blob_store-info-repo compression-type
assert_output 'zstd'

# Query a specific store
run_dodder blob_store-info-repo .archive encryption
assert_output --regexp '.+'
```

### Creating and Packing an Archive Store

```bash
run_dodder blob_store-init-inventory-archive -encryption generate .archive
run_dodder blob_store-write .archive <(echo content)
run_dodder blob_store-pack .archive
```

## Key Source Locations

| Concern | Package |
|---------|---------|
| Blob store ID parsing | `go/internal/bravo/blob_store_id/` |
| Config interfaces & key-value system | `go/internal/golf/blob_store_configs/` |
| Store factory & initialization | `go/internal/india/blob_stores/` |
| Directory layout & paths | `go/internal/echo/directory_layout/` |
| CLI commands | `go/internal/lima/commands_madder/` |
| Command components (flags, env) | `go/internal/kilo/command_components_madder/` |
| BATS tests | `zz-tests_bats/blob_store_*.bats`, `zz-tests_bats/info_repo.bats` |
