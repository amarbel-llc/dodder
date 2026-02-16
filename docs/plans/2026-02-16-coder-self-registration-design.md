# Coder Self-Registration Design

## Problem

Adding a new versioned blob type requires changes in multiple disconnected locations:

1. A type constant in `echo/ids/types_builtin.go`
2. A `registerBuiltinTypeString()` call in the same file's `init()`
3. A coder map entry in the `*_blobs` package's `coding.go`

Steps 2 and 3 are easy to forget, and the compiler won't catch it. Missing coder
registrations only surface at runtime as `"no coders available for type"` errors.

## Goals

- Each versioned struct file self-registers its coder so there is no central map
  literal to keep in sync.
- A static analyzer verifies that every builtin type ID with a known genre has a
  corresponding coder registration.
- The `echo/ids` type registry stays unchanged (avoids init-order hazards).

## Non-Goals

- Replacing the `ids` builtin type constants or their `init()` registration.
- Generating code with `go:generate` (pure Go approach preferred).
- Changing the `lima/inventory_list_coders` package (different pattern, uses
  constructor factories with `envRepo`).

## Design

### Part 1: Coder Self-Registration

Each `*_blobs` / `*_configs` package that currently uses a `CoderTypeMapWithoutType`
map literal replaces that literal with:

1. A package-level `coderMap` initialized as an empty map.
2. A generic `register` function that adds a `CoderToml` entry to `coderMap`.
3. `var _ = register[StructType](ids.TypeConstant)` in each struct file.
4. The `Coder` var built from `coderMap` instead of a literal.

#### Affected Packages

| Package | Current Pattern | Structs |
|---------|----------------|---------|
| `golf/repo_blobs` | Map literal | TomlLocalOverridePathV0, TomlXDGV0, TomlUriV0 |
| `golf/repo_configs` | Map literal | V0, V1, V2 |
| `golf/blob_store_configs` | Map literal + `registerToml` (mixed) | V0-V2, SFTP variants, Pointer, InventoryArchive |
| `hotel/genesis_configs` | Map literal (Private + Public) | TomlV1Private/Public, TomlV2Private/Public |
| `hotel/workspace_config_blobs` | Map literal | V0 |

#### Example: `golf/repo_blobs`

**`coding.go` (after):**

```go
var coderMap = make(map[string]interfaces.CoderBufferedReadWriter[*Blob])

func register[IMPL any, IMPL_PTR interface {
    Blob
    interfaces.Ptr[IMPL]
}](typeString string) struct{} {
    if _, ok := coderMap[typeString]; ok {
        panic(fmt.Sprintf(
            "coder for type %q registered more than once",
            typeString,
        ))
    }
    coderMap[typeString] = triple_hyphen_io.CoderToml[Blob, *Blob]{
        Progenitor: func() Blob {
            var impl IMPL
            return IMPL_PTR(&impl)
        },
    }
    return struct{}{}
}

var Coder = triple_hyphen_io.CoderToTypedBlob[Blob]{
    Metadata: triple_hyphen_io.TypedMetadataCoder[Blob]{},
    Blob:     triple_hyphen_io.CoderTypeMapWithoutType[Blob](coderMap),
}
```

**`toml_uri_v0.go` (add one line):**

```go
var _ = register[TomlUriV0](ids.TypeTomlRepoUri)
```

#### Special Case: `golf/blob_store_configs`

Already uses `registerToml` for two types. Migrate remaining five types from the
map literal to `registerToml` calls, then remove the map literal entirely. The
existing `registerToml` function already handles duplicate detection and
progenitor construction.

#### Special Case: `hotel/genesis_configs`

Has two coder maps (Private and Public). The register function takes two
progenitor types:

```go
func register[
    PRIV ConfigPrivate, PRIV_PTR interface { ConfigPrivate; interfaces.Ptr[PRIV] },
    PUB ConfigPublic, PUB_PTR interface { ConfigPublic; interfaces.Ptr[PUB] },
](typeString string) struct{} {
    // register into both privateCoderMap and publicCoderMap
}
```

Each version file (toml_v1.go, toml_v2.go) self-registers once:

```go
var _ = register[TomlV1Private, TomlV1Public](ids.TypeTomlConfigImmutableV1)
```

#### Special Case: `lima/type_blobs`

Uses `blob_library.MakeBlobStore` (requires `envRepo` at construction time), not
`CoderTypeMapWithoutType`. Self-registration records a factory descriptor:

```go
type blobFactory[T any] struct {
    typeString string
    makeFormat func(env_repo.Env) blob_library.BlobFormat[T, *T]
    reset      func(*T)
}
```

`MakeTypeStore` iterates over registered factories to build typed stores
dynamically. The switch statement is replaced by a map built from the registered
factories.

### Part 2: Static Analyzer

A new analyzer at `alfa/analyzers/builtin_coders/` following the same structure
as the existing `repool` analyzer.

#### What It Checks

For each `ids.Type*` constant that:
- Is registered via `registerBuiltinTypeString` with a genre of Tag, Type,
  Config, or Repo (genres that have `*_blobs` packages)
- Is not deprecated (no `// Deprecated` comment)

The analyzer verifies there exists a `register[...]` call somewhere in the
codebase that references that constant.

#### Implementation Approach

Uses `golang.org/x/tools/go/analysis` framework:

1. In a first pass, collect all `register[...]()` calls and extract the type
   string argument.
2. In a second pass (or via facts from the `ids` package), collect all builtin
   type constants.
3. Report any constant without a matching registration.

#### Integration

- `justfile` target: `check-go-builtin-coders`
- Added to `check` target alongside `check-go-repool`
- Runs via `go vet -vettool=build/builtin-coders-analyzer ./...`

### Init Order Safety

The `ids` package `init()` stays unchanged and registers all builtin types.
The `*_blobs` packages only register coders. Since `*_blobs` packages import
`ids`, the `ids` init runs first. Packages like `commands_madder` that call
`ids.GetOrPanic()` in their `init()` are unaffected because they don't depend
on the coder maps.

The coder maps are only accessed at runtime (when encoding/decoding), not during
init. By the time any code path reaches a coder map lookup, all package inits
have completed and all registrations are done.

## Risks

- **Accidental import removal**: If a struct file is not imported (e.g., blank
  import removed), its coder silently disappears. The static analyzer mitigates
  this by flagging missing registrations.
- **Init-time panics on duplicates**: Already the behavior for `registerToml`.
  This is desirable — duplicates should fail hard.

## Testing

- Unit tests for each `register` function: verify duplicate detection panics,
  verify progenitor creates correct type.
- Static analyzer tests with testdata packages (following repool's pattern).
- Existing integration tests (BATS) exercise the full encode/decode path and
  will catch any registration regressions.
