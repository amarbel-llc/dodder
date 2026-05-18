---
date: 2026-03-27
promotion-criteria: a standalone binary in go/packages/ can put, get, list, and
  walk history for secrets stored as hyphence blobs in a madder blob store;
  blobs are self-describing and the index is rebuildable from a full blob scan;
  age encryption and blob store scoping are inherited from madder with zero
  additional configuration
status: abandoned
---

# Secret Store

## Problem Statement

Madder provides content-addressed, age-encrypted blob storage with multi-store
scoping, but there is no way to store and retrieve blobs by human-readable name.
Secrets (API keys, credentials, tokens) need name-based access, version history,
and salt-per-write to prevent digest equality from leaking plaintext equality.

Today, users manage secrets in external tools (password managers, environment
files, encrypted dotfiles) that don't participate in madder's storage, scoping,
or encryption model. A thin tool on top of madder could provide secret
management that inherits all of madder's properties --- age encryption at rest,
blob store scoping (CWD, user, system), atomic writes, content-addressed
deduplication of the index --- without modifying madder itself.

## Interface

### Name candidates

Final name undecided. Candidates and their short aliases:

  Name      Alias   Origin
  --------- ------- --------------------------------------
  raunen    rn      German: to whisper (archaic, poetic)
  klatsch   kl      German: gossip
  kvetch    kv      Yiddish: to complain/mutter

### Blob format

Each secret is stored as a hyphence document. The hyphence header carries the
type string; a blank line separates it from the TOML body:

    ---
    type = "<name>-secret-v1"
    ---

    [secret]
    name = "db-password"
    salt = "base64-encoded-32-bytes"
    value = "base64-encoded-plaintext"
    predecessor = "blake2b256-..."
    created = 2026-03-27T14:30:00Z

- **salt**: 32 random bytes, unique per write. Prevents identical secrets from
  producing identical blob digests.
- **predecessor**: digest of the previous version's blob, or empty string for
  the first version. Forms a linked list for history traversal.
- **name**: included in the blob so the index is rebuildable from a full blob
  scan via `BlobStore.AllBlobs()`.

The TOML struct is tommy-codegen'd. Hyphence coders are registered in the
package --- no changes to `go/internal/`.

### Index format

A TOML file mapping names to their current head digest, persisted via
`NamedBlobAccess` (atomic writes through `env_dir.NewMover`):

``` toml
[secrets]
db-password = "blake2b256-..."
api-key = "blake2b256-..."
```

Also tommy-codegen'd. Rebuildable: scan all blobs in the store, parse hyphence
headers to identify secret blobs, follow predecessor chains, find heads.

### Commands

    <name> put <secret-name> [--store <id>]    # read stdin, salt, chain to head, write blob, update index
    <name> get <secret-name>                   # resolve head, read blob, output plaintext
    <name> get <secret-name> --version N       # walk chain N steps back
    <name> history <secret-name>               # walk chain, print digest + created date per version
    <name> ls                                  # list all names + head digests
    <name> rm <secret-name>                    # remove from index (blobs + history preserved in store)
    <name> rebuild-index                       # scan all blobs, reconstruct index from blob metadata

Store scoping is inherited from madder's blob store ID system: - `.secrets` ---
CWD-scoped (per-project) - `secrets` --- XDG user-scoped (personal) - `/secrets`
--- XDG system-scoped

### What is inherited from madder

  Concern              Mechanism
  -------------------- -----------------------------------------------
  Encryption at rest   Age encryption per blob store config
  Store scoping        Blob store ID prefixes (`.`, unprefixed, `/`)
  Atomic writes        `env_dir.NewMover` via `NamedBlobAccess`
  Blob enumeration     `BlobStore.AllBlobs()` for index rebuilds
  Content hashing      `BlobWriter` returns `MarklId` on close

### Package layout

    go/packages/
      <name>/
        main.go                 # CLI entry point
        secret_blob/            # tommy-codegen'd hyphence type for secret blobs
        secret_index/           # tommy-codegen'd TOML type for the name→digest index
        commands/               # put, get, ls, rm, history, rebuild-index

This is the first package under `go/packages/`, establishing the pattern for the
eventual dodder/madder split. The package imports from `go/internal/alfa`
through `golf` (blob store interfaces, blob store ID resolution, age encryption,
hyphence) but nothing from `hotel`/`india` (stream index, object model).

A new top-level flake output exposes the binary.

## Limitations

- **No sync/push/pull.** Secrets stay in the local blob store. Sharing secrets
  between machines requires copying the blob store directory or using madder's
  blob store sync independently.
- **No access control.** Anyone with filesystem access to the blob store and the
  age key can read all secrets. This is a single-user tool, not a team vault.
- **`rm` is index-only.** Removing a secret deletes the name mapping but leaves
  all blobs (current + history) in the content-addressed store. Reclaiming
  storage requires a blob store pack/GC cycle.
- **Salt defeats deduplication.** By design, identical secrets produce different
  blob digests. This is a feature (prevents equality leaks), not a bug.

## More Information

- Madder blob store interfaces: `go/internal/0/domain_interfaces/blob_store.go`
- `NamedBlobAccess` implementation:
  `go/internal/golf/env_repo/named_blob_access.go`
- Blob store ID system: `go/internal/alfa/blob_store_id/`
- Hyphence format: `go/internal/alfa/hyphence/`
- Hyphence blank-line requirement:
  [#41](https://github.com/amarbel-llc/dodder/issues/41)
