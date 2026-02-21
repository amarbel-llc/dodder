---
name: design_pattern-horizontal-versioning
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
