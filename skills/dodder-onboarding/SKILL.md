---
name: Dodder Onboarding
description: >
  Introduces dodder, a distributed zettelkasten and content-addressable blob
  store. Covers core concepts (zettels, object IDs, tags, types, blobs, the
  store), installation via Nix, a first-steps walkthrough including repository
  initialization and zettel creation, and the mental model for working with
  dodder. Activated when a user is new to dodder, asks what dodder is, wants
  to get started, install, set up, or learn how dodder works.
triggers:
  - get started with dodder
  - set up dodder
  - install dodder
  - new to dodder
  - what is dodder
  - dodder tutorial
  - learn dodder
  - how does dodder work
---

# Dodder Onboarding

## What Is Dodder

Dodder is a distributed zettelkasten written in Go. It functions as a
content-addressable blob store with automatic two-part IDs, flat organization,
and full version tracking. Think of it as Git for knowledge: every piece of
content is stored by its hash, every change is recorded in an append-only
inventory list, and the working copy on disk is just a checkout of what lives
in the store.

Unlike traditional note-taking tools that impose folder hierarchies, dodder
keeps everything flat. Organization comes from tags and types rather than
directory structure. Each note (called a "zettel") receives an automatically
assigned human-friendly identifier composed of two parts drawn from
user-supplied word lists, producing IDs like `one/uno` or `red/apple`.

Two user-facing binaries exist: `dodder` and `der` (same tool, shorter name). A
third binary, `madder`, is low-level and intended for internal or advanced
repository operations. All three share the same Go module and source tree.

The `.dodder/` directory is the store, analogous to `.git/` in a Git
repository. It holds all blob content (by SHA hash), inventory lists (the
authoritative version history), configuration, and derived indexes.

## Core Concepts at a Glance

| Concept | Description |
|---------|-------------|
| **Zettel** | A note or blob with an automatically assigned ID. The primary content type. |
| **Object ID** | Two-part human-friendly identifier (e.g., `one/uno`) drawn from user-supplied yin and yang word lists. |
| **Tag** | Organizational label attached to zettels. Plain (`project`), dependent (`-dependency`), or virtual (`%computed`). |
| **Type** | Content format definition prefixed with `!` (e.g., `!md`, `!txt`, `!toml-config-v2`). Types are versioned. |
| **Blob** | Raw content stored by its SHA hash (blake2b-256). Same content always produces the same hash. |
| **Store** | The `.dodder/` directory. Holds blobs, inventory lists, configuration, and indexes. Like `.git/`. |

## Installation

Dodder is built and distributed through Nix.

### Build from source

Clone the repository and build with Nix:

```bash
nix build
```

This produces debug and release binaries. Alternatively, enter the development
shell to get `dodder` (and the `der` alias) on the PATH:

```bash
nix develop
```

### Build with Just

From within the development shell or a configured environment:

```bash
just build
```

This compiles debug binaries to `go/build/debug/` and release binaries to
`go/build/release/`.

### Verify installation

```bash
dodder --help
```

Or equivalently:

```bash
der --help
```

Both names invoke the same binary.

## First Steps

### Initialize a repository

Every dodder repository requires two word lists -- called "yin" and "yang" --
that dodder combines to generate unique zettel IDs. Supply them during `init`:

```bash
dodder init -yin <(echo -e "one\ntwo\nthree") \
  -yang <(echo -e "uno\ndos\ntres") my-repo
```

This creates a `my-repo/` directory containing a `.dodder/` store. The yin list
(`one`, `two`, `three`) and the yang list (`uno`, `dos`, `tres`) define the ID
space. Dodder assigns IDs by combining entries from these lists: the first
zettel gets `one/uno`, the second gets `one/dos`, and so on. The cross-product
of the two lists determines how many unique IDs are available before the lists
need to be extended.

Change into the new repository before continuing:

```bash
cd my-repo
```

### Create a zettel

Create a new empty zettel without opening an editor:

```bash
dodder new -edit=false
```

Dodder assigns the next available ID from the yin/yang cross-product, stores
the empty blob, and records the new object in its inventory list. Output shows
the assigned ID and metadata.

To preview which IDs dodder will assign next, use:

```bash
dodder peek-zettel-ids
```

This displays the upcoming IDs from the yin/yang combination without creating
anything.

### View a zettel

Display all zettels in the repository:

```bash
dodder show :z
```

The `:z` query selects the zettel genre. Output shows each zettel's ID, type,
tags, and blob hash in a log-style format. For a single zettel, pass its ID
directly:

```bash
dodder show one/uno
```

### Edit a zettel

Open a zettel in the configured editor:

```bash
dodder edit one/uno
```

This checks out the zettel to a temporary file, opens the editor, and checks
the modified content back in when the editor closes.

### Add tags and set types

Create a zettel with tags and a specific type:

```bash
dodder new -edit=false -tags project,important -type md
```

Tags and types can also be modified after creation through the `edit` and
`organize` commands.

## Mental Model

Understanding dodder requires internalizing a few principles that differ from
traditional file managers and note-taking applications.

### Flat, not hierarchical

There are no folders or directories for organizing content. Every zettel exists
at the same level. Organization comes from tags (labels) and types (format
definitions), not from filesystem paths. This eliminates the problem of deciding
where a note "belongs" and makes cross-referencing straightforward.

### Everything is versioned

Every change to a zettel -- content, tags, type -- is recorded. Inventory lists
(text-format append-only logs stored in `.dodder/local/share/inventory_lists_log`)
serve as the authoritative version history. The stream index (binary pages in
`.dodder/local/share/objects_index/`) provides fast access but is derived from
the inventory lists and can be rebuilt.

### Working copy vs. store

Like Git, dodder distinguishes between the store (`.dodder/`) and the working
copy (files on the filesystem). The store is the source of truth. The working
copy is a convenience for editing.

- `checkout` syncs objects from the store to the filesystem.
- `checkin` syncs changes from the filesystem back into the store.
- Zettels can exist only in the store (internal), only on the filesystem
  (untracked), or in both (checked out). Conflicts arise when both sides
  diverge independently.

### Content-addressable storage

All blob content is stored by its blake2b-256 hash. Identical content always
produces the same hash, providing automatic deduplication. Blobs are immutable
once stored. The hash serves as both the storage key and a content integrity
check.

### Tags organize, types define format

Tags are flexible labels: plain tags (`project`), dependent tags (`-blocked`),
and virtual tags (`%computed`). They can be nested hierarchically. Types define
the content format and are prefixed with `!` (e.g., `!md` for markdown, `!txt`
for plain text). Types are versioned -- the `v1`, `v2` suffixes track format
evolution.

## Next Steps

For day-to-day workflows -- querying, organizing, checking in and out, remote
synchronization -- refer to the **dodder-usage** skill.

For deeper understanding of the object ID system, genres, the type system, tags,
content-addressable storage internals, and the working copy lifecycle, see
`references/concepts.md`.
