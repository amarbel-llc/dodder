---
status: proposed
date: 2026-03-14
---

# Hyphence Serialization Format

## Abstract

Hyphence (hyphen-fence) is a text-based serialization format that uses `---`
boundary lines to enclose a metadata section. The metadata section MUST be
present and contains structured lines describing the object: its type, blob
reference, description, object references, and comments. The body content is
OPTIONAL and can be provided either inline (after the closing boundary) or by
reference (via a blob line in the metadata section). Hyphence is the primary
persistence and interchange format for dodder objects.

## Introduction

Dodder stores versioned, typed data across multiple domains: repository
configurations, blob store configurations, workspace configs, genesis configs,
inventory list entries, and user-facing zettels. Each domain evolves
independently with its own version history, but all share the same structural
need: a way to describe an object's metadata (type, references, description) and
associate it with blob content.

Hyphence provides this by defining a structured metadata format enclosed by
`---` boundary lines (the "hyphen fence"). The metadata section contains
prefixed lines that describe the object. Body content may follow the metadata
section inline, or may be referenced by digest or file path within the metadata
itself.

The `hyphence` Go package also handles opaque documents (raw content without
metadata). This RFC specifies only the hyphence document format; opaque document
handling is an implementation detail of the package.

A hyphence-like format is also used by the organize-text system, which reuses
the same boundary and line-prefix syntax but with different semantics: the type
line defines the type for all contained objects, and reference lines define tags
common to all contained objects. This RFC does not specify the organize-text
variant.

The intent of the hyphence format is to provide a self-contained,
human-readable, and human-writable representation of an object that defines how
to validate, format, and syntax-highlight the body content.

This RFC specifies the on-disk format, metadata line types, body provision
mechanisms, encoding sequence, and decoding sequence.

## Requirements Language

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be
interpreted as described in RFC 2119.

## Specification

### Document Structure

A hyphence document consists of:

1. **Metadata section** (REQUIRED): enclosed by two boundary lines
2. **Body section** (OPTIONAL): inline content after the closing boundary, or
   referenced via a blob line in the metadata

A minimal document (metadata only, no body):

```
---
! <type-string>
---
```

A document with inline body:

```
---
# my description
! <type-string>
---

<body-content>
```

A document with a blob reference (no inline body):

```
---
# my description
@ <markl-id-or-path>
! <type-string>
---
```

### Boundary Line

A boundary line MUST consist of exactly three ASCII hyphen-minus characters
(`U+002D`) followed by a single newline character (`U+000A`):

```
---\n
```

A boundary line MUST NOT contain any other characters. Trailing spaces, carriage
returns, or additional hyphens invalidate the boundary.

### Metadata Section

The metadata section is the content between the opening and closing boundary
lines. It MUST be present in every hyphence document. It contains zero or more
metadata lines, each terminated by a newline. Empty lines within the metadata
section MUST be ignored.

Each metadata line begins with a single-character prefix that determines the
line type, followed by a space and the line content:

| Prefix | Name | Description |
|--------|------|-------------|
| `!` | Type | Object type identifier, optionally locked |
| `@` | Blob | Blob reference (markl ID or file path) |
| `#` | Description | Human-readable description text |
| `-` | Tag or Reference | Tag identifier or object reference |
| `<` | Reference | Object reference (explicit prefix) |
| `%` | Comment | Opaque implementation-specific comment |

#### Type Line (`!`)

The type line identifies the object's type. A metadata section SHOULD contain at
most one type line.

```
! <type-string>
```

The type string identifies the versioned format of the body content. Type
strings follow the convention `<format>-<domain>-<version>`, for example:

- `toml-blob_store_config-v2`
- `toml-repo_config-v1`
- `gob-repo_config-v0`

The type string MUST NOT contain spaces. The type string MUST NOT be empty.

A type line MAY include a lock (a markl ID that pins the type to a specific
version of its definition):

```
! <type-string>@<markl-id>
```

The `@` character is part of the markl ID format and separates the type string
from its lock value.

#### Blob Line (`@`)

The blob line references the object's body content within the metadata section,
as an alternative to providing the body inline after the closing boundary.

A blob line MUST contain either a markl ID (content digest) or a file path:

```
@ <markl-id>
@ <file-path>
```

When the value is a file path, both absolute and relative paths are accepted.
When the value is a markl ID, it identifies the blob by its content-addressable
digest.

A document MUST NOT contain both a blob line in the metadata section and inline
body content after the closing boundary. Implementations MUST return an error if
both are present.

#### Description Line (`#`)

Description lines provide human-readable text describing the object:

```
# <description-text>
```

A metadata section MAY contain multiple description lines. When more than one
description line is present, the description is the space-concatenation of all
description lines (in order of appearance).

The use of the term "description" is intentional — descriptions are NOT titles,
names, or identifiers. Descriptions MUST NOT be used to identify objects.
Object identity is established by the object ID, which is immutable and
content-addressable. This is a deliberate solution to the curse of the
`$PATH`/filename problem, where identity is the file name — mutable, fragile,
and collision-prone.

Descriptions MAY appear in the box format when objects appear in the contents of
another object. Descriptions embedded in containing objects MAY become stale;
when objects are reformatted, implementations SHOULD update embedded
descriptions to reflect the current description of the referenced object.

#### Tag and Reference Line (`-`)

The `-` prefix is used for both tags and object references. The two are
distinguished by content: values containing a path separator (`/`) are parsed as
object references; simple identifiers are parsed as tags.

**Tags:**

```
- <tag-identifier>
```

**Object references** can appear in several forms:

Bare reference (just the object ID):

```
- <object-id>
```

Locked reference (object ID followed by `<`, space, and lock markl ID):

```
- <object-id> < <markl-id>
```

Aliased and locked reference (alias, space, `<`, space, locked object ID):

```
- <alias> < <object-id>@<markl-id>
```

Object locks are markl IDs that pin the reference to a specific version of the
referenced object.

#### Reference Line (`<`)

The `<` prefix is an explicit alternative to `-` for object references. It uses
the same reference syntax as described above (bare, locked, or aliased).

```
< <object-id>
< <object-id> < <markl-id>
< <alias> < <object-id>@<markl-id>
```

#### Comment Line (`%`)

Comment lines are opaque and implementation-specific. Their content is preserved
during round-trips but their semantics are not specified by this RFC.

```
% <comment-text>
```

Implementations MAY use comment lines for internal bookkeeping. Implementations
MUST preserve comment lines during encoding if they were present during
decoding.

### Body Section

The body section is OPTIONAL. When present, it begins after the closing boundary
and a separator line.

When a body follows the metadata section, there MUST be exactly one empty line
(a single `\n`) between the closing boundary and the start of the body content:

```
---\n
<metadata-lines>\n
---\n
\n
<body-content>
```

This empty line is a separator and is NOT part of the body content. The body
extends to the end of the input stream.

The body format is determined by the type string in the metadata section. Common
body formats include TOML and Gob, but the hyphence format itself is agnostic to
the body encoding.

Body content can alternatively be provided via a blob line (`@`) in the metadata
section. See "Blob Line" above.

### Encoding Sequence

An encoder MUST produce output in this order:

1. Write the opening boundary: `---\n`
2. Write metadata lines (description, blob, type, references, comments)
3. Write the closing boundary: `---\n`
4. If inline body content follows:
   a. Write the separator: `\n`
   b. Write body content

If no body content follows (either because there is none or because it is
referenced via a blob line), the encoder MUST NOT write the separator line.

### Decoding Sequence

A decoder MUST process input as follows:

1. Read the opening boundary line
2. Read metadata lines until a second boundary line is encountered
3. Read and discard the closing boundary
4. If content remains after the closing boundary:
   a. Read and discard the separator line
   b. Decode body content
5. If a blob line was present in the metadata, resolve the blob reference

A document MUST NOT have both inline body content and a blob line. If both are
present, the decoder MUST return an error.

### Type-Dispatched Decoding

After extracting the type string from the metadata section, the decoder MUST
look up the type string in a coder map to find the appropriate version-specific
decoder. If no decoder is registered for the type string, the decoder MUST
return an error indicating the unrecognized type.

This mechanism enables horizontal versioning: multiple versions of the same
domain can coexist, with older versions remaining decodable even after newer
versions are introduced. When encoding, implementations SHOULD use the current
(latest) version's type string and encoder.

### Metadata Line Ordering

Metadata lines MAY appear in any order within the metadata section. A document
that does not follow the canonical sort order is still valid. Implementations
MUST NOT reject or alter the semantics of a document based on line ordering.

However, when encoding (formatting) a metadata section, implementations SHOULD
write lines in canonical order:

1. Description lines (`#`)
2. Locked object references (references with a `<` lock)
3. Aliased object references (references with an alias)
4. Bare object references (references without lock or alias)
5. Tags
6. Blob line (`@`)
7. Type line (`!`) — MUST be the last non-comment line

#### Comment Entanglement

Comment lines (`%`) MAY appear throughout the metadata section. Each comment
line is "entangled" with its following non-comment line: the comment describes
or annotates the line that immediately follows it. When metadata lines are
sorted into canonical order, each comment MUST remain immediately before its
entangled line. A comment line at the end of the metadata section (with no
following non-comment line) is entangled with the closing boundary.

#### Description Line Stability

Description lines MUST preserve their relative ordering with respect to each
other. The description block as a whole MAY be moved to satisfy the canonical
sort (e.g., moved to the top), but the individual description lines within the
block MUST NOT be reordered.

### Identity Model

Object IDs MAY collide across repositories. However, repository public keys
MUST NOT collide, and signatures MUST NOT collide. The combination of repository
identity (public key) and object ID provides globally unique identification.

### Future: Fully-Qualified Object References

Two concepts not yet introduced into the object reference and type definition
system are **domains** and **repo IDs**. The intent is for a fully-qualified
object reference to take the form:

```
<domain>/<repo-id>/<object-id>@<sig_type>-<blech32>
```

For example:

```
example.com/my-repo/ceroplastes/midtown@ed25519_sig-1qxyz...
```

This would allow universally unambiguous references in metadata sections,
eliminating the need for out-of-band context to resolve object references across
repository boundaries. This extension is not yet specified and will be the
subject of a future RFC.

## Security Considerations

Hyphence is a framing format with no built-in authentication or integrity
checking. Content integrity and authenticity are handled at a higher layer by
dodder's content-addressable storage (markl IDs provide hash-based integrity)
and signature system (ed25519 signatures provide authenticity).

Implementations MUST validate that the type string maps to a registered decoder
before attempting to decode body content. Processing body content with an
incorrect decoder could lead to undefined behavior.

Blob lines referencing file paths introduce a file-system access vector.
Implementations MUST validate that file paths resolve within expected boundaries
and SHOULD NOT follow symbolic links outside the repository.

The format itself does not impose size limits. Implementations SHOULD enforce
reasonable limits on metadata section size to prevent resource exhaustion from
malformed input.

## Compatibility

### Versioning Strategy

The hyphence format itself has no version indicator. Format evolution is handled
through type strings: new versions of a domain introduce new type strings (e.g.,
`toml-blob_store_config-v2` succeeds `toml-blob_store_config-v1`) while old type
strings retain their registered decoders.

Implementations MUST NOT remove decoders for old type strings when adding new
versions. This ensures that data written by older versions remains readable.

### YAML Front Matter

The `---` boundary syntax intentionally mirrors YAML front matter conventions
used in Markdown and other formats. However, hyphence metadata is NOT YAML. The
metadata section uses a line-prefix format. Implementations MUST NOT parse the
metadata section as YAML.

### Legacy `!` Blob Paths

Older versions of the format used the `!` prefix for blob file paths (when the
value contained `/`). Implementations SHOULD accept `!` lines containing `/` as
blob references for backward compatibility, but MUST use `@` for new output.

## References

### Normative

- [RFC 2119] Bradner, S., "Key words for use in RFCs to Indicate Requirement
  Levels", BCP 14, RFC 2119, March 1997.
- [markl ID format] (RFC pending) Content-addressable identifier format used for
  blob digests, type locks, and reference locks.

### Informative

- dodder horizontal versioning pattern (`design_patterns-horizontal_versioning`
  skill)
- dodder content-addressable identifiers (`design_patterns-markl_id` skill)
