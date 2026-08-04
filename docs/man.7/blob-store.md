---
author:
-
date: August 2026
title: BLOB-STORE(7) Dodder \| Miscellaneous
---

# NAME

blob-store - dodder content-addressable blob storage

# DESCRIPTION

A blob store is a content-addressable storage backend that holds the raw data
(blobs) referenced by dodder objects. Each blob is identified by a markl ID
derived from its content digest (see **markl-id**(7)). A dodder repository can
have multiple blob stores configured simultaneously, each with a unique store
ID.

Blob stores are managed by the **madder** utility. Most users never touch them
directly; they appear when initializing a repository, sharing storage across
repositories, or inspecting store configuration.

# STORE IDS

Every blob store has an ID of the form *prefix* + *name*. The prefix determines
the storage scope --- where on the filesystem the store lives. Two IDs with the
same name but different prefixes refer to different stores at different paths.

**(unprefixed)**
:   XDG user store. Located under **$XDG_DATA_HOME/madder/blob_stores/**
    (typically **~/.local/share/madder/blob_stores/**). Visible to every
    repository sharing the same XDG environment. Example: **shared**

**.**
:   CWD-relative store. Located under
    **$PWD/.madder/local/share/blob_stores/**, resolved relative to the current
    working directory --- not the ancestor directory where **.madder/** was
    found. The store lives inside the repository tree and travels with it.
    Example: **.archive**

**/**
:   XDG system store. Located under system-wide XDG data directories.

The name portion after the prefix may contain only **\[a-zA-Z0-9_-\]**.

The tilde prefix (**~**) is accepted on parse as backward compatibility for XDG
user stores but is never emitted.

# STORE TYPES

## Local Hash-Bucketed

The default store type. Blobs are stored as individual files in a directory
tree bucketed by digest prefix (similar to Git's object storage). Created with
**madder init**.

## Inventory Archive

Packs multiple blobs into indexed archive files for efficient storage and O(1)
lookups via a fan-out table. Supports optional delta compression. Created with
**madder init-inventory-archive**.

Archive management commands: **madder pack** consolidates loose blobs into
archives, **madder pack-list** lists archive files, and **madder pack-cat-ids**
lists blob digests within archives.

## SFTP

Remote blob store accessed over SSH/SFTP. Two initialization modes:

**madder init-sftp-explicit**
:   Explicit host, port, user, and key path.

**madder init-sftp-ssh_config**
:   Connection parameters resolved from **~/.ssh/config** host entries.

Both support **-discover** to detect an existing remote store's configuration
from its directory structure.

## Pointer

A store that delegates to another store by reference. Created with **madder
init-pointer**. The pointer store does not hold blobs itself but redirects reads
and writes to the target store.

## Multi

A store that composes other stores: one write store plus zero or more read
stores consulted as fallbacks. In **write_through** mode every write lands in
the write store only; read stores are read-only members. Because write_through
imposes no hash-type agreement across members, a read member may use a
different (e.g. legacy single-hash) hash type than the write store. Dodder
authors one multi per repository as its default store (see below).

# THE REPOSITORY DEFAULT STORE

**dodder init** authors the repository's default blob store as a write_through
multi named **default-**\<repo-id\>. Its write store is always
**default-local**, a plain local hash-bucketed store shared by every
repository in the same scope (the multi itself is repo-scoped; only its name
carries the repo ID). For a CWD-scoped repository the write store's ID is
**.default-local**; for an XDG-user-scoped repository it is **default-local**.

The **-blob_store-id** flag on **dodder init** names an additional store to
attach as a read-only fallback:

    madder init shared
    dodder init -blob_store-id shared ...

The named store must already exist --- init fails otherwise --- and is pinned
by its configuration digest. It receives no writes: all blobs the repository
commits land in **default-local**. Use a read fallback to consume blobs that
already live elsewhere (a pre-populated shared store, a legacy remote), not to
route new writes there.

# CROSS-REPO SHARING

Repositories in the same scope share **default-local** automatically: blobs
written by one repository are readable by any other repository whose default
multi wraps the same write store.

To copy blobs into another store explicitly --- for example to populate a
shared store that other repositories attach as a read fallback --- use **madder
sync** with source and destination store IDs:

    madder sync .default-local shared

**madder sync** is idempotent and copies across hash types (each blob is
re-hashed under the destination's hash type).

# ENCRYPTION

The **-encryption** flag is shared by **madder init**, **madder
init-inventory-archive**, and **dodder init**. Its value is interpreted as:

**none**
:   No encryption.

**generate** (or an empty value)
:   Generate a fresh age X25519 key.

*path*
:   If the value names an existing file, the key is read from that file.

*key*
:   Otherwise the value is parsed directly as a markl key ID.

# INLINE STORE SWITCHING

Several madder commands accept positional arguments that can be either data
arguments (file paths, markl IDs) or store IDs. When an argument parses as a
store ID, it switches the active store for all subsequent arguments.

For file-accepting commands (**write**, **pack-blobs**), the shared helper tries
to open the argument as a file first, falling back to store ID parsing. For
digest-accepting commands (**cat**), store ID parsing is tried first since markl
IDs are unambiguous (they start with a hash algorithm name).

Example:

    madder write file1.txt .archive file2.txt file3.txt

This writes **file1.txt** to the default store, then switches to **.archive**
and writes **file2.txt** and **file3.txt** there.

# CONFIGURATION

Blob store configurations are persisted as hyphence-encoded
**blob_store-config** files (see **hyphence**(7)) under each store's
directory. Every configuration carries a uuidv7 **instance-id** minted at
creation, so configuration digests are unique per store instance --- two
stores initialized with identical settings still have distinct digests.

**madder info-repo** inspects store configuration. With no arguments it prints
the default store's immutable configuration. With one argument, the argument
is a key looked up on the default store; with two, the first is a store ID and
the second a key:

    madder info-repo compression-type
    madder info-repo .archive encryption

Unknown keys produce an error listing the keys available for that store's
type. Key availability depends on the store type: for example
**compression-type**, **encryption**, and **hash_type-id** exist only for
stores implementing the matching feature, and **loose-blob-store-id**,
**max-pack-size**, and the **delta.** keys exist only for inventory archives.

**madder init-from** initializes a store from an existing configuration file;
**madder init-from --from-store** copy-migrates an existing store into a new
one with a fresh instance identity. The repository-level counterpart is
**dodder init-from** (fresh repo instance identity, same keys, source
untouched).

# SEE ALSO

**madder**(1), **markl-id**(7), **hyphence**(7)
