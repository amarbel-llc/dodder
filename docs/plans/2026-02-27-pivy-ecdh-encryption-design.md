# PIV ECDH Encryption Design (delta/pivy)

## Goal

Allow madder blob stores to encrypt data using a PIV token's ECDH key
(slot 9d), accessed via pivy-agent over a Unix domain socket. The private key
never leaves the hardware token -- ECDH key agreement is performed by the agent,
and symmetric streaming encryption happens locally using age's STREAM format.

## Relation to Existing Designs

- **Archive encryption** (`2026-02-23-archive-encryption-design.md`): Defined
  the `IOWrapper` integration point and the "encrypted payloads, plaintext
  framing" strategy. This design provides a PIV-backed `IOWrapper`
  implementation that slots into that architecture unchanged.
- **PIV token signing** (`2026-02-27-piv-token-signing-design.md`): Covers
  Ed25519 signing via slot 9c. This design covers ECDH encryption via slot 9d.
  Both can coexist on the same token using different slots.

## Architecture

### Key Encapsulation: pivy ECDH, Payload: age STREAM

Pivy's box format (`0xB0C5`) is a single AEAD over the entire payload -- it
cannot stream. Dodder requires streaming encryption to keep memory usage bounded
during pack and sync operations. Age's STREAM format encrypts in 64 KiB
ChaCha20-Poly1305 chunks, supporting true streaming encrypt/decrypt.

The design splits responsibilities:

- **Key encapsulation**: pivy ECDH (P-256) wraps a random 32-byte file key into
  an age stanza of type `pivy-ecdh-p256`
- **Payload encryption**: age's STREAM format encrypts the data in 64 KiB
  chunks using the file key

The pivy-agent is contacted once per encrypt/decrypt to perform the ECDH
operation. After that, streaming is entirely local.

### On-Disk Format

The ciphertext is a standard age file with a pivy-specific stanza:

```
age-encryption.org/v1
-> pivy-ecdh-p256 <compressed_ephemeral_pubkey_b64> <nonce_b64>
<wrapped_file_key_b64>
--- <HMAC>
<STREAM chunked payload: 64 KiB ChaCha20-Poly1305 chunks>
```

This reuses age's full streaming infrastructure. The only pivy-specific part is
the stanza, which contains:

- Stanza type: `pivy-ecdh-p256`
- Arg 0: base64-encoded compressed ephemeral P-256 public key (33 bytes)
- Arg 1: base64-encoded nonce (16 bytes)
- Body: wrapped file key (32 bytes + 16 byte Poly1305 tag = 48 bytes)

### Crypto Operations

**Wrap (encryption, local -- no agent needed):**

1. Generate random 32-byte file key
2. Generate ephemeral P-256 keypair
3. `shared_secret = ECDH(ephemeral_private, recipient_pubkey)`
4. `wrapping_key = SHA512(shared_secret || nonce)[:32]`
5. `wrapped_file_key = ChaCha20-Poly1305.Seal(wrapping_key, file_key)`
6. Zeroize ephemeral private key and shared secret
7. Return age stanza with type `pivy-ecdh-p256`

**Unwrap (decryption -- calls agent):**

1. Parse stanza args: ephemeral pubkey, nonce
2. Connect to pivy-agent at `PIVY_AUTH_SOCK`
3. `AgentECDH("ecdh@joyent.com", recipient_pubkey, ephemeral_pubkey)` ->
   shared secret
4. `wrapping_key = SHA512(shared_secret || nonce)[:32]`
5. `file_key = ChaCha20-Poly1305.Open(wrapping_key, stanza.Body)`
6. Zeroize shared secret and wrapping key
7. Return file key

If no stanza matches, return `age.ErrIncorrectIdentity`. The identity attempts
trial decryption on each `pivy-ecdh-p256` stanza (AEAD open failure means it
wasn't for this recipient). This matches age's X25519 identity behavior.

## Package Structure

### `delta/pivy/` (4 files)

| File | Responsibility |
|------|----------------|
| `recipient.go` | `age.Recipient` impl -- local ECDH key wrapping |
| `identity.go` | `age.Identity` impl -- agent-backed ECDH unwrapping |
| `agent.go` | SSH agent extension client, `PIVY_AUTH_SOCK` resolution |
| `io_wrapper.go` | `IOWrapper` struct, delegates to `age.Encrypt`/`age.Decrypt` |

**`Recipient`** implements `age.Recipient`:

```go
type Recipient struct {
    Pubkey *ecdh.PublicKey
}

func (r *Recipient) Wrap(fileKey []byte) ([]*age.Stanza, error)
```

**`Identity`** implements `age.Identity`:

```go
type Identity struct {
    Pubkey *ecdh.PublicKey
}

func (id *Identity) Unwrap(stanzas []*age.Stanza) ([]byte, error)
```

**`IOWrapper`** implements `interfaces.IOWrapper`:

```go
type IOWrapper struct {
    RecipientPubkey *ecdh.PublicKey
}

func (iow *IOWrapper) WrapWriter(w io.Writer) (io.WriteCloser, error) {
    return age.Encrypt(w, &Recipient{Pubkey: iow.RecipientPubkey})
}

func (iow *IOWrapper) WrapReader(r io.Reader) (io.ReadCloser, error) {
    return age.Decrypt(r, &Identity{Pubkey: iow.RecipientPubkey})
}
```

**Agent client:**

```go
func AgentECDH(
    socketPath string,
    recipientPubkey []byte,
    ephemeralPubkey []byte,
) (sharedSecret []byte, err error)

// TODO: add support for reading socket path from blob store config and CLI
// flags
func ResolveAgentSocketPath() (string, error) {
    // reads PIVY_AUTH_SOCK env var
}
```

Error types distinguish: agent unreachable, extension unsupported, key not
found, ECDH failure.

### `echo/markl/format_family_pivyecdhp256.go`

New format constant and registration:

```go
const FormatIdPivyEcdhP256Pub = "pivy_ecdh_p256_pub"
```

Registered in `init()`:

```go
makeFormat(
    FormatSec{
        Id:           FormatIdPivyEcdhP256Pub,
        Size:         33, // compressed P-256 point
        GetIOWrapper: PivyEcdhP256GetIOWrapper,
    },
)
```

This is a pub-key format: the config stores only the public key. The private
key lives on the hardware token. The `GetIOWrapper` function constructs the
`IOWrapper` from the public key bytes:

```go
func PivyEcdhP256GetIOWrapper(
    id domain_interfaces.MarklId,
) (interfaces.IOWrapper, error) {
    pubkey, err := ecdh.P256().NewPublicKey(id.GetBytes())
    // ...
    return &pivy.IOWrapper{RecipientPubkey: pubkey}, nil
}
```

### `echo/markl/purposes.go`

Add pivy format to the existing madder encryption purpose:

```go
makePurpose(
    PurposeMadderPrivateKeyV1,
    PurposeTypePrivateKey,
    FormatIdEd25519Sec,
    FormatIdAgeX25519Sec,
    FormatIdPivyEcdhP256Pub,  // new
)
```

### `echo/markl/pivy_agent_discover.go`

Discover ECDH-capable keys from pivy-agent:

```go
func DiscoverPivyAgentECDHKeys() ([]Id, error)
```

1. Connect to `PIVY_AUTH_SOCK`
2. `agent.NewClient(conn).List()` -> all public keys
3. Filter for ECDSA P-256 keys
4. For each, extract compressed EC point, construct `markl.Id` with
   `FormatIdPivyEcdhP256Pub`
5. Return list

### `yankee/commands_dodder/info_pivy_agent.go`

CLI command parallel to `info_ssh_agent.go`:

```go
func init() {
    utility.AddCmd("info-pivy_agent", &InfoPivyAgent{})
}
```

Calls `markl.DiscoverPivyAgentECDHKeys()`, prints each key via `MarshalText()`.
Output is one markl.Id-encoded pubkey string per line, ready to copy-paste into
`-encryption`.

## CLI Usage

```sh
# Discover available ECDH keys from pivy-agent
$ dodder info-pivy_agent
pivy_ecdh_p256_pub1...

# Init a blob store with pivy encryption
$ dodder blob_store-init \
  -encryption pivy_ecdh_p256_pub1... \
  shared

# Existing commands work unchanged -- IOWrapper handles encrypt/decrypt
$ dodder blob_store-pack shared
$ dodder blob_store-cat shared <hash>
```

The `-encryption` flag's `default` case calls `encryption.Set(value)`, which
does blech32 decoding and resolves the format ID. No changes to
`golf/blob_store_configs/encryption.go` needed.

## Agent Socket Resolution

The agent socket path comes from the `PIVY_AUTH_SOCK` environment variable.

```go
// TODO: add support for reading socket path from blob store config TOML
// TODO: add -pivy-agent-sock CLI flag
func ResolveAgentSocketPath() (string, error) {
    path := os.Getenv("PIVY_AUTH_SOCK")
    if path == "" {
        return "", errors.Errorf("PIVY_AUTH_SOCK not set")
    }
    return path, nil
}
```

## Curve Support

Initial scope: P-256 only (`nistp256`). This is the default PIV curve,
supported by all YubiKeys, and what pivy uses for slot 9d by default.

```go
// TODO: add pivy-ecdh-p384 and pivy-ecdh-p521 stanza types for P-384 and
// P-521 curve support
```

## Agent Extension

Initial scope: `ecdh@joyent.com` (simple shared secret return).

```go
// TODO: add support for ecdh-rebox@joyent.com extension, which performs
// decryption and re-encryption on the agent side so the shared secret never
// crosses the socket
```

## Dependencies

No new external dependencies. All required packages are already available:

- `golang.org/x/crypto/ssh/agent` -- transitively available (used for SFTP)
- `crypto/ecdh` -- stdlib, P-256 ECDH
- `crypto/elliptic` -- stdlib, EC point compression/decompression
- `golang.org/x/crypto/chacha20poly1305` -- symmetric cipher
- `crypto/sha512` -- stdlib, KDF
- `filippo.io/age` -- already in go.mod

## NATO Hierarchy Placement

| Layer     | Component                          | Role                           |
|-----------|------------------------------------|--------------------------------|
| `delta`   | `pivy/`                            | Box crypto, IOWrapper, agent   |
| `echo`    | `markl/format_family_pivyecdhp256` | Format registration            |
| `echo`    | `markl/pivy_agent_discover`        | Key discovery from agent       |
| `echo`    | `markl/purposes`                   | Add pivy to madder purpose     |
| `yankee`  | `commands_dodder/info_pivy_agent`  | CLI discovery command          |

## Testing

- **Unit tests**: Mock agent (or software ECDH) to test `Recipient.Wrap` +
  `Identity.Unwrap` round-trip without hardware
- **IOWrapper round-trip**: Encrypt with `WrapWriter`, decrypt with
  `WrapReader`, verify plaintext matches -- confirms streaming works
- **Integration tests (bats)**: End-to-end `blob_store-init -encryption` +
  `blob_store-pack` + `blob_store-cat` with a running pivy-agent
- **Compatibility**: Verify that age files with `pivy-ecdh-p256` stanzas are
  valid age format (standard age tooling ignores unknown stanzas gracefully)

## Future: Ebox Recovery

The ebox layer would wrap the file key (not the ciphertext) with N-of-M Shamir
secret sharing. Each share is ECDH-encrypted for a different PIV token. Recovery
reconstructs the file key, which then decrypts the age STREAM payload. The
ciphertext format (age STREAM) does not change -- only the key encapsulation
gains recovery semantics.
