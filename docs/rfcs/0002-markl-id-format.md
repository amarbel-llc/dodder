---
date: 2026-07-05
status: proposed
---

# Markl ID Format

## Abstract

A markl ID is a self-describing, checksummed, human-readable identifier for
binary data in dodder. It encodes cryptographic digests, signatures, and keys
using a modified bech32 encoding called blech32, layered with a
purpose-and-format system that provides semantic typing and format validation.

## Introduction

Dodder uses content-addressable identifiers extensively: blob digests index
stored content, object digests represent versioned metadata snapshots,
signatures authenticate objects, and keys identify repositories and blob stores.
Each of these identifiers carries binary data (hashes, signatures, key material)
that must be represented in text form for storage, display, and interchange.

The markl ID format addresses three requirements:

1.  **Human readability.** Identifiers appear in CLI output, metadata files
    (hyphence documents), log messages, and URLs. They must be copy-pasteable
    and visually distinguishable.

2.  **Self-description.** An identifier must declare what kind of data it
    contains (hash algorithm, key type, signature scheme) so that consumers can
    validate and dispatch without out-of-band context.

3.  **Integrity checking.** Transcription errors in identifiers (manual copying,
    OCR, serial protocols) must be detectable without accessing the underlying
    data.

The format uses blech32, a modified bech32 encoding, as its base layer. On top
of this, it adds a two-tier naming system: format IDs identify the binary data
type (e.g., `blake2b256`, `ed25519_sig`), and purpose IDs provide semantic
context (e.g., `dodder-blob-digest-sha256-v1`, `dodder-object-sig-v2`). Purposes
constrain which format IDs are valid, preventing misuse such as treating a
signature as a hash digest.

## Requirements Language

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be
interpreted as described in RFC 2119.

## Specification

### Identifier Structure

A markl ID has the following text representation:

    [<purpose>@]<format>-<blech32-data>

Where:

- `<purpose>` (OPTIONAL): a semantic purpose identifier
- `<format>`: a registered format identifier (the blech32 HRP)
- `<blech32-data>`: the blech32-encoded binary data including checksum

The `@` character separates the purpose from the format. When no purpose is
present, the identifier consists of only the format and encoded data.

Examples:

    blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
    dodder-blob-digest-sha256-v1@blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
    ed25519_sig-qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw

### Blech32 Encoding

Blech32 is a modified version of the bech32 encoding specified in BIP173. The
sole modification is the separator character: blech32 uses `-` (U+002D
HYPHEN-MINUS) where bech32 uses `1`.

This section specifies blech32 as a self-contained encoding scheme.

#### Character Set

Blech32 uses a 32-character alphabet for the data portion:

    qpzry9x8gf2tvdw0s3jn54khce6mua7l

Each character maps to a 5-bit value (0--31) by position in the string. This
character set is identical to BIP173 and deliberately excludes characters that
are visually ambiguous: `1` (one), `b` (bee), `i` (eye), `o` (oh).

#### Encoding

Given a human-readable part (HRP) and binary data:

1.  **Validate HRP.** Every character in the HRP MUST be in the ASCII range
    33--126 (printable, excluding space). The HRP MUST NOT be empty when
    encoding with an HRP. The HRP MUST be uniformly cased (all uppercase or all
    lowercase).

2.  **Convert bits.** Convert the binary data from 8-bit groups to 5-bit groups.
    Pad the final group with zero bits on the right if the input length is not a
    multiple of 5 bits.

3.  **Create checksum.** Compute a 6-value checksum over the HRP and 5-bit data
    values (see Checksum Computation below).

4.  **Assemble output.** Concatenate: the lowercase HRP, the separator `-`, the
    data values mapped through the character set, and the checksum values mapped
    through the character set.

5.  **Apply case.** If the input HRP was uppercase, convert the entire output to
    uppercase. Otherwise, output lowercase.

The result is:

    <hrp>-<encoded-data><checksum>

When encoding without an HRP (data-only mode), skip steps 1, and omit the HRP
and separator from the output. The checksum is computed with an empty HRP.

#### Decoding

Given a blech32 string:

1.  **Validate case.** The input MUST be uniformly cased. Mixed case MUST be
    rejected.

2.  **Find separator.** Locate the last occurrence of `-` in the input. The
    substring before it is the HRP; the substring after it is the data portion.
    The HRP MUST contain at least one character. The data portion MUST contain
    at least 7 characters (6 checksum + at least 1 data character).

3.  **Validate HRP.** Every character in the HRP MUST be in ASCII range 33--126.

4.  **Map data characters.** Convert each character in the data portion to its
    5-bit value using the character set. Any character not in the character set
    MUST be rejected.

5.  **Verify checksum.** Verify the checksum over the lowercase HRP and all
    mapped data values (see Checksum Computation). If verification fails, reject
    the input.

6.  **Convert bits.** Convert the data values (excluding the last 6 checksum
    values) from 5-bit groups to 8-bit groups without padding. If the remaining
    bits in the last 5-bit group are non-zero, reject the input (non-zero
    padding). If more than 4 bits remain unused in the last group, reject the
    input (illegal zero padding).

7.  **Preserve HRP case.** Return the HRP in its original case (before
    lowercasing for checksum verification).

When decoding data-only input (no separator), skip steps 2--3 and use an empty
HRP for checksum verification.

#### Checksum Computation

The checksum uses the same BCH code as BIP173.

**HRP expansion.** For an HRP of length *n*, produce a sequence of 2*n* + 1
values: the high 3 bits of each character (right-shifted by 5), then a zero
byte, then the low 5 bits of each character (masked with 31). All characters are
lowercased before expansion.

**Polymod.** The polynomial modular checksum function processes a byte sequence
and returns a 32-bit integer:

    polymod(values):
        chk = 1
        for v in values:
            top = chk >> 25
            chk = ((chk & 0x1ffffff) << 5) ^ v
            for i in 0..4:
                if (top >> i) & 1 == 1:
                    chk ^= generator[i]
        return chk

The generator values are:

    generator[0] = 0x3b6a57b2
    generator[1] = 0x26508e6d
    generator[2] = 0x1ea119fa
    generator[3] = 0x3d4233dd
    generator[4] = 0x2a1462b3

**Checksum creation.** Compute
`polymod(hrpExpand(hrp) || data || [0,0,0,0,0,0]) ^ 1`. Extract 6 values of 5
bits each from the result, most significant first:
`result >> (5 * (5 - i)) & 31` for *i* in 0..5.

**Checksum verification.** Compute `polymod(hrpExpand(hrp) || data)`. The input
is valid if and only if the result equals 1.

#### Error Detection Properties

The blech32 checksum guarantees detection of:

- Any single character error (substitution) in the HRP or data portion
- Any single transposition of adjacent characters
- Errors in the checksum characters themselves

The checksum is a 30-bit value (6 characters x 5 bits). It is NOT a
cryptographic integrity mechanism --- it detects accidental transcription
errors, not intentional tampering. Cryptographic integrity is provided by the
higher layers (content-addressable digests and ed25519/ECDSA signatures).

#### Differences from BIP173

  Property              BIP173 (bech32)                      Blech32
  --------------------- ------------------------------------ -----------------------
  Separator             `1` (last occurrence)                `-` (last occurrence)
  Character set         `qpzry9x8gf2tvdw0s3jn54khce6mua7l`   identical
  Checksum algorithm    BCH code, polymod == 1               identical
  Max length            90 characters                        no limit
  HRP character range   ASCII 33--126                        identical
  Case handling         uniform required                     identical
  Data-only mode        not specified                        supported (empty HRP)

The `-` separator was chosen because it is unambiguous in dodder's identifier
contexts, where `1` could be confused with `l` in the character set.

The 90-character length limit from BIP173 is not enforced. Markl IDs encoding
64-byte signatures or private keys exceed this limit.

### Format IDs

A format ID identifies the type and expected size of the binary data. Format IDs
serve as the blech32 HRP.

Format IDs MUST consist of ASCII printable characters (33--126) and MUST NOT
contain `@` (which is reserved as the purpose separator) or `-` within the data
encoding context (the last `-` is always the blech32 separator).

#### Registered Formats

  Format ID              Data size (bytes)   Description
  ---------------------- ------------------- -----------------------------------------
  `sha256`               32                  SHA-256 hash digest
  `blake2b256`           32                  BLAKE2b-256 hash digest
  `ed25519_pub`          32                  Ed25519 public key
  `ed25519_sec`          64                  Ed25519 private key
  `ed25519_sig`          64                  Ed25519 signature
  `ed25519_ssh`          variable            Ed25519 key in SSH wire format
  `ecdsa_p256_pub`       33                  ECDSA P-256 public key (compressed)
  `ecdsa_p256_sig`       64                  ECDSA P-256 signature
  `ecdsa_p256_ssh`       variable            ECDSA P-256 key in SSH wire format
  `age_x25519_pub`       32                  age X25519 public key
  `age_x25519_sec`       32                  age X25519 secret key
  `pivy_ecdh_p256_pub`   33                  PIV ECDH P-256 public key (for YubiKey)
  `nonce`                32                  Random nonce value

Implementations MUST reject markl IDs whose decoded data length does not match
the registered size for the format ID. Variable-size formats (`ed25519_ssh`,
`ecdsa_p256_ssh`) are exempt from this check.

Implementations MUST reject markl IDs with unrecognized format IDs.

### Purpose IDs

A purpose ID provides semantic context for a markl ID, indicating how the
underlying data is used within dodder's protocol. Purpose IDs are OPTIONAL --- a
markl ID without a purpose is valid but carries less semantic information.

#### Purpose ID Structure

Purpose IDs follow the naming convention:

    <system>-<domain>-<role>-<version>

Where:

- `<system>`: the software system (`dodder`, `madder`)
- `<domain>`: the functional area (e.g., `blob`, `object`, `repo`,
  `request_auth`)
- `<role>`: the data's function (e.g., `digest`, `sig`, `public_key`,
  `private_key`)
- `<version>`: version tag (e.g., `v0`, `v1`, `v2`)

Some purpose IDs use additional segments (e.g.,
`dodder-object-metadata-digest-without_tai-v1`). The structure is a convention,
not a grammar --- implementations MUST treat purpose IDs as opaque strings for
matching purposes.

#### Purpose Types

Each purpose belongs to a purpose type that groups related purposes:

  Purpose Type              Description
  ------------------------- ---------------------------------------------
  Blob Digest               Content-addressable hash of blob data
  Object Digest             Hash of object metadata snapshot
  Object Signature          Repository signature over an object digest
  Object Mother Signature   Parent-chain signature
  Private Key               Repository or blob store private key
  Public Key                General public key (e.g., madder)
  Repo Public Key           Repository-specific public key
  Request Auth              Authentication challenge/response/signature

#### Registered Purposes

  ------------------------------------------------------------------------------------------------------
  Purpose ID                                       Type                       Allowed Formats
  ------------------------------------------------ -------------------------- --------------------------
  `dodder-blob-digest-sha256-v1`                   Blob Digest                `sha256`, `blake2b256`

  `dodder-object-digest-sha256-v1`                 Object Digest              `sha256`, `blake2b256`

  `dodder-object-digest-v2`                        Object Digest              `sha256`, `blake2b256`

  `dodder-object-digest-v3`                        Object Digest              `sha256`, `blake2b256`

  `dodder-object-metadata-digest-without_tai-v1`   Object Digest              `sha256`, `blake2b256`

  `dodder-object-mother-sig-v1`                    Mother Sig                 `ed25519_sig`

  `dodder-object-mother-sig-v2`                    Mother Sig                 `ed25519_sig`

  `dodder-object-mother-sig-v3`                    Mother Sig                 `ed25519_sig`

  `dodder-repo-sig-v1`                             Object Sig                 `ed25519_sig`

  `dodder-object-sig-v1`                           Object Sig                 `ed25519_sig`,
                                                                              `ecdsa_p256_sig`

  `dodder-object-sig-v2`                           Object Sig                 `ed25519_sig`,
                                                                              `ecdsa_p256_sig`

  `dodder-object-sig-v3`                           Object Sig                 `ed25519_sig`,
                                                                              `ecdsa_p256_sig`

  `dodder-repo-public_key-v1`                      Repo Public Key            `ed25519_pub`,
                                                                              `ecdsa_p256_pub`

  `dodder-repo-private_key-v1`                     Private Key                `ed25519_sec`,
                                                                              `ed25519_ssh`,
                                                                              `ecdsa_p256_ssh`

  `dodder-request_auth-challenge-v1`               Request Auth               (any)

  `dodder-request_auth-response-v1`                Request Auth               (any)

  `dodder-request_auth-repo-sig-v1`                Request Auth               `ed25519_sig`,
                                                                              `ecdsa_p256_sig`

  `madder-public_key-v1`                           Public Key                 `ed25519_pub`,
                                                                              `ecdsa_p256_pub`

  `madder-private_key-v0`                          Private Key                `ed25519_sec`,
                                                                              `age_x25519_sec`

  `madder-private_key-v1`                          Private Key                `ed25519_sec`,
                                                                              `age_x25519_sec`,
                                                                              `pivy_ecdh_p256_pub`
  ------------------------------------------------------------------------------------------------------

When a purpose is present, the format ID MUST be one of the allowed formats for
that purpose. Implementations MUST reject markl IDs where the purpose and format
are incompatible.

### Text Representation

#### Encoding Sequence

To produce the text form of a markl ID:

1.  If a purpose is set, construct the blech32 HRP as `<purpose>@<format>`.
    Otherwise, use `<format>` as the HRP.
2.  Encode the binary data using blech32 with the constructed HRP.

#### Decoding Sequence

To parse a markl ID from text:

1.  Decode the blech32 input, obtaining the HRP and binary data.
2.  Split the HRP at the first `@`:
    - If `@` is present: the left part is the purpose, the right part is the
      format ID.
    - If `@` is absent: there is no purpose; the entire HRP is the format ID.
3.  Look up the format ID in the format registry. Reject if unrecognized.
4.  If a purpose is present, validate that the format ID is allowed for that
    purpose. Reject if incompatible.

### Data-Only Mode

Blech32 supports encoding without an HRP, producing only the encoded data and
checksum. In this mode, the checksum is computed with an empty HRP (the HRP
expansion produces only a single zero byte).

Data-only mode is used internally where the format is known from context and
does not need to be embedded in the encoded string. It MUST NOT be used in
interchange formats or user-facing output, where self-description is required.

## Error Handling

Implementations MUST detect and report the following error conditions:

  -------------------------------------------------------------------------------
  Condition                               Description
  --------------------------------------- ---------------------------------------
  Empty HRP                               HRP is required but not provided

  Separator missing                       No `-` found in input

  Data portion too short                  Fewer than 7 characters after separator

  Invalid HRP character                   Character outside ASCII 33--126 in HRP

  Invalid data character                  Character not in blech32 character set

  Mixed case                              Input contains both upper and lowercase

  Invalid checksum                        Checksum verification failed

  Non-zero padding                        Unused bits in last 5-bit group are
                                          non-zero

  Illegal zero padding                    More than 4 unused bits in last 5-bit
                                          group

  Unknown format ID                       Format ID not in registry

  Incompatible purpose                    Format ID not allowed for given purpose

  Data size mismatch                      Decoded data length does not match
                                          format's expected size
  -------------------------------------------------------------------------------

## Security Considerations

The blech32 checksum detects accidental transcription errors. It provides no
protection against intentional modification. An attacker can compute a valid
checksum for arbitrary data.

Content integrity and authenticity in dodder are provided by higher layers:
content-addressable digests (SHA-256, BLAKE2b-256) ensure blob integrity, and
cryptographic signatures (Ed25519, ECDSA P-256) provide authenticity. The markl
ID format encodes these values but does not itself provide cryptographic
guarantees.

The format ID and purpose system prevents accidental misinterpretation of binary
data (e.g., using a signature where a hash digest is expected). This is a
type-safety mechanism, not a security boundary --- an attacker who can modify
markl IDs can also change format and purpose fields.

Private key formats (`ed25519_sec`, `age_x25519_sec`) encode secret key
material. Implementations MUST handle markl IDs containing private keys with
appropriate care: avoid logging them, store them with restricted file
permissions, and clear them from memory when no longer needed.

## Compatibility

### Legacy Format Aliases

The following legacy format IDs are accepted during decoding and mapped to their
current equivalents:

  Legacy ID                      Maps to
  ------------------------------ ---------------
  `zit-repo-private_key-v1`      `ed25519_sec`
  `dodder-repo-private_key-v1`   `ed25519_sec`

These aliases exist because early versions of dodder (then named "zit") used
purpose-like strings as format IDs before the purpose/format separation was
introduced.

Implementations MUST accept these legacy IDs during decoding. Implementations
MUST NOT emit legacy IDs during encoding.

### Bech32m

Blech32 uses the original bech32 checksum constant (verification against 1), not
the modified constant introduced by BIP350 (bech32m). The two are incompatible:
a valid blech32 string will fail bech32m verification and vice versa.

## Test Vectors

### Valid Encodings

Empty data with HRP:

    A-2UEL5L     (uppercase)
    a-2uel5l     (lowercase)

Data with HRP:

    abcdef-qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw

Long HRP:

    an83characterlonghumanreadablepartthatcontainsthenumber1andtheexcludedcharactersbio-tt5tgs

Mixed content:

    split-checkupstagehandshakeupstreamerranterredcaperred2y9e3w

### Invalid Encodings

  --------------------------------------------------------------------------------------------------------
  Input                                                            Reason
  ---------------------------------------------------------------- ---------------------------------------
  `split-checkupstagehandshakeupstreamerranterredcaperred2y9e2w`   invalid checksum

  `s lit-checkupstagehandshakeupstreamerranterredcaperredp8hs2p`   space in HRP

  `split-cheo2y9e2w`                                               invalid character `o` in data

  `split-a2y9w`                                                    data portion too short

  `-checkupstagehandshakeupstreamerranterredcaperred2y9e3w`        empty HRP

  `pzry9x0s0muk`                                                   no separator

  `A-G7SGD8`                                                       invalid character `G` in data
                                                                   (uppercase of invalid)
  --------------------------------------------------------------------------------------------------------

## References

### Normative

- \[RFC 2119\] Bradner, S., "Key words for use in RFCs to Indicate Requirement
  Levels", BCP 14, RFC 2119, March 1997.
- \[BIP173\] Wuille, P., "Base32 address format for native v0-16 witness
  outputs", Bitcoin Improvement Proposal 173, March 2017.
- \[RFC 0001\] "Hyphence Serialization Format", dodder RFC 0001. Defines the
  metadata format in which markl IDs appear as blob references, type locks, and
  object reference locks.

### Informative

- \[BIP350\] Wuille, P., "Bech32m format for v1+ witness addresses", Bitcoin
  Improvement Proposal 350, December 2020. Defines bech32m, which blech32 does
  NOT implement.
- \[age\] Filippo Valsorda, "age file encryption", 2019. Source of the original
  bech32-with-hyphen modification that inspired blech32.
- dodder horizontal versioning pattern (`design_patterns-horizontal_versioning`
  skill)
