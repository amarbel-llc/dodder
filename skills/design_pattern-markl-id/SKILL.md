---
name: design_pattern-markl-id
description: >
  Use when working with content-addressable identifiers, markl IDs, blob
  digests, purpose-tagged hashes, blech32 encoding, or encryption keys. Also
  applies when encountering markl.Id, MarklId, FormatHash, purpose@digest
  syntax, or hash format selection.
triggers:
  - markl
  - markl.Id
  - MarklId
  - blech32
  - blob digest
  - purpose@digest
  - FormatHash
  - content addressable
  - hash format
---

# Markl ID System

## Overview

Markl IDs are dodder's content-addressable identifiers. Each ID combines a
purpose tag, a format specifier (hash algorithm or key type), and binary data
into a single value. Human-readable representation uses blech32 encoding (a
bech32 variant using `-` as separator). The system supports multiple hash
algorithms, cryptographic key types, and encryption identifiers through a
unified interface.

## Structure

```go
// echo/markl/id.go
type Id struct {
    purposeId string                          // semantic purpose
    format    domain_interfaces.MarklFormat   // hash/key format
    data      []byte                          // binary content
}
```

### String Representation

Without purpose: `format-encodeddata`
With purpose: `purpose@format-encodeddata`

Example: `dodder-blob-digest-sha256-v1@sha256-qpzry9x8gf2tvdw0s3jn54khce6mua7l...`

## Purpose Tags

Purposes give semantic meaning to otherwise opaque identifiers:

| Purpose Constant | Meaning |
|-----------------|---------|
| `PurposeBlobDigestV1` | SHA-256 digest of blob content |
| `PurposeObjectDigestV1` | SHA-256 digest of full object |
| `PurposeObjectSigV1` | Ed25519 signature of object |
| `PurposeRepoPubKeyV1` | Repository public key |

## Supported Formats

| Format ID | Type | Usage |
|-----------|------|-------|
| `sha256` | Hash | Default blob content hashing |
| `blake2b256` | Hash | Alternative content hashing |
| `ed25519_pub` | Key | Repository public keys |
| `ed25519_sec` | Key | Repository secret keys |
| `ed25519_sig` | Signature | Object signatures |
| `age_x25519_pub` | Key | Age encryption public keys |
| `age_x25519_sec` | Key | Age encryption secret keys |
| `nonce` | Nonce | Cryptographic nonces |

## Blech32 Encoding

Blech32 is a modified bech32 encoding using `-` instead of `1` as the
separator between the human-readable prefix and the data portion. It provides:

- Base32 encoding with checksum validation (6 bytes)
- Human-readable prefix carrying the format ID
- Case-insensitive with validation

```go
// bravo/blech32/main.go
func Encode(hrp string, data []byte) ([]byte, error)
func Decode(encoded string) (hrp string, data []byte, err error)
```

## Pool Integration

Markl IDs are pooled for efficient memory management:

```go
// echo/markl/main.go
var idPool interfaces.PoolPtr[Id, *Id] = pool.MakeWithResetable[Id]()

func GetId() (domain_interfaces.MarklIdMutable, interfaces.FuncRepool) {
    return idPool.GetWithRepool()
}
```

## Creating and Parsing IDs

### Parsing from string

```go
var id markl.Id
err := id.Set("purpose@format-encodeddata")
```

The `Set` method splits on `@` to extract purpose, then decodes the blech32
body to extract format and data.

### Comparing IDs

```go
// echo/markl/util.go
func Equals(a, b domain_interfaces.MarklId) bool {
    // Compares format ID and binary data (ignores purpose)
}
```

### Binary serialization

```go
bytes, err := id.MarshalBinary()   // purpose\x00format\x00data
err = id.UnmarshalBinary(bytes)
```

### Text serialization

```go
text, err := id.MarshalText()      // purpose@format-blech32data
err = id.UnmarshalText(text)
```

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Comparing IDs by string representation | Use `markl.Equals()` which compares format + data |
| Creating IDs without pooling in hot paths | Use `markl.GetId()` with repool |
| Assuming all IDs use SHA-256 | Check `format.GetMarklFormatId()` — multiple formats exist |
| Using bech32 instead of blech32 | Dodder uses `-` separator (blech32), not `1` (bech32) |
