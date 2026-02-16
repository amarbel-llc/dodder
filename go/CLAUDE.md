# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

Dodder is a Zettelkasten-style knowledge management system written in Go. It
provides Git-like version control for managing interconnected notes (Zettels)
with content-addressable storage, sophisticated querying, and remote
synchronization capabilities.

## Build and Development Commands

All commands should be run from the `go/` directory.

### Core Development Tasks

-   **Build**: `just build` (builds debug and release binaries to `build/`)
-   **Test**: `just test` (runs Go unit tests + BATS integration tests)
-   **Unit Tests Only**: `just test-go-unit` or `go test -v -tags test,debug ./...`
-   **Single Package Test**: `go test -v -tags test,debug ./src/path/to/package`
-   **Clean**: `just clean` (clears Go caches)
-   **Check**: `just check` (vulnerability scan + vet)
-   **Generate**: `just build-go-generate` (runs `go generate ./...`)

### Code Quality

-   **Format**: `just codemod-go-fmt` (runs goimports + gofumpt)
-   **Vulnerability Check**: `just check-go-vuln`
-   **Go Vet**: `just check-go-vet`
-   **Repool Analyzer**: `just check-go-repool` (detects leaked or discarded pool return functions)
-   **All Checks**: `just check` (runs vuln + vet + repool)

### Alternative Build Systems

-   **Nix**: `just build-nix` (requires Nix with flakes)
-   **Docker**: `just build-docker`

### BATS Integration Tests

-   **Run all**: `just test-bats` (generates fixtures + runs tests)
-   **Generate fixtures only**: `just test-bats-generate`
-   **Run tests only**: `just test-bats-run` (requires pre-generated fixtures)
-   **Specific test file**: `cd ../zz-tests_bats && bats clone.bats`
-   **Filter by tag**: `cd ../zz-tests_bats && just test-tags migration`
-   **Show failed files**: `just test-bats-failed_files`

See the root `CLAUDE.md` for the full fixture workflow documentation, including
how to regenerate and commit fixtures after store format changes.

## Architecture Overview

### Module Organization

The codebase uses NATO phonetic alphabet naming (alfa, bravo, charlie, etc.) to
enforce a DAG dependency structure - each layer can only depend on previous
layers alphabetically. This prevents circular dependencies and encourages
modularity.

### Core Domain Model

#### Key Concepts

-   **Zettels**: Fundamental content units with unique IDs and metadata
-   **SKUs (Stock Keeping Units)**: Versioned objects representing all content
    types
-   **Object IDs**: Three-part identifiers with genre/type information
-   **Content-Addressable Storage**: SHA-based blob storage with inventory
    tracking
-   **Tags and Types**: Hierarchical organization system
-   **Working Copy**: Git-like checked-out file system

#### Critical Types and Interfaces

-   `sku.Transacted`: Core versioned object type (src/juliett/sku/)
-   `interfaces.ObjectId`: Universal identifier interface (src/alfa/interfaces/)
-   `store.Store`: Main storage engine (src/mike/store/)
-   `command.Command`: CLI command interface (src/golf/command/)

### Storage Architecture

-   **Blob Store**: Content-addressable binary storage
-   **Inventory Lists**: Object metadata and relationship tracking\
-   **Stream Index**: Binary indexing for fast object access
-   **Zettel ID Index**: Specialized indexing for note relationships
-   **Dormant Index**: Inactive/archived object tracking

### Command System

Commands follow a consistent pattern:
-   Flag parsing via `flag.FlagSet`
-   Request objects with context and configuration
-   Standardized error handling through `alfa/errors`
-   Entry points: `cmd/dodder/`, `cmd/der/`, `cmd/madder/`

## Key Development Patterns

### Error Handling

Uses custom error system in `src/alfa/errors/` with: - Context-aware error
wrapping - Stack trace support - Signal handling for graceful shutdown - Helpful
error formatting

### Object Lifecycle

1.  Objects created as `sku.Proto` (prototype)
2.  Transacted through `sku.Transacted`
3.  Stored via content-addressable SHA
4.  Indexed for query and retrieval
5.  Can be checked out to filesystem

### sku.Transacted Pool Management

**CRITICAL REQUIREMENT**: `sku.Transacted` objects must follow strict pool management and **NEVER** be dereferenced:

-   **Never dereference `sku.Transacted` pointers**: Never use `*object` - this violates pool management
-   **Use ResetWith for value structures**: When you need a value type, create a local value and use `ResetWith`
-   **Pool management for persistence**: Use `sku.GetTransactedPool().Get()` and `object.CloneTransacted()` for objects that persist
-   **Always return to pool**: Use `defer sku.GetTransactedPool().Put(object)` after cloning
-   **Reset when needed**: Use `sku.TransactedResetter.Reset()` or `sku.TransactedResetter.ResetWith()` for clean state

#### Correct Patterns:

**For temporary value structures (no dereferencing - preferred pattern):**
```go
// Create target structure and reset its field directly from source
typedBlob := &triple_hyphen_io2.TypedBlob[sku.Transacted]{
    Type: tipe,
    // Blob field is zero-value sku.Transacted
}
sku.TransactedResetter.ResetWith(&typedBlob.Blob, sourcePointer)
// Use typedBlob directly - no copying, no dereferencing
return encoder.EncodeTo(typedBlob, writer)
```

**For simple local values (alternative pattern):**
```go
// Create a local value structure and reset it with source data
var valueObject sku.Transacted
sku.TransactedResetter.ResetWith(&valueObject, sourcePointer)
// Use valueObject directly as a value type
```

**For persistent objects (with pool management):**
```go
// Clone and return to pool for objects that persist
clonedObject := originalObject.CloneTransacted()
defer sku.GetTransactedPool().Put(clonedObject)

// Get from pool and return
newObject := sku.GetTransactedPool().Get()
defer sku.GetTransactedPool().Put(newObject)
sku.TransactedResetter.ResetWith(newObject, sourceObject)
```

#### NEVER DO:
```go
// INCORRECT: Direct dereferencing - NEVER DO THIS
// value := *object  // VIOLATES POOL MANAGEMENT
// someStruct.Field = *object  // VIOLATES POOL MANAGEMENT
```

This pattern ensures efficient memory usage, prevents memory leaks, and maintains strict separation between pointer-managed pool objects and temporary value structures.

### Interfaces and Versioned Structs with Typed Blob Store

The system uses a sophisticated pattern for type-safe, versioned data structures:

#### Interface-First Design

- **Common Interfaces**: Define stable contracts in `src/alfa/interfaces/` (e.g., `BlobStoreConfigImmutable`)
- **Versioned Implementations**: Multiple struct versions implement the same interface (e.g., `TomlV1Common`, `TomlV2Common`)
- **Backward Compatibility**: Old versions remain functional while new versions add features

#### Typed Blob Store Pattern

- **Generic Type Safety**: `typed_blob_store.BlobStore[T, TPtr]` provides compile-time type checking
- **Format Abstraction**: Each content type has a dedicated formatter handling serialization
- **Version Resolution**: Triple-hyphen IO system maps type strings to appropriate decoders

#### Example: Configuration Evolution

```go
// Common interface (stable)
type BlobStoreConfigImmutable interface {
    GetBlobCompression() BlobCompression
    GetBlobEncryption() BlobEncryption
    GetLockInternalFiles() bool
}

// V1 implementation (embedded config)
type TomlV1Common struct {
    BlobStore BlobStoreTomlV1 `toml:"blob-store"`
}

// V2 implementation (referenced config)
type TomlV2Common struct {
    BlobStores       map[string]BlobStoreReference `toml:"blob-stores"`
    DefaultBlobStore string                        `toml:"default-blob-store"`
}
```

#### Key Benefits

- **Type Safety**: Compile-time verification of data structure compatibility
- **Version Migration**: Gradual migration from old to new formats
- **Interface Stability**: External code depends on interfaces, not implementations
- **Extensibility**: New versions can add fields without breaking existing code

### Pool Repool Lifecycle Rules

`GetWithRepool()` returns `(element, FuncRepool)`. The repool function must be
called exactly once when the caller is done with the element. Three tools enforce
this:

1.  **Static analyzer** (`just check-go-repool`): CFG-based `go vet` checker in
    `src/alfa/analyzers/repool/`. Detects discarded repool functions (blank `_`
    without `//repool:owned`) and repool variables not called on all code paths.

2.  **Runtime debug poisoning** (`repool_debug.go`, build tag `debug`): Wraps
    every repool function with an `atomic.Bool` guard that panics on
    double-repool. Tracks outstanding borrows via `pool.OutstandingBorrows()`.
    Zero overhead in release builds (`repool_release.go`).

3.  **Lint check** (`bin/lint.bash`): Grep-based check for discarded repool
    functions missing `//repool:owned` annotations (currently TODO-P2, pending
    annotation of existing intentional discards).

**Common repool pitfall — hash lifetime in blob readers/writers:**

When a pooled `hash.Hash` is embedded in a blob reader or writer (via
`markl_io.MakeWriter`), the hash's lifetime extends from construction through
all reads/writes to `GetMarklId()` *after* `Close()`. The `localFileMover`
pattern calls `writer.Close()` then `GetMarklId()` to compute the digest for
the final file path. Any repool before `GetMarklId()` corrupts the digest.

Because value pools (`pool.MakeValue[Hash]`) share the underlying `hash.Hash`
interface pointer across copies, a premature `Reset()` via repool corrupts all
copies simultaneously. If the hash lifetime cannot be bounded to a single scope,
discard the repool with `//repool:owned`:

```go
hash, _ := config.hashFormat.GetHash() //repool:owned
writer.digester = markl_io.MakeWriter(hash, nil)
```

### Testing Strategy

-   Unit tests: `*_test.go` files throughout codebase
-   Integration tests: BATS framework in `zz-tests_bats/`
-   Test data: Generated fixtures via `test-generate_fixtures`

## Module Import Patterns

-   Import paths follow `code.linenisgreat.com/dodder/go/src/{module}/{package}`
-   Respect the NATO alphabet dependency hierarchy (earlier letters cannot import later letters)
-   Use existing interfaces rather than concrete types where possible
-   Follow established patterns in similar modules

## Common Development Pitfalls

### When Adding New Blob Stores

1. **Type Registration**: New blob store configs need THREE registrations:
   - Add type constant to `src/echo/ids/types_builtin.go` (e.g., `TypeTomlBlobStoreConfigSftpV0`)
   - Register in init() function of the same file
   - Add to type map in `src/echo/blob_store_configs/io.go`

2. **Interface Implementation Gotchas**:
   - `TemporaryFS` uses `FileTempWithTemplate()` not `TempFile()`
   - SHA writers are created with `sha.MakeWriter()` not `sha.NewWriter()`
   - Implement `ReadFrom()` method when creating custom `interfaces.ShaWriteCloser`
   - `interfaces.Sha` is already a pointer type - never use `*interfaces.Sha`

3. **Build Commands**:
   - `just build` may fail if dependencies are missing
   - Use `go build -o build/dodder ./cmd/dodder/main.go` as fallback
   - Dependencies are added with `go get` (e.g., `go get github.com/pkg/sftp`)

4. **SHA Type Handling**:
   - Use `sha.WriteCloser` type alias which maps to `interfaces.ShaWriteCloser`
   - Access SHA values via `GetShaLike()` method, not by dereferencing
   - SHA paths use Git-like bucketing: first 2 chars as directory

5. **Streaming vs Temporary Files**:
   - Remote blob stores should use remote temporary files with atomic moves
   - Create temporary files on remote server, then rename to final location
   - Use Git-like bucketing: `id.Path()` for final path generation
   - Implement proper cleanup with defer statements

6. **Compression/Encryption Streaming**:
   - Use `env_dir.WriteOptions` and custom writers for output streaming
   - For input, implement custom readers that handle the full pipeline
   - Chain: file -> decryption -> decompression -> SHA calculation
   - Avoid `env_dir.ReadOptions` for non-file streams (expects `*os.File`)

7. **Error Wrapping**: Always use `errors.Wrap()` or `errors.Wrapf()` for consistent error handling
