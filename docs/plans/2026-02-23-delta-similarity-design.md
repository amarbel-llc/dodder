# Delta Similarity: Content-Based Base Selection for Inventory Archives

## Motivation

The current `SizeBasedSelector` groups blobs by similar size and picks the
largest as the delta base. This is a rough heuristic — blobs of similar size are
not necessarily similar in content. For zettelkasten workloads where successive
note edits produce blobs that share most of their content, a content-aware
selector produces better delta assignments: smaller deltas, higher compression
ratios, faster sync.

## Design Principles

- **Pluggable composition**: Signature computation and base selection are
  separate interfaces, paired at the config level. Different computers and
  selectors can be mixed to find the right tradeoff for a given workload.
- **Backward compatible**: The existing `SizeBasedSelector` remains the default.
  New selectors are opt-in via config.
- **Parallelization friendly**: Signature computation is per-blob and
  embarrassingly parallel. The packer owns the parallelism; the interfaces are
  synchronous.
- **Scale-ready**: LSH banding provides sublinear candidate finding, handling
  tens of thousands of blobs now and hundreds of thousands in the future.

## Interfaces

### SignatureComputer

Produces a fixed-length similarity signature from blob content:

```go
type SignatureComputer interface {
    SignatureLen() int
    ComputeSignature(content io.Reader) ([]uint32, error)
}
```

Signatures from the same computer are comparable: the fraction of matching
positions at the same index estimates content similarity (Jaccard similarity for
MinHash-based computers).

### BlobMetadata (extended)

```go
type BlobMetadata struct {
    Id        domain_interfaces.MarklId
    Size      uint64
    Signature []uint32 // nil when no SignatureComputer is configured
}
```

### BaseSelector (unchanged)

```go
type BaseSelector interface {
    SelectBases(blobs BlobSet, assignments DeltaAssignments)
}
```

`SizeBasedSelector` ignores `Signature`. New selectors read it.

## Concrete Implementations

### GearCDCMinHashComputer

The first `SignatureComputer` implementation:

1. Split blob content into variable-length chunks using Gear hash (FastCDC-style)
   - Average chunk size: 48-64 bytes (tuned for small zettelkasten blobs)
   - Min chunk: 16 bytes, max chunk: 256 bytes
2. Hash each chunk (FNV-1a or similar fast non-crypto hash) to get a set of
   uint32 chunk fingerprints
3. Compute MinHash signature of length `k` over that set using one-permutation
   hashing

Config parameters: `avgChunkSize`, `minChunkSize`, `maxChunkSize`,
`signatureLen` (k).

### LSHBandingSelector

The first similarity-aware `BaseSelector`:

1. Read `Signature` from each `BlobMetadata`
2. Divide each signature into `bands` bands of `rowsPerBand` rows
3. For each band, hash the rows into a bucket; blobs sharing a bucket in any
   band are candidates
4. For each blob, among its candidates, pick the one with the highest estimated
   Jaccard similarity (fraction of matching signature positions) as the base
5. Call `Assign(blobIndex, bestCandidateIndex)`
6. Respect min/max blob size thresholds
7. Fall back gracefully: blobs with nil signatures or no candidates are left
   unassigned (stored as full entries)

Config parameters: `bands`, `rowsPerBand`, `minBlobSize`, `maxBlobSize`.

The similarity threshold is approximately `(1/bands)^(1/rowsPerBand)`:

| bands | rowsPerBand | k   | threshold |
|-------|-------------|-----|-----------|
| 16    | 4           | 64  | ~0.50     |
| 20    | 5           | 100 | ~0.55     |
| 32    | 4           | 128 | ~0.42     |

## Packer Integration

The packer flow in `pack_v1.go` becomes:

```
collect metadata
  -> compute signatures (new, parallelizable)
  -> split chunks by max pack size
  -> load blob data per chunk
  -> select bases
  -> compute deltas
  -> write archive
```

### Signature phase

1. Packer checks config for a `SignatureComputer` (nil means skip)
2. For each blob in `metas`, open a reader from the loose blob store, call
   `ComputeSignature`, store result in `BlobMetadata.Signature`
3. Parallelization point: a worker pool of N goroutines, each reading one blob
   and computing its signature independently
4. Close reader immediately after computation (no blob data retained)
5. TAP output: `ok N - compute M signatures (Xms)`

Signature computation requires reading blob content, but the packer also reads
content later for delta computation. That is two reads per blob. This is
acceptable because signature reads are streaming (no data retained) and the
second read only happens for the subset of blobs assigned as deltas or bases.
Pre-computed signatures at ingest time (see future directions) eliminate the
first read entirely.

## Configuration

```toml
# !toml-blob_store_config-inventory_archive-v1
hash_type-id = "blake2b-256"
compression-type = "zstd"
loose-blob-store-id = "my-loose-store"

[delta]
enabled = true
algorithm = "bsdiff"
min-blob-size = 256
max-blob-size = 10485760

[delta.selector]
type = "lsh-banding"
bands = 16
rows-per-band = 4
min-blob-size = 256
max-blob-size = 10485760

[delta.signature]
type = "gear-cdc-minhash"
signature-len = 64
avg-chunk-size = 64
min-chunk-size = 16
max-chunk-size = 256
```

When `delta.signature.type` is empty and `delta.selector.type` is
`"lsh-banding"`, the packer errors at startup. When selector is `"size-based"`,
signature config is ignored.

### Config interfaces

```go
type SignatureConfigImmutable interface {
    GetSignatureType() string
    GetSignatureLen() int
    GetAvgChunkSize() int
    GetMinChunkSize() int
    GetMaxChunkSize() int
}

type SelectorConfigImmutable interface {
    GetSelectorType() string
    GetSelectorBands() int
    GetSelectorRowsPerBand() int
    GetSelectorMinBlobSize() uint64
    GetSelectorMaxBlobSize() uint64
}
```

### Type registries

Registries map type strings to factory functions, same pattern as
`DeltaAlgorithm`:

```go
var signatureComputers = map[string]func(SignatureConfigImmutable) SignatureComputer{...}
var baseSelectors = map[string]func(SelectorConfigImmutable) BaseSelector{...}
```

See `docs/plans/2026-02-23-dynamic-type-registries.md` for the future direction
where these registries support dynamic loading via plugins.

## Module Layout

All new code in `echo/inventory_archive`:

| File                                | Contents                              |
|-------------------------------------|---------------------------------------|
| `signature_computer.go`             | `SignatureComputer` interface          |
| `signature_gear_cdc_minhash.go`     | `GearCDCMinHashComputer`              |
| `signature_gear_cdc_minhash_test.go`| Unit tests                            |
| `base_selector_lsh.go`              | `LSHBandingSelector`                  |
| `base_selector_lsh_test.go`         | Unit tests                            |
| `gear_hash.go`                      | Gear hash + CDC chunking primitives   |
| `gear_hash_test.go`                 | Unit tests                            |
| `minhash.go`                        | MinHash one-permutation hashing       |
| `minhash_test.go`                   | Unit tests                            |

Config extensions in `golf/blob_store_configs`:

| File                                | Contents                              |
|-------------------------------------|---------------------------------------|
| `toml_inventory_archive_v1.go`      | Extended with signature + selector config |

Packer changes in `india/blob_stores`:

| File                                | Contents                              |
|-------------------------------------|---------------------------------------|
| `pack_v1.go`                        | Signature phase + config-driven selector |

## Testing Strategy

### Unit tests (echo/inventory_archive)

- **Gear hash**: Known-input chunking. Inserting bytes in the middle only
  affects nearby chunk boundaries. Min/max chunk size enforcement.
- **MinHash**: Two sets with known Jaccard similarity. Signature-based estimate
  within expected error bounds. Identical sets produce identical signatures.
- **GearCDCMinHashComputer**: End-to-end. Two blobs sharing 80% content produce
  high similarity. Unrelated blobs produce low similarity.
- **LSHBandingSelector**: BlobSet with precomputed signatures. Similar blobs
  assigned together, dissimilar ones not. Nil-signature blobs left unassigned.
  Min/max size thresholds respected.
- **Config validation**: LSH selector without signature config errors. Size
  selector with signature config is fine (ignored).

### Integration tests (BATS)

- Pack with LSH selector + gear-cdc-minhash. Verify pack succeeds, produces
  delta entries, unpacked blobs match originals.
- Round-trip: create similar blobs (same text with small edits), pack with LSH,
  read back, verify content integrity.
- Compare pack size: LSH vs size-based on a corpus of similar blobs.

### Benchmarks

- Signature computation throughput: blobs/sec at various sizes
- LSH candidate finding: candidates/sec at 1K, 10K, 100K blobs
- End-to-end pack time: size-based vs LSH on synthetic corpus

## Relationship to Existing Design

This design extends the delta compression system described in
`2026-02-21-delta-compression-design.md`. That document covers the v1 archive
format, delta algorithm interface, and size-based base selection. This document
adds content-similarity-based selection as a second `BaseSelector`
implementation without changing the archive format or delta algorithm layer.
