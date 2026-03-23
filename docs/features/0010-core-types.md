---
date: 2026-03-22
promotion-criteria: "the null type (!) is recognized by the type system without
  a backing object and round-trips through box, hyphence, binary, and inventory
  list formats; a PEG grammar blob and rumdl binary+config blob can be stored as
  !-typed objects; !md's type blob references them via typed blob locks; the
  reference extraction pipeline runs end-to-end on a markdown blob containing
  dodder:// links"
status: exploring
---

# Core Types

## Problem Statement

Types in dodder define how to interpret blob content, but the type system has no
ground truth beneath it. Every type object has a type blob (currently
`!toml-type-v1`), but there is no terminal type --- no fixed point that anchors
the recursion. This prevents storing blobs that need no schema, validation, or
interpretation as first-class typed objects.

Concretely, the `!md` type today has no way to:

1.  **Extract references** from markdown content. Reference discovery uses
    external shell scripts, but there is no native mechanism that participates
    in the blob/object graph.
2.  **Validate content** against markdown style rules. No validation exists
    beyond successful TOML parsing of the type blob itself.

Both capabilities require tool blobs --- a PEG grammar for reference extraction,
a linter binary for validation, a linter config for rules. These tool blobs need
to be stored as objects in the repo, but there is no type to assign them. They
are not markdown, not TOML configs, not any existing type. They are raw content
whose meaning comes from the reference that points to them, not from their own
type definition.

## Design

### The Null Type: `!`

`!` is the null type --- the only type exempt from needing creation. It has no
blob, no definition, no validation. It is axiomatic: the fixed point of the type
recursion where every chain of "type of type of type" terminates.

Properties:

- **No blob.** `!` has no type blob and no object in the store. It is recognized
  by the type system as a hardcoded sentinel, like EOF.
- **No validation.** Objects of type `!` accept any blob content. The type
  imposes no schema.
- **No formatters.** No default rendering. Content is opaque bytes.
- **No reference discovery.** `!` declares no references. Objects typed `!` are
  leaf nodes in the reference graph.
- **No genesis entry.** `!` is not created during `init`. It exists prior to the
  store.

`!` enables bootstrap: tool blobs (grammars, binaries, configs) can be stored as
`!`-typed objects immediately, without defining `!peg-grammar` or `!executable`
first. As the type system matures, these objects graduate to richer types that
provide validation and formatting --- `!` becomes the staging ground.

#### Serialization

`!` round-trips through all formats but with format-appropriate rules:

- **Box:** always present --- `[yin/yang @digest ! tag1 tag2]`. Bare `!` in the
  type position, no trailing identifier.
- **Hyphence:** omitted --- when an object's type is `!`, the type line is not
  emitted. On read, a missing type line implies `!`. This keeps the user-facing
  format clean; users never type `! !`.
- **Binary index:** encoded normally as a zero-length identifier after the `!`
  genre byte. Internal format, must round-trip.
- **Inventory list:** `!`-typed objects may appear as entries within an
  inventory list (e.g., `[yin/yang @digest !]`). However, the inventory list
  object itself must NOT have type `!` --- it requires a traversable type
  definition so the system knows how to parse its contents.

#### Implementation Requirements

v2 does not implicitly support bare `!`. Three layers reject it today:

1.  **Box reader** (`hotel/box_format/read.go`): `TokenMatcherType` requires `!`
    followed by `TokenTypeIdentifier` (two tokens). Bare `!` is a single token
    and does not match.
2.  **Type ID parser** (`bravo/ids/type.go:Set`): trims `!` prefix, leaving an
    empty string, which fails `TagRegex` validation.
3.  **Type ID writer** (`bravo/ids/type.go:String`): empty `Value` field returns
    `""`, not `"!"`.

All three must be updated to recognize `!` as a valid type.

#### Versioning Impact

Introducing `!` as a valid type for objects within inventory lists requires an
inventory list type bump to v3. The v3 codec:

- **Reading:** accepts bare `!` as a type in box-formatted entries.
- **Writing:** emits bare `!` for null-typed objects.
- **v2 compatibility:** v2 readers do not know about `!` and will reject or
  misparse entries with a bare `!` type. Repos containing `!`-typed objects
  cannot downgrade to v2 inventory lists.

The box format writer is configuration-aware, not version-aware: inventory list
codecs configure the box writer with the appropriate options for their version.
The v2 configuration does not permit `!`-typed entries (error if encountered).
The v3 configuration permits bare `!` normally.

This follows the existing horizontal versioning pattern --- the v2 codec remains
for reading older inventory lists, v3 is registered alongside it, and the store
version determines which codec is used for new writes.

### Licensing

dodder is relicensed from MIT to GPL-3.0 to enable linking against langlang
(GPL-3.0) as a Go library. This is the simplest path --- dodder has a single
copyright holder and no downstream Go consumers of its `go/lib/` packages today.

GPL-3.0 does not restrict use, only distribution of modified binaries. Content
stored in dodder repos is not affected by the license. MCP servers and plugins
that call dodder as a subprocess are separate works and not subject to GPL.

### Progression

The intended lifecycle for a tool blob:

1.  **Phase 1 (now):** Store as `!`-typed object. The type blob lock in the
    referencing type (e.g., `!md`) is `!@blake2b256-abc...`.
2.  **Phase 2 (future):** Define a proper type (`!peg-grammar`,
    `!executable-tool`, `!lint-config`). Migrate the object's type. The lock
    becomes `!peg-grammar@blake2b256-abc...`.
3.  **Phase 3 (eventual):** A `dodder.net` seed repo provides these core types
    and their tool blobs as defaults. `init` clones from the seed repo. Users
    can override the seed repo to use their own provider.

Phase 3 is the target architecture for shipping tool blobs. Embedding binaries
in the dodder binary (\~4.5 MB for rumdl alone) is acceptable for Phase 1 but
does not scale. The seed repo model distributes tool blobs as regular objects
--- `init` pulls them from `dodder.net` (or a user-specified provider) the same
way it would pull from any remote.

### Markdown Reference Extraction

#### URI Scheme

Object references in markdown use `dodder://` URIs in standard markdown link
syntax:

- **Inline links:** `[link text](dodder://ceroplastes/midtown)`
- **Reference links:** `[text][ref]` with `[ref]: dodder://ceroplastes/midtown`

The URI scheme supports all addressable entities exposed through object locks:

  -------------------------------------------------------------------------------
  Entity     URI                              Example
  ---------- -------------------------------- -----------------------------------
  Zettel     `dodder://<yin>/<yang>`          `dodder://ceroplastes/midtown`

  Type       `dodder://!<type-id>`            `dodder://!md`

  Tag        `dodder://<tag-id>`              `dodder://priority-0_must`

  Blob       `dodder://<digest>`              `dodder://blake2b256-9ft3...`

  Typed blob `dodder://!<type-id>@<digest>`   `dodder://!md@blake2b256-9ft3...`
  -------------------------------------------------------------------------------

Disambiguation: zettel IDs always contain at least one `/` character. Tags never
contain `/`. This makes `dodder://ceroplastes/midtown` (zettel) unambiguous from
`dodder://priority-0_must` (tag). Bare identifiers without `/` are tags.
`konfig` is a reserved tag for repo-local mutable config.

#### PEG Grammar

A langlang PEG grammar extracts `dodder://` URIs from markdown link structures.
The grammar targets only link syntax, not full markdown parsing --- it skips
non-link content and collects URIs.

The grammar is stored as a `!`-typed blob in the repo. The `!md` type blob
references it via a typed blob lock.

At runtime, dodder loads the grammar blob and compiles it using langlang's
`Matcher` (bytecode VM). The `Matcher` executes against blob content and returns
a parse tree. Dodder walks the tree to extract `dodder://` URIs, then resolves
each URI to an object reference or blob reference for the metadata.

#### langlang Integration

dodder depends on langlang as a Go library
(`github.com/clarete/langlang/go/langlang`). The `Matcher` type compiles PEG
grammars to bytecode and executes them:

``` go
m := langlang.NewMatcher(grammarBytes)
tree, _, err := m.Match(blobContent)
// walk tree for dodder:// URIs
```

The `Matcher` is used at runtime (not `go generate`). This keeps the grammar as
pure data --- a blob, not compiled Go code. A future optimization can add
generated Go parsers as a fast path for builtin types while preserving the
runtime VM path for user-defined grammars.

### Markdown Validation

#### rumdl

Content validation uses rumdl, a Rust implementation of markdownlint (all 53
rules + 18 additional, \~4.5 MB static binary, \~60x faster than
markdownlint-cli2).

Three blobs are stored as `!`-typed objects at genesis:

1.  **rumdl binary** --- platform-specific static binary
2.  **rumdl config** --- `.markdownlint.json` rules
3.  **PEG grammar** --- langlang grammar for `dodder://` link extraction

The `!md` type blob references all three via typed blob locks.

At commit time, dodder resolves the blob digests, executes rumdl with the config
against the blob content, and fails the commit if validation errors are found.

#### Shipping

The rumdl binary and config are embedded in the dodder binary (or provided by
the Nix flake) and written into the blob store during genesis --- the same
pattern as the signing key. This is a Phase 1 expedient; the target architecture
is Phase 3 (seed repo), where tool blobs are distributed as regular objects from
`dodder.net` and never embedded in the binary.

### Type Blob Config Changes

The `[references]` section becomes a oneof semantic struct --- either a native
engine or a shell script, not both:

``` toml
file-extension = "md"
vim-syntax-type = "markdown"

# Native engine variant:
[references]
engine = "langlang-vm"
grammar = "!@blake2b256-abc..."

# OR script variant (existing behavior):
# [references]
# script = ["sh", "-c", "extract-refs.sh"]
# optional = true

[validation]
tool = "!@blake2b256-def..."
config = "!@blake2b256-ghi..."
```

The `engine` and `script` fields are mutually exclusive. When `engine` is
present, dodder uses the native mechanism. When `script` is present, dodder
shells out. Presence of both is a validation error in the type blob.

The `[validation]` section is new. It declares a tool and config for content
validation at commit time.

## Relationship to Existing Features

- **FDR-0001 (Object Locks):** The null type and tool blobs are referenced via
  typed blob locks from FDR-0001. This feature is a consumer of that
  infrastructure.
- **Reference discovery (`papa/store/reference_discovery.go`):** The native
  langlang engine is an alternative to the existing shell-script discovery path.
  Both produce the same output: object references and blob references added to
  metadata.

## Open Questions

- Should the engine field (`langlang-vm`) be a string enum or itself a typed
  blob reference pointing to the engine implementation?
- How should platform-specific binaries (rumdl per OS/arch) be handled? Multiple
  blobs with platform tags? A single blob with an embedded fat binary?
- What is the cache/invalidation strategy for compiled `Matcher` bytecode? Cache
  keyed on grammar blob digest seems natural.

## Related Issues

- [#46](https://github.com/amarbel-llc/dodder/issues/46) --- typed blob locks as
  tool/grammar/config references (origin of this FDR)
