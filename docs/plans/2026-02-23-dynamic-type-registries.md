# Future Direction: Dynamic Type Registries via Dodder Objects

## Vision

All type-to-implementation registries in dodder and madder currently use
compile-time maps:

- `deltaAlgorithms` (byte -> `DeltaAlgorithm`)
- `signatureComputers` (string -> factory)
- `baseSelectors` (string -> factory)
- Compression/encryption wrappers
- Dodder's own type system

The long-term goal is for these registries to support dynamic loading and
registration of concrete implementations at runtime.

## Mechanism

An object stored in dodder whose content implements a Go plugin interface or a
WASM module can be loaded and registered as a concrete implementation of any of
these interfaces:

- `SignatureComputer` — custom similarity signature algorithms
- `BaseSelector` — custom delta base selection strategies
- `io.Writer`/`io.Reader` wrappers — custom compression or encryption
- `DeltaAlgorithm` — custom delta encoding algorithms
- Dodder's type system itself — new object types defined as objects

Dodder becomes self-hosting: the objects it stores define the behavior used to
process them.

## Loading Strategies

- **Go plugins** (`plugin.Open`): Native speed, platform-specific. Suitable for
  performance-critical paths like signature computation and delta encoding.
- **WASM modules** (via wasmtime/wazero): Portable, sandboxed. Suitable for
  user-contributed algorithms where isolation matters.
- **Config-driven**: The blob store config references an object ID. The runtime
  resolves the object, loads the implementation, and registers it in the type
  registry.

## Immutable Blob Signatures

Since blobs are content-addressed and immutable, a similarity signature is a
pure function of the blob content and the signature algorithm. This means:

- Signatures can be computed once at blob ingest time (when the blob enters the
  loose store) and cached permanently
- The cache key is `(blob_id, signature_algorithm_id)`
- When a new signature algorithm is registered (via a new object), existing
  blobs can be lazily re-signed on next pack or eagerly re-signed in batch
- Old algorithm signatures remain valid forever — they just become less useful
  if a better algorithm is available

This applies to both madder (blob store signatures for delta compression) and
dodder (object-level signatures for content-based querying, deduplication, and
related-object discovery).

### Pre-computed Signature Store

A lightweight key-value store mapping `(blob_id, algorithm_id)` to signature
bytes. This could be:

- A column in an existing index file
- A separate signature cache file per algorithm (like the archive cache)
- An object in dodder itself (self-referential: the signature of a blob is
  itself a blob)

The self-referential option is most aligned with the vision: dodder stores
everything as objects, including metadata about objects.

## Relationship to Other Designs

- `2026-02-23-delta-similarity-design.md`: The `SignatureComputer` and
  `BaseSelector` interfaces are designed with this future in mind. The
  registries use `map[string]factory` today, replaced by dynamic loaders later.
- `2026-02-21-delta-compression-design.md`: The `DeltaAlgorithm` registry
  follows the same pattern.
- Dodder's type system: The ultimate expression of this pattern — types
  themselves are objects that define how objects of that type are processed.

## Not Implemented Now

This document captures the direction. The current implementation uses
compile-time registries. Migration to dynamic loading is a separate project that
depends on:

1. Dodder's plugin loading infrastructure (Go plugins or WASM runtime)
2. A stable interface versioning scheme (so old plugins work with new dodder)
3. A trust/verification model for loaded code (especially for WASM)
