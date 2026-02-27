# PIV Token Signing Design

## Goal

Allow dodder to sign inventory list objects using a private key stored on a PIV
hardware token, accessed via the PCSC framework. The private key never leaves
the card -- signing operations are performed on-card via GENERAL AUTHENTICATE
APDUs.

## Use Case

A PIV token (e.g., YubiKey) holds an existing Ed25519 private key in a signing
slot (typically 9c). Dodder uses this key to sign objects in inventory lists,
replacing the current software-only Ed25519 signing path. Verification remains
unchanged -- signatures are standard Ed25519 and can be verified with the
corresponding public key regardless of how they were produced.

## Current Signing Architecture

### Data Flow

```
Genesis Config (TomlV2Private)
    │
    │ GetPrivateKey() → markl.Id (contains raw Ed25519 bytes)
    │ GetPublicKey()  → markl.Id (derived from private key)
    ▼
Object Finalizer (kilo/object_finalizer)
    │
    │ 1. Set metadata.pubRepo from config.GetPublicKey()
    │ 2. Calculate metadata.digSelf (object digest)
    │ 3. privateKey.Sign(objectDigest, sigDst, sigPurpose)
    │ 4. Verify signature
    ▼
markl.Id.Sign() (echo/markl/id_crypto_sec.go:159-208)
    │
    │ Delegates to FormatSec.Sign function pointer
    ▼
Ed25519Sign() (echo/markl/format_family_ed25519.go:71-92)
    │
    │ sec.GetBytes() → cast to ed25519.PrivateKey → privateKey.Sign()
    ▼
Standard Ed25519 signature bytes
```

### Key Types

- **`FormatSec`** (`echo/markl/format_sec.go:17-30`): Struct of function
  pointers (`Generate`, `GetPublicKey`, `Sign`) registered via `makeFormat()`
- **`FuncFormatSecSign`** (`format_sec.go:15`):
  `func(sec, mes MarklId, readerRand io.Reader) ([]byte, error)`
- **`ConfigPrivate`** (`hotel/genesis_configs/main.go:26-31`): Interface
  providing `GetPrivateKey() MarklId` and `GetPublicKey() MarklId`
- **`TomlV2Private`** (`hotel/genesis_configs/toml_v2.go:22-25`): Stores raw
  `markl.Id` with Ed25519 private key bytes

### The Coupling Point

`FuncFormatSecSign` receives the private key as a `MarklId` and calls
`GetBytes()` to extract raw key material. This is the assumption that breaks
with hardware-backed keys -- the private bytes never leave the PIV token.

## Ed25519 on PIV Tokens

Standard PIV (NIST SP 800-73) defines only RSA and ECDSA (P-256/P-384).
Ed25519 on PIV is a proprietary extension:

| Implementation   | Algorithm ID | Notes                              |
|------------------|--------------|------------------------------------|
| pivy             | `0xE0`       | YubiKey-compatible                 |
| go-piv / SoloKeys| `0x22`       | Non-standard; SoloKeys convention  |

Ed25519 PIV support is available on YubiKey 5 series (firmware 5.2.3+) and
SoloKeys. For Ed25519, the full message is sent to the card for on-card
hashing (not a pre-hash), matching pivy's behavior
(`piv.c:4834-4840`, `cardhash = B_TRUE`).

## PCSC Signing Protocol

The signing flow via PCSC (as implemented by pivy at `src/piv.c:4950-5048`):

```
1. Select PIV applet (AID A0 00 00 03 08 00 00 10 00 01 00)
2. Verify PIN (INS_VERIFY, 0x20) if slot requires it
3. GENERAL AUTHENTICATE (INS_GEN_AUTH, 0x87):
   ┌─ 7C (Dynamic Authentication Template)
   │  ├─ 82 (Response placeholder, empty)
   │  └─ 81 (Challenge = message to sign)
   └─
4. Parse response:
   ┌─ 7C
   │  └─ 82 (Response = signature bytes)
   └─
```

For Ed25519 specifically, the challenge is the raw message (not a hash digest),
and the card performs SHA-512 hashing internally as part of Ed25519 signing.

## PIV Slots

| Slot | Name              | Typical Use         | PIN Required |
|------|-------------------|---------------------|--------------|
| 9a   | PIV Authentication| Login, SSH          | Once/session |
| 9c   | Digital Signature | Document/code sign  | Every use    |
| 9d   | Key Management    | Encryption/ECDH     | Once/session |
| 9e   | Card Auth         | Physical access     | Never        |

Slot **9c** is the natural choice for dodder object signing -- it requires PIN
verification before every signing operation, which matches the security
posture of protecting a repo's private key.

## Go Library Options

### go-piv/piv-go

- Pure Go PCSC implementation (no CGo for PCSC layer)
- Implements `crypto.Signer` interface for PIV slot keys
- Supports `AlgorithmEd25519` (SoloKeys convention)
- YubiKey-focused but PIV-compliant operations work on other tokens
- https://github.com/go-piv/piv-go

### cunicu/go-piv (fork)

- Active fork of go-piv/piv-go, uses `ebfe/scard` for PCSC
- `AlgEd25519 = 0x22`
- Apache-2.0 licensed, REUSE compliant
- https://github.com/cunicu/go-piv

### CGo wrapper around pivy

- Direct access to pivy's C library (already in `~/eng/repos/pivy`)
- `PIV_ALG_ED25519 = 0xE0` (YubiKey convention)
- Broader token support, established error handling
- Requires linking against libpcsc, OpenSSL

### Decision

Start with **go-piv/piv-go** (pure Go, `crypto.Signer` interface). Define a
`PIVSigner` abstraction so a pivy CGo backend can be swapped in later if
YubiKey-specific Ed25519 (`0xE0` vs `0x22`) becomes an issue.

## Integration Design

### Strategy: Closure-Captured Signer

Register a new `FormatSec` with format ID `ed25519_piv` whose `Sign` function
closes over a PIV token handle rather than extracting key bytes from the
`MarklId`. The `sec` markl.Id holds the PIV reference (GUID + slot) encoded as
its bytes, and the format ID distinguishes it from software keys during
serialization.

### New Format: `ed25519_piv`

```go
// echo/markl/format.go — new constant
const FormatIdEd25519PIV = "ed25519_piv"
```

The format is registered at runtime (not in `init()`) because it requires a
live `crypto.Signer` from the PIV token:

```go
// echo/markl/format_family_piv_ed25519.go

func RegisterPIVEd25519Format(signer crypto.Signer) {
    makeFormat(FormatSec{
        Id:          FormatIdEd25519PIV,
        Size:        0, // no raw private key bytes
        PubFormatId: FormatIdEd25519Pub,
        GetPublicKey: func(_ domain_interfaces.MarklId) ([]byte, error) {
            pub := signer.Public().(ed25519.PublicKey)
            return []byte(pub), nil
        },
        SigFormatId: FormatIdEd25519Sig,
        Sign: func(
            sec, mes domain_interfaces.MarklId,
            readerRand io.Reader,
        ) ([]byte, error) {
            return signer.Sign(readerRand, mes.GetBytes(), crypto.Hash(0))
        },
    })
}
```

Signatures produced are standard `FormatIdEd25519Sig` — verification uses the
existing `Ed25519Verify` path unchanged.

### Config File Representation

The private-key field in the genesis config TOML uses the new `ed25519_piv`
format ID with a URI-style encoding in the markl.Id bytes. On disk the
serialized markl.Id looks like:

```
dodder-repo-private_key-v1@ed25519_piv1<blech32-encoded PIV reference>
```

The PIV reference bytes encode:

```
piv:guid=<hex-guid>,slot=<slot-id>
```

Example genesis config TOML:

```toml
store-version = 7
id = "..."
inventory_list-type = "dodder-inventory_list-json-v0"
object-sig-type = "dodder-object-sig-v2"
private-key = "dodder-repo-private_key-v1@ed25519_piv1..."
```

When the config loader deserializes this markl.Id and sees format
`ed25519_piv`, it:

1. Decodes the PIV reference from the bytes (GUID + slot)
2. Enumerates PCSC readers, selects the PIV applet on each token
3. Matches by GUID
4. Reads the certificate from the specified slot to extract the Ed25519
   public key
5. Obtains a `crypto.Signer` for that slot
6. Calls `RegisterPIVEd25519Format(signer)` to install the closure-backed
   format
7. The markl.Id's format pointer now resolves to the live `FormatSec`

### Token Discovery

GUID-based enumeration, matching pivy's `piv_box_find_token` approach:

1. `SCardEstablishContext()` (via go-piv)
2. List all smart card readers
3. For each reader: connect, select PIV applet, read CHUID to get GUID
4. Match GUID against the one encoded in the private-key markl.Id
5. Error if no match found ("PIV token with GUID xxxx not found")

This handles tokens moving between readers or USB ports.

### Object Finalizer

No changes. The finalizer calls `config.GetPrivateKey().Sign(...)`, which
dispatches to the `ed25519_piv` format's closure-backed `Sign`, which calls
the PIV token via `crypto.Signer`. The finalizer then calls `Verify()` using
the public key, which uses the standard `Ed25519Verify` path.

### Offline Operation

When the PIV token is absent, dodder operates in read-only mode by default:
reads, queries, and signature verification all work normally using the public
key stored in the config.

Write operations that require signing fail with a clear error:

```
error: PIV token with GUID <guid> not found — cannot sign objects
```

A `--no-sign` flag allows creating unsigned objects during development. These
objects have an empty `sigRepo` field. They are not considered fully valid
and will fail verification. When the token becomes available, the user can
re-commit the objects to add signatures. This is not automatic — the user
must explicitly re-commit.

### PIN Handling

Slot 9c requires PIN verification before each signing operation.

- **Interactive CLI**: Use go-piv's `KeyAuth{PINPrompt: func() (string, error)}`
  callback to prompt the user
- **Automation**: Allow PIN via `DODDER_PIV_PIN` environment variable
- **Session caching**: go-piv handles per-operation PIN verification
  transparently when using `KeyAuth`

### PIV Library Abstraction

Define a signing interface in dodder so the PCSC backend is replaceable:

```go
// echo/markl/piv_signer.go

type PIVSigner interface {
    crypto.Signer
    GUID() [16]byte
    SlotId() byte
    Close() error
}
```

Start with go-piv as the initial backend. A pivy CGo backend can be added
later if YubiKey-specific Ed25519 (`0xE0` vs `0x22`) or broader token support
is needed. The `PIVSigner` interface isolates the rest of the codebase from
the backend choice.

### NATO Hierarchy Placement

| Layer   | Component                      | Role                          |
|---------|--------------------------------|-------------------------------|
| `alfa`  | `domain_interfaces/markl.go`   | No changes needed             |
| `echo`  | `markl/format_family_piv_*`    | `ed25519_piv` format + PIVSigner interface |
| `echo`  | `markl/piv_signer_gopiv.go`    | go-piv backend (initial)      |
| `hotel` | `genesis_configs/toml_v2.go`   | No struct changes — URI in existing private-key field |
| `hotel` | `genesis_configs/piv_connect.go`| Token discovery + format registration |
| `kilo`  | `object_finalizer/`            | No changes needed             |

The PIV dependency (go-piv) enters at `echo`. The `PIVSigner` interface
depends only on `crypto.Signer` (stdlib), so the interface itself has no
external dependencies. Only the go-piv backend implementation imports go-piv.

## Relation to Archive Encryption

The archive encryption design (`2026-02-23-archive-encryption-design.md`)
already anticipates PIV-backed keys for the `IOWrapper` encryption path.
This signing design is complementary:

- **Signing** (this doc): PIV token holds Ed25519 key for object signatures
- **Encryption** (archive doc): PIV token holds X25519 key for blob encryption
  via pivy's ebox/ECDH mechanism (slot 9d, key management)

Both can coexist on the same token using different slots.

## Resolved Design Decisions

1. **Format ID: New `ed25519_piv`**. A distinct format ID makes the PIV backing
   explicit in serialized config files. The markl.Id text encoding becomes
   `purpose@ed25519_piv1...` which unambiguously signals hardware-backed
   signing during deserialization.

2. **Config file: URI in private-key field**. The existing `private-key` TOML
   field holds a markl.Id with format `ed25519_piv` and bytes encoding a PIV
   reference (`piv:guid=<hex>,slot=<id>`). No new TOML fields or config types
   needed — the format ID itself carries all the semantic information.

3. **Token discovery: GUID-based enumeration**. Enumerate PCSC readers, select
   PIV applet on each, match by GUID. Handles tokens moving between readers.
   Mirrors pivy's `piv_box_find_token` approach.

4. **Offline operation: Read-only by default, `--no-sign` flag for unsigned
   objects**. Without the token, reads and verification work normally. Writes
   fail with a clear error. The `--no-sign` flag is a development fail-safe
   that creates unsigned objects (empty `sigRepo`). These must be explicitly
   re-committed when the token is available — no automatic backfill.

5. **PIV library: Abstracted interface, start with go-piv**. Define a
   `PIVSigner` interface wrapping `crypto.Signer`. Implement with go-piv
   initially (`AlgEd25519 = 0x22`). A pivy CGo backend (`PIV_ALG_ED25519 =
   0xE0`) can be added later if YubiKey-specific support is needed.

## Testing

- Unit tests: Mock `crypto.Signer` to test the closure-backed `FormatSec`
  without hardware
- Integration tests: Require a PIV token with Ed25519 key in slot 9c
  (gated behind a build tag or environment variable)
- Round-trip: Sign with PIV, verify with software public key -- confirms
  signature compatibility

## Implementation Notes

- `FormatSec.Size` is set to `PIVReferenceSize` (17 bytes: 16-byte GUID +
  1-byte slot ID) rather than 0. The `setData` size validator in `id.go:234`
  requires `len(bytes) == format.Size` for non-empty data, so a zero-size
  format cannot hold reference bytes.

- `RegisterPIVEd25519Format` is idempotent — calling it multiple times with
  the same or different signers silently returns if the format is already
  registered. This handles config being accessed multiple times.

- `FormatIdEd25519PIV` is registered in `init()` for purpose validation
  (`PurposeRepoPrivateKeyV1`) but the actual `FormatSec` is registered at
  runtime when a PIV token is connected. Between init and runtime registration,
  `GetFormatOrError("ed25519_piv")` will fail — this is expected and only
  happens if the config is accessed before PIV connection.

- go-piv GUID matching uses the serial number encoded as the first 4 bytes,
  since go-piv does not expose the full PIV CHUID/GUID. A future pivy backend
  would use the real 16-byte GUID.

- PIV reference encoding is a flat 17-byte binary format (not the URI string
  described in the original design). The first 16 bytes are the GUID, the
  last byte is the slot ID. This is simpler and avoids string parsing.

- Tests that register different mock signers must `delete(formats,
  FormatIdEd25519PIV)` before calling `RegisterPIVEd25519Format` to avoid
  using a stale signer closure from a previous test. This is a test-only
  concern — in production, the format is registered once.

- Cross-format signature test confirms Ed25519 determinism: signing the same
  message with the same key via `ed25519_piv` (mock signer) and `ed25519_sec`
  (raw bytes) produces byte-identical signatures.
