---
author:
- 
date: April 2026
title: MARKL-ID(7) Dodder \| Miscellaneous
---

# NAME

markl-id - dodder content-addressable identifier format

# SYNOPSIS

\[*purpose***@**\]*format***-*\*\*data\*

# DESCRIPTION

A markl ID is a self-describing, checksummed, human-readable identifier for
binary data in dodder. It encodes cryptographic digests, signatures, and keys
using blech32, a modified bech32 encoding.

Markl IDs appear throughout dodder: as blob digests in object metadata, as
signatures in inventory lists, as type locks in hyphence documents, and as
repository public keys.

# STRUCTURE

A markl ID has the text form:

    [purpose@]format-data

**format**
:   Identifies the binary data type (hash algorithm, key type, signature
    scheme). Serves as the blech32 human-readable part (HRP).

**purpose** (optional)
:   Provides semantic context for how the data is used. Separated from the
    format by \*\*@*\*.

**data**
:   The blech32-encoded binary payload including a 6-character checksum.

# EXAMPLES

A BLAKE2b-256 blob digest (no purpose):

    blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd

The same digest with a purpose:

    dodder-blob-digest-sha256-v1@blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd

An Ed25519 signature:

    ed25519_sig-qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw

# BLECH32 ENCODING

Blech32 is identical to BIP173 bech32 except the separator is **-** (hyphen)
instead of **1**. It uses the 32-character alphabet:

    qpzry9x8gf2tvdw0s3jn54khce6mua7l

This alphabet excludes visually ambiguous characters: **1** (one), **b** (bee),
**i** (eye), **o** (oh).

The encoding converts binary data from 8-bit to 5-bit groups, maps each through
the alphabet, and appends a 6-character BCH checksum. The checksum detects all
single-character substitutions and adjacent transpositions.

The entire string must be uniformly cased (all upper or all lower). The
90-character length limit from BIP173 is not enforced.

# FORMAT IDS

  Format               Size   Description
  -------------------- ------ ---------------------------
  sha256               32     SHA-256 digest
  blake2b256           32     BLAKE2b-256 digest
  ed25519_pub          32     Ed25519 public key
  ed25519_sec          64     Ed25519 private key
  ed25519_sig          64     Ed25519 signature
  ecdsa_p256_pub       33     ECDSA P-256 public key
  ecdsa_p256_sig       64     ECDSA P-256 signature
  age_x25519_pub       32     age X25519 public key
  age_x25519_sec       32     age X25519 secret key
  pivy_ecdh_p256_pub   33     PIV ECDH P-256 public key
  nonce                32     Random nonce

Variable-size formats **ed25519_ssh** and **ecdsa_p256_ssh** encode keys in SSH
wire format.

# PURPOSE IDS

Purpose IDs follow the convention *system*-*domain*-*role*-*version*. When
present, the purpose constrains which format IDs are valid.

Common purposes:

**dodder-blob-digest-sha256-v1**
:   Blob content hash. Formats: sha256, blake2b256.

**dodder-object-digest-v2**
:   Object metadata hash. Formats: sha256, blake2b256.

**dodder-object-sig-v2**
:   Object signature. Formats: ed25519_sig, ecdsa_p256_sig.

**dodder-repo-public_key-v1**
:   Repository public key. Formats: ed25519_pub, ecdsa_p256_pub.

**dodder-repo-private_key-v1**
:   Repository private key. Formats: ed25519_sec, ed25519_ssh, ecdsa_p256_ssh.

See RFC 0002 (docs/rfcs/0002-markl-id-format.md) for the complete registry.

# SEE ALSO

**dodder**(1), **doddish**(7)

RFC 0002: Markl ID Format (docs/rfcs/0002-markl-id-format.md)
