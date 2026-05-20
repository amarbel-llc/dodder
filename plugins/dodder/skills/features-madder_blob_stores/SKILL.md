---
name: features-madder_blob_stores
description: Use when creating, querying, or sharing dodder blob stores at the command line --- the `blob_store-*` subcommands, the `-blob_store-id` flag, blob store ID prefixes, encryption choices, or cross-repo sharing via XDG-user stores.
---

# Dodder Blob Stores

Dodder's content lives in **blob stores** --- the content-addressable
storage layer that holds the actual bytes behind each zettel. Most users
never touch blob stores directly; they appear when you initialize a repo,
share storage across repos, or query repo metadata.

Madder commands manage blob stores directly; dodder exposes the same
operations under the `blob_store-` prefix (e.g., madder's `init` becomes
dodder's `blob_store-init`).

## Blob Store ID Format

A blob store ID is `<prefix><name>`. The prefix determines where on the
filesystem the store lives. Two IDs with the same name but different
prefixes refer to *different stores at different paths*.

| Prefix   | Scope            | Example      |
|----------|------------------|--------------|
| *(none)* | XDG user dir     | `shared`     |
| `.`      | Current repo dir | `.default`   |
| `/`      | XDG system dir   | `/system`    |
| `~`      | (compat alias for unprefixed) | `~default` |

Unprefixed IDs default to the XDG user location. The `.` prefix scopes
the store to the current working repo --- the store directory lives
inside the repo on disk and travels with it. Cross-repo sharing uses
unprefixed (XDG user) IDs so any repo running in the same user session
can see the same store.

## Store Types

| Type                          | When to use                                                  |
|-------------------------------|--------------------------------------------------------------|
| `local`                       | Default. Each blob is a file in a hash-bucketed directory.   |
| `local-inventory-archive`     | Packs blobs into archive files; better for large stores.     |
| `sftp` (explicit / via ssh-config) | Remote storage over SFTP.                               |
| `pointer`                     | Indirection to another store's config.                       |

## Common Commands

| Command (dodder)                          | Purpose                                          |
|-------------------------------------------|--------------------------------------------------|
| `blob_store-init <id>`                    | Create a hash-bucketed local store.              |
| `blob_store-init-inventory-archive <id>`  | Create an archive store.                         |
| `blob_store-init-sftp-explicit <id>`      | Create an SFTP store.                            |
| `blob_store-info-repo [store] <key>`      | Read a config value from a store.                |
| `blob_store-pack [store...]`              | Pack loose blobs into archives.                  |
| `blob_store-cat <sha>`                    | Print a blob to stdout.                          |
| `blob_store-write <store> <file>`         | Write a blob into a store.                       |
| `blob_store-sync`                         | Sync blobs between stores.                       |
| `blob_store-fsck`                         | Consistency check.                               |

### info-repo

With no positional args, `blob_store-info-repo` prints the immutable config
of the default store. With one arg, it treats it as a key on the default
store. With two, the first is a store ID and the second a key. Unknown keys
print an error listing the available keys for that store type. Some keys
(`compression-type`, `encryption`, `hash_type-id`, etc.) only exist for
stores that implement the matching feature.

## The `-encryption` Flag

Shared by `blob_store-init`, `blob_store-init-inventory-archive`, and repo
`init`. Accepts:

- `none` --- no encryption.
- `generate` (or empty value) --- auto-generate an age X25519 key.
- A path --- read the key from that file.
- A key string --- use the string directly.

## Cross-Repo Sharing

To share blobs across repos:

1. Create an XDG-user store (unprefixed ID): `dodder blob_store-init shared`
2. Initialize each repo against it: `dodder init -blob_store-id shared ...`

Each repo's metadata stays repo-local, but the blob bytes live in the
shared store and are visible to any other repo on the same machine that
initializes against the same ID.
