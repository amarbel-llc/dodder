---
name: Dodder Usage
description: >
  Guides day-to-day dodder and der operations including creating zettels,
  querying and filtering objects, checking out working copies, organizing
  content, syncing with remotes, and managing tags, types, and dormant
  objects. Activated by requests to create, edit, organize, query, show,
  checkout, checkin, push, pull, sync, tag, or otherwise operate on dodder
  repositories.
triggers:
  - create a zettel
  - organize zettels
  - dodder checkout
  - dodder push
  - dodder pull
  - query dodder
  - find zettels
  - sync dodder
  - tag a zettel
  - edit a zettel
  - dodder commands
  - der
  - dodder status
  - dodder show
---

# Dodder Usage

## dodder and der

`dodder` and `der` are the same tool. `der` is a shortname alias that behaves
identically. Every command and flag documented here works with either binary.
`madder` is a separate low-level tool for internal repository operations and is
not covered by this skill.

## Creating Content

Create new zettels with `dodder new`. By default, the command creates one empty
zettel and opens it in the configured editor (`$EDITOR` or `$VISUAL`).

```bash
dodder new                         # create one empty zettel, open editor
dodder new -edit=false             # create one empty zettel, no editor
dodder new -edit=false file.txt    # import content from a file
dodder new -tags project -type md  # set metadata at creation time
dodder new -count=5 -edit=false    # create 5 empty zettels at once
dodder new -shas                   # treat arguments as already-checked-in blob SHAs
dodder new -organize               # create zettel(s), then open organize UI
dodder new -delete                 # delete source file after import
```

The `-filter` flag accepts a script path that transforms each input file into
the standard zettel triple-hyphen format before import.

When creating from files, pass one or more file paths as positional arguments.
Each file becomes a separate zettel with its content as the blob. Combine with
`-description`, `-tags`, and `-type` to set metadata on all created zettels.

## Viewing and Querying

Use `dodder show` to query and display objects. The default output format is
`log` (one-line summary per object). Query syntax uses positional arguments with
genre, sigil, tag, and type selectors.

```bash
dodder show :z                     # all zettels (latest versions)
dodder show one/uno                # specific zettel by ID
dodder show -format text one/uno   # detailed multi-line view
dodder show -format json one/uno   # JSON output for scripting
dodder show tag-name:z             # zettels carrying a specific tag
dodder show !md:z                  # zettels of type md
dodder show :?z                    # include dormant/hidden zettels
dodder show :t                     # all tags
dodder show :e                     # all types
dodder show :z,t                   # zettels and tags
dodder show :r                     # all repos
dodder show konfig                 # system configuration object
dodder show -before 2024-01-01 :z  # zettels before a date
dodder show -after 2024-01-01 :z   # zettels after a date
dodder show -repo remote-id :z     # query a remote repository
```

The `-format` flag supports `log` (default, one-line), `text` (detailed
multi-line), and `json` (machine-readable).

Use `dodder cat` for raw blob output, and `dodder last` to display the most
recently changed objects from the latest inventory list.

## Editing

Open a zettel for editing in the configured editor with `dodder edit`. The
`-mode` flag controls what to edit.

```bash
dodder edit one/uno                       # edit metadata and blob (default)
dodder edit -mode metadata-only one/uno   # edit metadata only
dodder edit -mode blob-only one/uno       # edit blob only
dodder edit -delete one/uno               # delete after editing
dodder edit -organize one/uno             # open organize after editing
```

After external edits (outside the editor), use `dodder checkin` to commit
changes back to the store. `checkin` has two aliases: `add` and `save`.

```bash
dodder checkin one/uno                    # commit changes from working copy
dodder checkin -ignore-blob one/uno       # update metadata only, keep blob
dodder checkin -each-blob "cmd" :z        # run command on each blob
dodder checkin -tags project :z           # add tags during checkin
dodder checkin -type md :z                # set type during checkin
dodder checkin -delete :z                 # delete working copy after checkin
dodder checkin -organize :z               # open organize after checkin
```

## Checkout and Working Copy

The checkout system syncs objects from the store to the filesystem. Use
`dodder checkout` to materialize objects as files.

```bash
dodder checkout :z                 # checkout all zettels
dodder checkout one/uno            # checkout a specific zettel
dodder checkout -organize :z       # checkout then open organize UI
dodder checkout :?z                # include dormant objects
```

View the current state of checked-out objects with `dodder status`.

```bash
dodder status                      # show all checked-out objects and their state
dodder status one/uno              # show state for specific object
```

Remove checked-out files from the working directory with `dodder clean`.

```bash
dodder clean                       # remove unchanged checked-out files
dodder clean -force                # remove all checked-out files regardless of changes
dodder clean -organize             # open organize UI to select files to remove
dodder clean -recognized-blobs     # also remove recognized but not checked-out blobs
```

View differences between the store and working copy with `dodder diff`.

```bash
dodder diff                        # diff all checked-out objects
dodder diff one/uno                # diff a specific object
```

## Organizing

`dodder organize` opens a text-based bulk editing interface. It generates a
structured text file listing objects, which can be edited to rename, retag,
retype, or reorganize objects. Changes are committed when the editor closes.

```bash
dodder organize :z                          # organize all zettels
dodder organize -mode output-only :z        # preview the organize file (stdout)
dodder organize -mode commit-directly :z    # apply changes from stdin, no editor
dodder organize -mode interactive :z        # open vim to edit (default for TTY)
dodder organize -filter script.lua :z       # transform via script
dodder organize tag-name:z                  # organize zettels with a specific tag
dodder organize !md:z                       # organize zettels of a specific type
```

The organize interface represents objects as a hierarchical text document. Move
lines to change tag assignments, edit descriptions, and retype objects. Deleting
a line from the organize file does not delete the object from the store.

## Remote Sync

Dodder supports push/pull synchronization with remote repositories. Remotes are
registered objects within the local store.

### Add a remote

```bash
dodder remote-add remote:///path/to/repo remote-id
```

The first argument is the remote URI and the second is the local name. The
`-tags` and `-description` flags set metadata on the remote object.

### Clone a repository

```bash
dodder clone new-repo-id remote:///path/to/source
dodder clone new-repo-id remote:///path -override-xdg-with-cwd
```

Clone creates a new local repository and pulls all inventory lists from the
remote. Additional query arguments filter what to clone.

### Push and pull

```bash
dodder push remote-id              # push local changes to remote
dodder pull remote-id              # pull remote changes to local
dodder pull-blob-store path config # pull blobs from a blob store
```

Push and pull transfer inventory lists and their referenced objects. Both
commands accept query arguments to limit what to transfer.

## Workspaces

A workspace is a directory linked to a dodder repository. It provides default
tags, type, and query for operations within that directory.

```bash
dodder init-workspace              # create workspace in current directory
dodder init-workspace subdir       # create workspace in a subdirectory
dodder init-workspace -tags work   # set default tags for the workspace
dodder init-workspace -type md     # set default type for the workspace
dodder init-workspace -query "project:z" # set default query
dodder info-workspace              # show workspace configuration
dodder info-workspace query        # show default query
dodder info-workspace defaults.tags # show default tags
dodder info-workspace defaults.type # show default type
```

When a workspace exists, `checkin`, `new`, and `organize` automatically apply
the workspace's default tags. `show` uses the workspace's default query when no
arguments are given.

## Query Syntax

Queries are positional arguments passed to commands like `show`, `checkout`,
`organize`, `edit`, and `checkin`. The syntax combines object IDs, genre
selectors, sigils, tag filters, and type filters.

| Syntax | Meaning |
|--------|---------|
| `:z` | All zettels (latest) |
| `:t` | All tags |
| `:e` | All types |
| `:r` | All repos |
| `:b` | All inventory lists |
| `:z,t` | Zettels and tags |
| `one/uno` | Specific object by ID |
| `tag-name:z` | Zettels with a specific tag |
| `!md:z` | Zettels of a specific type |
| `:?z` | Including dormant/hidden |
| `konfig` | System configuration object |

Sigil characters modify selection scope:

| Sigil | Character | Scope |
|-------|-----------|-------|
| Latest | `:` | Current versions only (default) |
| Hidden | `?` | Include dormant objects |
| History | `+` | Include historical versions |
| External | `.` | Include checked-out/external objects |

Refer to `references/querying.md` for the complete query syntax reference with
compound queries, time filtering, and practical examples.

## Working Copy vs Store

The `.dodder/` directory (or XDG-based equivalent) is the store. It contains all
blobs, inventory lists, indices, and configuration. The filesystem outside
`.dodder/` is the working copy.

- `checkout` syncs store to filesystem (read from store, write to working copy).
- `checkin` syncs filesystem to store (read from working copy, write to store).
- `status` shows the relationship between store and working copy.
- `diff` shows differences between the two.
- `clean` removes working copy files.

The store is always the source of truth. The working copy is ephemeral and can be
fully recreated from the store at any time.

## Tags and Types

Tags and types are first-class objects in dodder, each with their own IDs and
metadata stored in the repository.

**Tags** organize and categorize objects, similar to labels. A zettel can carry
any number of tags. Tags are referenced in queries by name (e.g.,
`project:z` matches all zettels tagged `project`).

Tag prefixes carry special meaning:

| Prefix | Meaning |
|--------|---------|
| (none) | Regular tag |
| `-` | Dependent tag (implicit, derived from parent) |
| `%` | Virtual tag (computed, not stored directly) |

**Types** define the format of a zettel's blob content, similar to file
extensions. Each zettel has exactly one type. Types are referenced in queries
with the `!` prefix (e.g., `!md:z` matches all markdown zettels).

Both tags and types can be viewed with `dodder show :t` (tags) and
`dodder show :e` (types).

## Dormant Objects

Dormant objects are hidden from default queries. Mark objects as dormant to
archive or suppress them without deleting.

```bash
dodder dormant-add tag-name        # hide objects matching this tag
dodder dormant-remove tag-name     # unhide objects matching this tag
dodder dormant-edit                # open dormant configuration in editor
```

Dormant objects reappear in queries that use the `?` sigil:

```bash
dodder show :?z                    # all zettels including dormant
dodder checkout :?z                # checkout including dormant
```

The dormant index is part of the repository configuration. Use `dormant-edit` to
directly edit the configuration blob that controls which tags trigger dormancy.

## Maintenance

```bash
dodder reindex                     # rebuild all indices from inventory lists
dodder fsck                        # verify integrity of objects and blobs
dodder fsck -skip-blobs            # skip blob content verification
dodder fsck -skip-probes           # skip probe index verification
dodder repo-fsck                   # verify repository-level integrity
dodder find-missing sha1 sha2      # check if blob SHAs exist in the store
dodder revert one/uno              # revert object to previous version
dodder revert -last                # revert all changes from last inventory list
dodder info store-version          # show current store version
dodder info compression-type       # show blob compression type
dodder info age-encryption         # show encryption status
dodder info env                    # show environment variables
dodder info xdg                    # show XDG directory paths
dodder deinit                      # remove dodder repository (requires confirmation)
dodder deinit -force               # remove without confirmation
```

`reindex` rebuilds the stream index and other derived data structures from the
authoritative inventory lists. Run it after a suspected corruption or after
manual edits to the store.

`fsck` verifies that every object's metadata, digest, and blob content are
consistent. It reports errors but does not modify data.

## Repository Initialization

```bash
dodder init my-repo-id                         # create new repo with XDG layout
dodder init -override-xdg-with-cwd my-repo-id  # create .dodder in current dir
dodder init -yin words.txt my-repo-id           # custom left-part word list
dodder init -yang words.txt my-repo-id          # custom right-part word list
```

The `-override-xdg-with-cwd` flag creates a `.dodder/` directory in the current
working directory instead of using XDG base directories. This is useful for
self-contained repositories.

## Global Flags

These flags apply to all commands:

| Flag | Purpose |
|------|---------|
| `-dir-dodder` | Override the dodder directory path |
| `-ignore-workspace` | Ignore workspace defaults |
| `-ignore-hook-errors` | Continue despite hook failures |
| `-hooks` | Path to hooks configuration |
| `-checkout-cache-enabled` | Enable checkout caching |
| `-predictable-zettel-ids` | Generate IDs in order (testing) |
| `-comment` | Attach a comment to the inventory list |
| `-debug` | Enable debug output |
| `-dry-run` | Preview changes without applying |
| `-verbose` | Increase output verbosity |

## Reference Documents

- `references/commands.md` -- Complete command reference organized by workflow,
  with flags and one-line examples.
- `references/querying.md` -- Full query syntax reference with genres, sigils,
  filters, compound queries, and practical examples.
