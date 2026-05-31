# Dodder

Dodder is a distributed, version-controlled platform for creating, editing, and
deploying notes and blobs. It draws on two lineages:

- **Zettelkasten** — a knowledge-management method (think Roam Research or
  Obsidian) built on small, densely linked, tagged notes with a flat hierarchy.
- **Git** — content-addressable storage where every change is tracked, every
  object is identified by a digest, and history is never lost.

The result is a tool where:

- everything is an **object** with an automatically-assigned identifier;
- there is a flat hierarchy — **no directories**, only tags;
- everything is a note or a blob, and the two are unified under one model;
- every change is tracked, saved, and content-addressed;
- repos can be cloned, pushed, and pulled between machines.

Dodder pairs with **[madder](#blob-stores-and-markl-ids)**, its companion
content-addressable blob-store CLI, and ships an **[MCP
server](#mcp-server)** so agents and editors can query and mutate a repo
programmatically.

## Core concepts

Each concept below has an authoritative manual page under
[`docs/man.7/`](docs/man.7/). The README is the orientation; the manpages are
the reference.

### Objects

Every object in dodder has an object-id, a date, an optional description, an
optional type, and zero or more tags. Optionally it also carries a **blob** — an
arbitrary content-addressed body (the text of a note, an image, a file).

Zettels, types, tags, and repos are all objects, so the same query language,
serialization, and display format apply uniformly across them.

### Zettel IDs

Most Zettelkasten systems use timestamps as identifiers. Timestamps are awkward
to type, autocomplete, and disambiguate, so dodder instead generates IDs by
combining two user-supplied word lists — historically called **yin** and
**yang**. Given a yin list of `{red, green, blue}` and a yang list of
`{apple, banana, orange}`, the available IDs are every unused combination:
`red/apple`, `red/banana`, … `blue/orange`. As new zettels are created, dodder
assigns the next unused pair.

See [`docs/man.7/markl-id.md`](docs/man.7/markl-id.md) for content-addressed
identifiers (a separate concept from human-facing zettel IDs).

### Tags and meta-tags

Tags (etiketten) replace directories. A tag is itself an object, which means a
tag can be tagged — meta-tags let you build conventions like
`priority-0_must`, `area-home`, or `project-2026-q2` and then organize tags the
same way you organize zettels.

### Types

A type (e.g. `!md`, `!task`) is an object that describes how other objects are
interpreted and validated. Object locks can pin the exact version of a
referenced type or tag at commit time, so a note's meaning doesn't silently
drift when its type definition later changes.

### Doddish — the query language

Most commands (`show`, `checkout`, `status`, `edit`, `push`, `pull`, …) accept
**doddish** queries of the form `predicate[sigil][genre]`:

- **predicate** — a tag name (`todo`), a type filter (`!md`), an object id
  (`ceroplastes/midtown`), or empty.
- **sigil** — which object set: `:` latest (default), `+` history, `.`
  checked-out/external, `?` hidden/dormant. Sigils combine (`:.`).
- **genre** — restrict the kind returned: `z` zettels (default), `t` types,
  `e` tags, `b` inventory lists/blobs, `r` repos.

Multiple terms are AND-combined (object ids are the exception — they OR).
Brackets group (`[!md,home]:z`) and `^` negates (`^todo`).

```sh
dodder show :              # all latest zettels
dodder show todo           # zettels tagged todo
dodder show '!md'          # zettels whose type is !md
dodder show '!md:t'        # the !md type object itself
dodder show :t             # all type objects
dodder show 'one/uno+'     # full history of one zettel
```

Full grammar in [`docs/man.7/doddish.md`](docs/man.7/doddish.md).

### Blob stores and markl IDs

Blobs live in **blob stores** — content-addressable backends keyed by **markl
IDs**, self-describing checksummed digests of the form `[purpose@]format-data`
(e.g. `blake2b256-9ft3m74l5…`). Store backends include local hash-bucketed
directories, inventory archives, SFTP, and pointers; a single repo can read
across multiple stores with automatic fallback.

The companion `madder` CLI manages stores directly (`madder init`,
`madder write`, `madder pack`, …). Store IDs use scope prefixes: unprefixed =
XDG user store, `.` = current directory, `/` = system.

See [`docs/man.7/blob-store.md`](docs/man.7/blob-store.md) and
[`docs/man.7/markl-id.md`](docs/man.7/markl-id.md).

### Workspaces

A **workspace** is dodder's working directory — the Git-like staging area where
you check objects out as files, edit them, and check them back in.

```sh
dodder init-workspace      # create a workspace in the current directory
dodder checkout todo       # materialize matching objects as files
$EDITOR ...                # edit them
dodder status              # CheckedOut / Recognized / Untracked / Conflicted
dodder checkin             # persist edits back into the store
dodder clean               # remove checked-out files
```

See [`docs/man.7/workspace.md`](docs/man.7/workspace.md).

### Serialization and display formats

- **hyphence** — the universal on-disk serialization. A `---`-delimited
  metadata section (type, tags, blob references, description) followed by an
  optional body. Every persistent object — zettels, type definitions, repo and
  blob-store configs — uses it. See
  [`docs/man.7/hyphence.md`](docs/man.7/hyphence.md).
- **box** — the compact one-object-per-line output format dodder commands speak:
  `[object-id @blob-digest timestamp !type tag1 tag2] description`. See
  [`docs/man.7/box.md`](docs/man.7/box.md).
- **organize-text** — the batch-editing format used by `dodder organize`:
  markdown headings act as tag scopes, and moving object lines between headings
  re-tags them. See
  [`docs/man.7/organize-text.md`](docs/man.7/organize-text.md).

## A short tour

Create a repo, write a note, query it, edit in bulk, and sync.

```sh
# 1. Initialize a repo
dodder init

# 2. Create and edit objects
dodder new -type md        # create a new md-typed zettel in $EDITOR
dodder edit ceroplastes/midtown
dodder show :              # list everything you've made

# 3. Query with doddish
dodder show todo                      # tagged todo
dodder show 'priority-0_must !task'   # must-do tasks
dodder show ':.z'                     # latest zettels that are checked out

# 4. Bulk-organize via the organize-text buffer
dodder organize todo       # opens an editable buffer; re-tag by moving lines

# 5. Inspect and diff a workspace
dodder init-workspace
dodder checkout '!task'
dodder status
dodder diff

# 6. Sync between repos
dodder clone -direct /path/to/other-repo
dodder pull -direct /path/to/other-repo
dodder push -direct /path/to/other-repo
```

Exact flags for each command live in the manpages and in `dodder <command>
-help`.

## Notable capabilities

These are designed in the [Feature Design Records](docs/features/) (FDRs);
each bullet links to the record with the full rationale and status.

- **Repo disambiguation** — operate on the current-directory repo or the XDG
  user repo within one session via `-repo_id` / `DODDER_REPO_ID`, using
  single-character prefixes (`.`, `/`, `default`).
  [FDR-0003](docs/features/0003-repo-disambiguation.md)
- **Bindingless local transfer** — `push`, `pull`, and `clone` between local
  repos with `-direct <path>`, no pre-registered remote object required.
  [FDR-0004](docs/features/0004-bindingless-local-repo-transfer.md)
- **Two-phase import** — `import` validates an entire inventory upfront
  (blobless types, TAI collisions, duplicates) before committing anything;
  supports `-dry-run`, `-interactive`, and `-omit-tags`.
  [FDR-0002](docs/features/0002-two-phase-import.md)
- **Object locks** — pin the versions of referenced types and tags at commit
  time to prevent silent schema drift.
  [FDR-0001](docs/features/0001-object-locks.md)
- **Two-stage commit** — separate id allocation from persistence for safer
  concurrent mutation, with fine-grained `flock(2)` locking.
  [FDR-0006](docs/features/0006-two-stage-commit.md)
- **Multi-store blob lookup** — blob reads fall back across the default store,
  walk-up ancestors, and the system store automatically.
  [FDR-0015](docs/features/0015-multi-store-blob-lookup.md)
- **Workspace-as-repo** *(experimental)* — `init-workspace -experimental-repo`
  gives a workspace its own commit history and alternative checkout stores.
  [FDR-0005](docs/features/0005-workspace-as-repo.md)

## MCP server

Dodder ships a [Model Context
Protocol](https://modelcontextprotocol.io) server so agents and editors can
read and mutate a repo as a tool:

```sh
dodder install-mcp         # register the server with an MCP client
dodder mcp                 # run the server over stdio
```

It exposes objects, queries, and the box/organize formats described above.

## Building

```sh
just build                 # codegen + debug binary (go/build/debug/) + nix release build
```

The release artifact is the Nix flake's default package (`.#dodder`); debug,
race, coverage, and analyzer variants are built on demand by the test and check
recipes. See [`CLAUDE.md`](CLAUDE.md) for the full build/test workflow.

```sh
just test                  # unit + integration (bats) tests
```

## Documentation map

- [`docs/man.7/`](docs/man.7/) — concept reference manpages (workspace,
  hyphence, blob-store, doddish, markl-id, box, organize-text).
- [`docs/features/`](docs/features/) — Feature Design Records (FDRs): the design
  intent, interface, and status of user-facing features.
- [`docs/decisions/`](docs/decisions/) — Architecture Decision Records (ADRs).
- [`docs/rfcs/`](docs/rfcs/) — interface and wire-format specifications
  (hyphence, markl-id).

## Contributing

At this time, contributions are welcome only after explicitly getting approval
from one of the authors.

The Go source is organized into NATO-phonetic dependency tiers (`alfa`,
`bravo`, `charlie`, …) where each tier may only depend on tiers below it,
enforcing a unidirectional, circular-dependency-free graph. This tiering is
computed and maintained by **dagnabit** — see its implementation in the
[purse-first](https://github.com/amarbel-llc/purse-first) project for how the
levels are derived and packages repositioned.

## License

See [`LICENSE`](LICENSE).
