---
name: design_patterns-triple_hyphen_io
description: >
  Use when working with the triple-hyphen serialization format, adding coders
  for versioned types, reading or writing typed blobs, or debugging
  serialization issues. Also applies when encountering --- boundary parsing,
  TypedBlob, CoderToTypedBlob, CoderTypeMapWithoutType, or type-dispatched
  decoding.
triggers:
  - triple hyphen
  - triple_hyphen_io
  - TypedBlob
  - CoderToTypedBlob
  - CoderTypeMapWithoutType
  - coder registration
  - boundary parsing
  - typed serialization
---

# Triple-Hyphen IO Format

## Overview

The triple-hyphen IO system is dodder's serialization format for typed,
versioned data. Files use `---` boundaries to separate a type metadata header
from the body content. On read, the type string from the header selects the
correct version-specific decoder. On write, the current version's type string
and encoder produce the output.

## On-Disk Format

```
---
! toml-blob_store_config-v2
---

[actual-content]
compression-type = "zstd"
lock-internal-files = true
```

The metadata section contains a type indicator line prefixed with `!`. The body
follows the second `---` boundary and an empty line.

## Key Types

### TypedBlob

Generic wrapper pairing a type identifier with blob data:

```go
// foxtrot/triple_hyphen_io/coder_type_map.go
type TypedBlob[BLOB any] struct {
    Type ids.TypeStruct
    Blob BLOB
}
```

### CoderToTypedBlob

Composes metadata and blob coders into a complete serializer:

```go
// foxtrot/triple_hyphen_io/coder_to_typed_blob.go
type CoderToTypedBlob[BLOB any] struct {
    RequireMetadata bool
    Metadata        interfaces.CoderBufferedReadWriter[*TypedBlob[BLOB]]
    Blob            CoderTypeMapWithoutType[BLOB]
}
```

### CoderTypeMapWithoutType

Maps type strings to version-specific decoders:

```go
// foxtrot/triple_hyphen_io/coder_type_map.go
type CoderTypeMapWithoutType[BLOB any] map[string]interfaces.CoderBufferedReadWriter[*BLOB]
```

### TypedMetadataCoder

Handles encoding/decoding of the `! type-string` metadata line:

```go
// foxtrot/triple_hyphen_io/coder_metadata.go
type TypedMetadataCoder[BLOB any] struct{}
```

Writes metadata as: `fmt.Fprintf(writer, "! %s\n", typedBlob.Type.StringSansOp())`

## Encoding Sequence

1. Write first boundary: `---\n`
2. Write metadata (type line via `TypedMetadataCoder`)
3. Write second boundary: `---\n`
4. Write empty line: `\n`
5. Write blob content via version-specific encoder

## Decoding Sequence

1. Read metadata section (extracts type string)
2. Look up type string in `CoderTypeMapWithoutType` map
3. Decode blob content using the matched coder
4. Return `TypedBlob[T]` with both type and deserialized data

```go
func (coderTypeMap CoderTypeMapWithoutType[BLOB]) DecodeFrom(
    typedBlob *TypedBlob[BLOB],
    bufferedReader *bufio.Reader,
) (n int64, err error) {
    coder, ok := coderTypeMap[typedBlob.Type.String()]
    if !ok {
        err = errors.ErrorWithStackf("no coders available for type: %q", typedBlob.Type)
        return
    }
    n, err = coder.DecodeFrom(&typedBlob.Blob, bufferedReader)
    return
}
```

## Registering a New Coder

Add an entry to the package's coder map with a `Progenitor` function that
returns a zero-value of the concrete struct:

```go
// golf/blob_store_configs/coding.go
var Coder = triple_hyphen_io.CoderToTypedBlob[Config]{
    Blob: triple_hyphen_io.CoderTypeMapWithoutType[Config](
        map[string]interfaces.CoderBufferedReadWriter[*Config]{
            ids.TypeTomlBlobStoreConfigV0: triple_hyphen_io.CoderToml[Config, *Config]{
                Progenitor: func() Config { return &TomlV0{} },
            },
            // ... more versions
        },
    ),
}
```

## Packages Using This Format

| Package | Content |
|---------|---------|
| `golf/blob_store_configs` | Storage backend configurations |
| `golf/repo_configs` | Repository configurations |
| `golf/repo_blobs` | Repository blob definitions |
| `hotel/workspace_config_blobs` | Workspace configurations |
| `hotel/genesis_configs` | Genesis configurations |
| `lima/inventory_list_coders` | Inventory list entries |

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Forgetting to add coder entry for new version | Every registered type string needs a coder map entry |
| Wrong `Progenitor` return type | Must return the concrete version struct, not the interface |
| Missing boundary in hand-crafted test data | Both `---` boundaries and the empty line after the second are required |
