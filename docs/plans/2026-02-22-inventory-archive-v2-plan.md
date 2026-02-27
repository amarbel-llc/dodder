# Inventory Archive V2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a V2 inventory archive config that embeds its own hash-bucketed loose store, removing the external loose store reference.

**Architecture:** `TomlInventoryArchiveV2` implements `ConfigInventoryArchiveDelta` with a zero `LooseBlobStoreId`. The factory detects the empty ID and constructs an embedded `localHashBucketed` at `<basePath>/loose/`. Reuses `inventoryArchiveV1` store struct unchanged.

**Tech Stack:** Go, TOML config via triple-hyphen IO, existing blob store infrastructure.

---

### Task 1: Register the V2 type constant

**Files:**
- Modify: `go/internal/echo/ids/types_builtin.go:32-34` (constants) and `:97-101` (init registration)

**Step 1: Add the V2 constant and update VCurrent**

In the constants block, add `TypeTomlBlobStoreConfigInventoryArchiveV2` and
update `VCurrent`:

```go
TypeTomlBlobStoreConfigInventoryArchiveV0       = "!toml-blob_store_config-inventory_archive-v0"
TypeTomlBlobStoreConfigInventoryArchiveV1       = "!toml-blob_store_config-inventory_archive-v1"
TypeTomlBlobStoreConfigInventoryArchiveV2       = "!toml-blob_store_config-inventory_archive-v2"
TypeTomlBlobStoreConfigInventoryArchiveVCurrent = TypeTomlBlobStoreConfigInventoryArchiveV2
```

**Step 2: Register in init()**

After the V1 registration block, add:

```go
registerBuiltinTypeString(
    TypeTomlBlobStoreConfigInventoryArchiveV2,
    genres.Unknown,
    false,
)
```

**Step 3: Build to verify**

Run: `go build ./src/echo/ids/...`
Expected: exit 0

**Step 4: Commit**

```
feat: register inventory archive V2 type constant
```

---

### Task 2: Create the V2 config struct

**Files:**
- Create: `go/internal/golf/blob_store_configs/toml_inventory_archive_v2.go`

**Step 1: Write the config struct**

Model after `toml_inventory_archive_v1.go` but without `LooseBlobStoreId`.
Implements `ConfigInventoryArchiveDelta` (which embeds `ConfigInventoryArchive`).
`GetLooseBlobStoreId()` returns zero value.

```go
package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

type TomlInventoryArchiveV2 struct {
	HashTypeId      string                           `toml:"hash_type-id"`
	CompressionType compression_type.CompressionType `toml:"compression-type"`
	Encryption      markl.Id                         `toml:"encryption"`
	Delta           DeltaConfig                      `toml:"delta"`
}

var (
	_ ConfigInventoryArchiveDelta = TomlInventoryArchiveV2{}
	_ ConfigMutable               = &TomlInventoryArchiveV2{}
	_                             = registerToml[TomlInventoryArchiveV2](
		Coder.Blob,
		ids.TypeTomlBlobStoreConfigInventoryArchiveV2,
	)
)

func (TomlInventoryArchiveV2) GetBlobStoreType() string {
	return "local-inventory-archive"
}

func (config *TomlInventoryArchiveV2) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	config.CompressionType.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&config.HashTypeId,
		"hash_type-id",
		markl.FormatIdHashBlake2b256,
		"hash type for archive checksums and blob hashes",
	)
}

func (config TomlInventoryArchiveV2) getBasePath() string {
	return ""
}

func (config TomlInventoryArchiveV2) SupportsMultiHash() bool {
	return false
}

func (config TomlInventoryArchiveV2) GetDefaultHashTypeId() string {
	return config.HashTypeId
}

func (config TomlInventoryArchiveV2) GetBlobCompression() interfaces.IOWrapper {
	return &config.CompressionType
}

func (config TomlInventoryArchiveV2) GetBlobEncryption() domain_interfaces.MarklId {
	return config.Encryption
}

func (config TomlInventoryArchiveV2) GetLooseBlobStoreId() blob_store_id.Id {
	var zero blob_store_id.Id
	return zero
}

func (config TomlInventoryArchiveV2) GetCompressionType() compression_type.CompressionType {
	return config.CompressionType
}

func (config TomlInventoryArchiveV2) GetDeltaEnabled() bool {
	return config.Delta.Enabled
}

func (config TomlInventoryArchiveV2) GetDeltaAlgorithm() string {
	return config.Delta.Algorithm
}

func (config TomlInventoryArchiveV2) GetDeltaMinBlobSize() uint64 {
	return config.Delta.MinBlobSize
}

func (config TomlInventoryArchiveV2) GetDeltaMaxBlobSize() uint64 {
	return config.Delta.MaxBlobSize
}

func (config TomlInventoryArchiveV2) GetDeltaSizeRatio() float64 {
	return config.Delta.SizeRatio
}
```

**Step 2: Build to verify**

Run: `go build ./src/golf/blob_store_configs/...`
Expected: exit 0

**Step 3: Commit**

```
feat: add TomlInventoryArchiveV2 config struct with no external loose store
```

---

### Task 3: Add V1 upgrade path to V2

**Files:**
- Modify: `go/internal/golf/blob_store_configs/toml_inventory_archive_v1.go`

**Step 1: Add Upgrade() method to V1**

V1 currently does not have an `Upgrade()` method (V0 upgrades to V1). Add one
that upgrades V1 to V2 by dropping `LooseBlobStoreId`:

```go
func (config TomlInventoryArchiveV1) Upgrade() (Config, ids.TypeStruct) {
	upgraded := &TomlInventoryArchiveV2{
		HashTypeId:      config.HashTypeId,
		CompressionType: config.CompressionType,
		Delta:           config.Delta,
	}

	upgraded.Encryption.ResetWithMarklId(config.Encryption)

	return upgraded, ids.GetOrPanic(
		ids.TypeTomlBlobStoreConfigInventoryArchiveV2,
	).TypeStruct
}
```

Also add `ConfigUpgradeable` to the var block:

```go
var (
	_ ConfigInventoryArchiveDelta = TomlInventoryArchiveV1{}
	_ ConfigUpgradeable           = TomlInventoryArchiveV1{}
	_ ConfigMutable               = &TomlInventoryArchiveV1{}
	// ...
)
```

**Step 2: Build to verify**

Run: `go build ./src/golf/blob_store_configs/...`
Expected: exit 0

**Step 3: Commit**

```
feat: add V1 -> V2 upgrade path for inventory archive config
```

---

### Task 4: Update factory to construct embedded loose store

**Files:**
- Modify: `go/internal/india/blob_stores/main.go:225-248` (the `ConfigInventoryArchiveDelta` case)

**Step 1: Add embedded store construction**

In the `ConfigInventoryArchiveDelta` case, check if the loose store ID is empty.
If so, construct an embedded `localHashBucketed` at `basePath + "/loose"` using
the archive's own config values. The embedded store needs to satisfy
`ConfigLocalHashBucketed`, so use the existing `DefaultType` (aliased to
`TomlV2`) with values from the archive config:

```go
case blob_store_configs.ConfigInventoryArchiveDelta:
    var looseBlobStore domain_interfaces.BlobStore

    if config.GetLooseBlobStoreId().IsEmpty() {
        loosePath := filepath.Join(configNamed.Path.GetBase(), "loose")

        embeddedConfig := &blob_store_configs.DefaultType{
            HashTypeId:        config.GetDefaultHashTypeId(),
            HashBuckets:       blob_store_configs.DefaultHashBuckets,
            CompressionType:   config.GetCompressionType(),
            LockInternalFiles: true,
        }

        if looseBlobStore, err = makeLocalHashBucketed(
            envDir,
            loosePath,
            embeddedConfig,
        ); err != nil {
            return store, err
        }
    } else if blobStores != nil {
        looseBlobStoreId := config.GetLooseBlobStoreId().String()
        if initialized, ok := blobStores[looseBlobStoreId]; ok {
            looseBlobStore = initialized.BlobStore
        }
    }

    if looseBlobStore == nil {
        err = errors.BadRequestf(
            "inventory archive store requires loose-blob-store-id %q but it was not found",
            config.GetLooseBlobStoreId(),
        )
        return store, err
    }

    return makeInventoryArchiveV1(
        envDir,
        configNamed.Path.GetBase(),
        config,
        looseBlobStore,
    )
```

Add `"path/filepath"` to imports if not already present.

**Step 2: Build to verify**

Run: `go build ./src/india/blob_stores/...`
Expected: exit 0

**Step 3: Commit**

```
feat: construct embedded hash-bucketed store for inventory archive V2
```

---

### Task 5: Add init command

**Files:**
- Modify: `go/internal/lima/commands_madder/init.go:44-64`

**Step 1: Add init-inventory-archive-embedded command**

After the existing `init-inventory-archive` registration:

```go
utility.AddCmd("init-inventory-archive-embedded", &Init{
    tipe: ids.GetOrPanic(
        ids.TypeTomlBlobStoreConfigInventoryArchiveV2,
    ).TypeStruct,
    blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV2{
        Delta: blob_store_configs.DeltaConfig{
            Enabled:     false,
            Algorithm:   "bsdiff",
            MinBlobSize: 256,
            MaxBlobSize: 10485760,
            SizeRatio:   2.0,
        },
    },
})
```

**Step 2: Update init-inventory-archive to use VCurrent (now V2)**

The existing `init-inventory-archive` uses `TypeTomlBlobStoreConfigInventoryArchiveVCurrent`,
which now points to V2. Update it to use `TomlInventoryArchiveV2` instead of
`TomlInventoryArchiveV1` so the config struct matches the type:

```go
utility.AddCmd("init-inventory-archive", &Init{
    tipe: ids.GetOrPanic(
        ids.TypeTomlBlobStoreConfigInventoryArchiveVCurrent,
    ).TypeStruct,
    blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV2{
        Delta: blob_store_configs.DeltaConfig{
            Enabled:     false,
            Algorithm:   "bsdiff",
            MinBlobSize: 256,
            MaxBlobSize: 10485760,
            SizeRatio:   2.0,
        },
    },
})
```

Rename the old V1 registration to `init-inventory-archive-v1`:

```go
utility.AddCmd("init-inventory-archive-v1", &Init{
    tipe: ids.GetOrPanic(
        ids.TypeTomlBlobStoreConfigInventoryArchiveV1,
    ).TypeStruct,
    blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV1{
        Delta: blob_store_configs.DeltaConfig{
            Enabled:     false,
            Algorithm:   "bsdiff",
            MinBlobSize: 256,
            MaxBlobSize: 10485760,
            SizeRatio:   2.0,
        },
    },
})
```

**Step 3: Build full madder binary to verify**

Run: `go build ./cmd/madder/`
Expected: exit 0

**Step 4: Commit**

```
feat: add madder init commands for inventory archive V2 (embedded store)
```

---

### Task 6: Build and smoke test

**Step 1: Full build**

Run: `just build`

**Step 2: Run unit tests**

Run: `just test-go`

**Step 3: Commit if any fixups needed**
