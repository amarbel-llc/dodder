---
name: design_patterns-horizontal_versioning
description: >
  Use when adding a new version of a persistent data structure, creating a new
  blob or config type, implementing an upgrade path between versions, or
  registering a coder for a versioned type. Also applies when working with
  type_blobs, blob_store_configs, blob_stores, triple_hyphen_io, or any
  package following the *_blobs / *_configs / *_stores naming convention.
triggers:
  - add a new version
  - add a blob type
  - add a config type
  - upgrade path
  - version migration
  - register a coder
  - type_blobs
  - blob_store_configs
  - triple_hyphen_io
  - horizontal versioning
---

# Horizontal Versioning

## Overview

Dodder's persistent data structures evolve through horizontal versioning: new
versions are added alongside old ones, never replacing them. Every version is a
separate concrete struct that implements a shared interface. Old versions remain
as valid deserialization targets so that legacy data can always be read and then
upgraded forward.

## Core Principles

1. **One interface, many versions.** All versions of a data structure implement
   the same interface. Consumers depend only on the interface.

2. **Concrete implementations are separate.** Each version is its own struct in
   its own file (e.g., `toml_v0.go`, `toml_v1.go`). Versions must not embed or
   compose each other. They may share imported utility types (e.g.,
   `compression_type.CompressionType`) but each struct stands alone.

3. **No redundant encoding of config data.** A concrete implementation encodes
   exactly the fields for its version. It does not carry compatibility shims,
   optional fields for other versions, or union-style "one of these is set"
   patterns. Each version can assume that its data is exclusively its own.

4. **Upgrade paths transform data between versions.** When data needs to move
   from one version to another, the `Upgrade()` method on the source version
   constructs the target version's struct, mapping and transforming fields as
   needed. This is the only mechanism for cross-version data flow.

5. **Serialization reflects the concrete version.** After an upgrade, the data
   is serialized using the new version's type string and coder. There is no
   concept of a "compatible" or "polymorphic" on-disk format.

## Architecture

### Package Naming Convention

| Suffix | Purpose | Examples |
|--------|---------|---------|
| `*_blobs` | Data schemas (metadata, content structure) | `type_blobs`, `tag_blobs`, `repo_blobs`, `workspace_config_blobs` |
| `*_configs` | Backend/infrastructure configuration | `blob_store_configs`, `repo_configs`, `genesis_configs` |
| `*_stores` | Runtime implementations consuming configs | `blob_stores` |

### Four Components of a Versioned Type

#### 1. Stable Interface

Defined once per data structure. All versions implement it.

```go
// go/src/lima/type_blobs/main.go
type Blob interface {
    GetFileExtension() string
    GetBinary() bool
    GetMimeType() string
    GetVimSyntaxType() string
    WithFormatters
    WithFormatterUTIGroups
    WithStringLuaHooks
}
```

#### 2. Versioned Concrete Structs

Each version is a separate struct with its own fields. No embedding between
versions. Shared utility types are fine; shared struct definitions are not.

```go
// go/src/golf/blob_store_configs/toml_v0.go
type TomlV0 struct {
    BasePath          string                           `toml:"base-path,omitempty"`
    AgeEncryption     markl_age_id.Id                  `toml:"age-encryption,omitempty"`
    CompressionType   compression_type.CompressionType `toml:"compression-type"`
    LockInternalFiles bool                             `toml:"lock-internal-files"`
}

// go/src/golf/blob_store_configs/toml_v2.go
type TomlV2 struct {
    HashBuckets       values.IntSlice                  `toml:"hash_buckets"`
    BasePath          string                           `toml:"base_path,omitempty"`
    HashTypeId        string                           `toml:"hash_type-id"`
    Encryption        markl.Id                         `toml:"encryption"`
    CompressionType   compression_type.CompressionType `toml:"compression-type"`
    LockInternalFiles bool                             `toml:"lock-internal-files"`
}
```

Note: both use `compression_type.CompressionType` (shared utility type), but
neither embeds the other. V2 has `HashBuckets` and `HashTypeId` that V0 lacks.
V0 has `AgeEncryption` which V2 replaced with the more general `Encryption`.

#### 3. Type String Registration

Each version gets a unique type string registered in `echo/ids/types_builtin.go`.
A `VCurrent` alias tracks the latest version.

```go
// go/src/echo/ids/types_builtin.go
const (
    TypeTomlBlobStoreConfigV0       = "!toml-blob_store_config-v0"
    TypeTomlBlobStoreConfigV1       = "!toml-blob_store_config-v1"
    TypeTomlBlobStoreConfigV2       = "!toml-blob_store_config-v2"
    TypeTomlBlobStoreConfigVCurrent = TypeTomlBlobStoreConfigV2
)

func init() {
    registerBuiltinTypeString(TypeTomlBlobStoreConfigV0, genres.Unknown, false)
    registerBuiltinTypeString(TypeTomlBlobStoreConfigV1, genres.Unknown, false)
    registerBuiltinTypeString(TypeTomlBlobStoreConfigV2, genres.Unknown, false)
}
```

#### 4. Coder Registration

A coder map associates each type string with a deserializer that produces the
correct concrete struct behind the shared interface.

```go
// go/src/golf/blob_store_configs/coding.go
var Coder = triple_hyphen_io.CoderToTypedBlob[Config]{
    Blob: triple_hyphen_io.CoderTypeMapWithoutType[Config](
        map[string]interfaces.CoderBufferedReadWriter[*Config]{
            ids.TypeTomlBlobStoreConfigV0: triple_hyphen_io.CoderToml[Config, *Config]{
                Progenitor: func() Config { return &TomlV0{} },
            },
            ids.TypeTomlBlobStoreConfigV1: triple_hyphen_io.CoderToml[Config, *Config]{
                Progenitor: func() Config { return &TomlV1{} },
            },
            ids.TypeTomlBlobStoreConfigV2: triple_hyphen_io.CoderToml[Config, *Config]{
                Progenitor: func() Config { return &TomlV2{} },
            },
        },
    ),
}
```

### Type-Dispatched Serialization

The `triple_hyphen_io` package provides the generic serialization layer. Files
on disk use `---` delimiters to separate a type header from the body. On read,
the type string from the header selects the correct coder:

```go
// go/src/foxtrot/triple_hyphen_io/coder_type_map.go
type TypedBlob[BLOB any] struct {
    Type ids.TypeStruct
    Blob BLOB
}

type CoderTypeMapWithoutType[BLOB any] map[string]interfaces.CoderBufferedReadWriter[*BLOB]

func (coderTypeMap CoderTypeMapWithoutType[BLOB]) DecodeFrom(
    typedBlob *TypedBlob[BLOB],
    bufferedReader *bufio.Reader,
) (n int64, err error) {
    tipe := typedBlob.Type
    coder, ok := coderTypeMap[tipe.String()]
    if !ok {
        err = errors.ErrorWithStackf("no coders available for type: %q", tipe)
        return
    }
    n, err = coder.DecodeFrom(&typedBlob.Blob, bufferedReader)
    return
}
```

### Upgrade Paths

Non-current versions implement `ConfigUpgradeable` to provide forward migration.
The `Upgrade()` method constructs the next version's struct directly, mapping
and transforming fields:

```go
// go/src/golf/blob_store_configs/main.go
type ConfigUpgradeable interface {
    Config
    Upgrade() (Config, ids.TypeStruct)
}

// go/src/golf/blob_store_configs/toml_v1.go
func (blobStoreConfig TomlV1) Upgrade() (Config, ids.TypeStruct) {
    upgraded := &TomlV2{
        HashBuckets:       blobStoreConfig.HashBuckets,
        BasePath:          blobStoreConfig.BasePath,
        HashTypeId:        markl.FormatIdHashSha256,
        CompressionType:   blobStoreConfig.CompressionType,
        LockInternalFiles: blobStoreConfig.LockInternalFiles,
    }
    upgraded.Encryption.ResetWithMarklId(blobStoreConfig.Encryption)
    return upgraded, ids.GetOrPanic(ids.TypeTomlBlobStoreConfigV2).TypeStruct
}
```

Callers run a loop that upgrades iteratively until reaching a version that does
not implement `ConfigUpgradeable` (the current version):

```go
for {
    configUpgraded, ok := typedConfig.Blob.(blob_store_configs.ConfigUpgradeable)
    if !ok {
        break
    }
    typedConfig.Blob, typedConfig.Type = configUpgraded.Upgrade()
}
```

After this loop, the data is at the current version and will be serialized with
the current version's type string.

## Adding a New Version

1. **Create the struct** in a new file (e.g., `toml_v3.go`). Define only the
   fields this version needs. Do not carry forward deprecated fields or
   compatibility shims.

2. **Implement the interface** that all versions of this type share.

3. **Register the type string** in `echo/ids/types_builtin.go`. Add a new
   constant and update `VCurrent` to point to the new version.

4. **Add an `Upgrade()` method** to the previous version that constructs the
   new version, mapping and transforming fields. The previous version now
   implements `ConfigUpgradeable`.

5. **Register the coder** in the package's `coding.go`, adding a new entry to
   the type-to-coder map with a `Progenitor` that returns a zero-value of the
   new struct.

6. **Regenerate test fixtures** with `just test-bats-update-fixtures` if the
   change affects stored data formats. Review and commit the updated fixtures.

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Embedding one version struct inside another | Each version is independent. Share imported utility types, not struct composition. |
| Adding optional fields to an existing version for new features | Create a new version. Existing versions are frozen. |
| Carrying forward deprecated fields "for compatibility" | New versions define only their own fields. The upgrade path handles transformation. |
| Forgetting to update `VCurrent` | Always point `VCurrent` at the newest version after adding it. |
| Modifying `Upgrade()` on the current version | Only non-current versions implement `ConfigUpgradeable`. The current version is the upgrade chain's terminus. |

## Migration Status

The codebase has three Coder architectures at different stages of migration
toward the canonical `CoderToTypedBlob` pattern. This section tracks each
versioned type family, its current architecture, and what remains to reach full
compliance with the pattern described above.

### Target Architecture

All versioned types should use **Architecture A: `CoderToTypedBlob`** with
`CoderTypeMapWithoutType` maps and `Progenitor` functions. All non-current
versions should implement `Upgrade()` to form a complete chain from the oldest
version to `VCurrent`.

### Architecture Summary

| Architecture | Description | Target? |
|---|---|---|
| **A: `CoderToTypedBlob`** | Declarative type-string→progenitor map in `coding.go`. Uses `triple_hyphen_io.CoderToml` with `Progenitor` functions. | Yes |
| **B: `TypedStore` + switch** | Hand-written struct with one `domain_interfaces.TypedStore` field per version. Dispatch via `switch` on type string. | No — migrate to A |
| **C: Custom constructor map** | Package-local `coderConstructors` map producing custom `coder` structs. `Closet` bridges to `CoderTypeMapWithoutType` at runtime. | No — migrate to A |

### Per-Type Family Status

#### `blob_store_configs` (Architecture A) — `golf/blob_store_configs/coding.go`

| Version | Type String | Struct | Coder | Upgrade |
|---|---|---|---|---|
| V0 | `!toml-blob_store_config-v0` | `TomlV0` | registered | **missing** — no `Upgrade()` to V1 |
| V1 | `!toml-blob_store_config-v1` | `TomlV1` | registered | V1→V2 implemented |
| V2 (current) | `!toml-blob_store_config-v2` | `TomlV2` | registered | terminus |
| SFTP Explicit V0 | `!toml-blob_store_config_sftp-explicit-v0` | `TomlSFTPV0` | registered | N/A (single version) |
| SFTP SSH Config V0 | `!toml-blob_store_config_sftp-ssh_config-v0` | `TomlSFTPViaSSHConfigV0` | registered | N/A (single version) |
| Pointer V0 | `!toml-blob_store_config-pointer-v0` | **no struct** | **no coder** | orphan type string |
| Inventory Archive V0 | `!toml-blob_store_config-inventory_archive-v0` | **no struct** | **no coder** | orphan type string |

Remaining work:
- [ ] Add `TomlV0.Upgrade()` → V1
- [ ] Remove or implement `TypeTomlBlobStoreConfigPointerV0` (orphan registration)
- [ ] Remove or implement `TypeTomlBlobStoreConfigInventoryArchiveV0` (orphan registration)
- [ ] Adopt `registerToml` helper (defined in `coding.go` but unused — dead code)

#### `repo_configs` (Architecture A) — `golf/repo_configs/coding.go`

| Version | Type String | Struct | Coder | Upgrade |
|---|---|---|---|---|
| V0 (deprecated) | `!toml-config-v0` | `V0` | registered | **missing** — no `Upgrade()` to V1 |
| V1 | `!toml-config-v1` | `V1` | registered | **missing** — no `Upgrade()` to V2 |
| V2 (current) | `!toml-config-v2` | `V2` | registered | terminus |

Remaining work:
- [ ] Add `V0.Upgrade()` → V1
- [ ] Add `V1.Upgrade()` → V2
- [ ] Define `ConfigUpgradeable` interface (or reuse from `blob_store_configs`)

#### `genesis_configs` (Architecture A) — `hotel/genesis_configs/coder.go`

| Version | Type String | Struct (Private) | Struct (Public) | Coder | Upgrade |
|---|---|---|---|---|---|
| V1 | `!toml-config-immutable-v1` | `TomlV1Private` | `TomlV1Public` | registered (both coders) | **missing** — no `Upgrade()` to V2 |
| V2 (current) | `!toml-config-immutable-v2` | `TomlV2Private` | `TomlV2Public` | registered (both coders) | terminus |

Remaining work:
- [ ] Add `TomlV1Private.Upgrade()` → V2
- [ ] Add `TomlV1Public.Upgrade()` → V2

#### `repo_blobs` (Architecture A) — `golf/repo_blobs/coding.go`

| Version | Type String | Struct | Coder | Upgrade |
|---|---|---|---|---|
| Local Override V0 | `!toml-repo-local_override_path-v0` | `TomlLocalOverridePathV0` | registered | N/A |
| XDG V0 | `!toml-repo-dotenv_xdg-v0` | `TomlXDGV0` | registered | N/A |
| URI V0 (default) | `!toml-repo-uri-v0` | `TomlUriV0` | registered | N/A |

No version progression — all at V0. No work needed unless new versions are added.

#### `workspace_config_blobs` (Architecture A) — `hotel/workspace_config_blobs/io.go`

| Version | Type String | Struct | Coder | Upgrade |
|---|---|---|---|---|
| V0 (current) | `!toml-workspace_config-v0` | `V0` | registered | terminus |

Single version. No work needed unless a new version is added.

#### `type_blobs` (Architecture B) — `lima/type_blobs/coder.go`

| Version | Type String | Struct | Coder | Upgrade |
|---|---|---|---|---|
| V0 (deprecated) | `!toml-type-v0` | `TomlV0` | `CoderToTypedBlob` map | **missing** — no `Upgrade()` to V1 |
| V1 (current) | `!toml-type-v1` | `TomlV1` | `CoderToTypedBlob` map | terminus |

Remaining work:
- [x] Migrate to `CoderToTypedBlob` pattern (replace struct + switch with `coding.go` map)
- [ ] Add `TomlV0.Upgrade()` → V1

#### `tag_blobs` (Architecture B) — `mike/typed_blob_store/tag.go`

| Version | Type String | Struct | Coder | Upgrade |
|---|---|---|---|---|
| V0 | `!toml-tag-v0` | `V0` | `TypedStore` field + switch | **missing** |
| Toml V1 | `!toml-tag-v1` | `TomlV1` | `TypedStore` field + switch | **missing** |
| Lua V1 | `!lua-tag-v1` | `LuaV1` | `TypedStore` field + switch | **missing** |
| Lua V2 (current) | `!lua-tag-v2` | `LuaV2` | `TypedStore` field + switch | terminus |

Remaining work:
- [ ] Migrate to `CoderToTypedBlob` pattern
- [ ] Decide upgrade chain across format boundaries (Toml→Lua or separate chains)
- [ ] Rename `V0` to `TomlV0` for naming consistency with other versions

#### `inventory_list_coders` (Architecture C) — `lima/inventory_list_coders/main.go`

| Version | Type String | Coder | Upgrade |
|---|---|---|---|
| V0 (deprecated) | `!inventory_list-v0` | **missing** — registered in `types_builtin.go` but no entry in `coderConstructors` | N/A |
| V1 | `!inventory_list-v1` | custom `coder` via `coderConstructors` | **missing** — no `Upgrade()` to V2 |
| V2 (current) | `!inventory_list-v2` | custom `coder` via `coderConstructors` | terminus |
| JSON V0 | `!inventory_list-json-v0` | custom `coder` via `coderConstructors` | N/A (alternate format) |

Remaining work:
- [ ] Remove or implement coder for `TypeInventoryListV0` (orphan type string)
- [ ] Migrate to `CoderToTypedBlob` pattern (partially bridged via `Closet`)
- [ ] Add V1 upgrade to V2

#### Orphan Type Strings

Type strings registered in `echo/ids/types_builtin.go` with no struct or coder:

| Type String | Constant | Status |
|---|---|---|
| `!toml-blob_store_config-pointer-v0` | `TypeTomlBlobStoreConfigPointerV0` | No struct, no coder |
| `!toml-blob_store_config-inventory_archive-v0` | `TypeTomlBlobStoreConfigInventoryArchiveV0` | No struct, no coder |
| `!inventory_list-v0` | `TypeInventoryListV0` | Deprecated, no coder |
| `!zettel_id_list-v0` | `TypeZettelIdListV0` | Marked "not used yet", no coder |
