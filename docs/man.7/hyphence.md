---
author:
-
date: April 2026
title: HYPHENCE(7) Dodder \| Miscellaneous
---

# NAME

hyphence - dodder object serialization format

# SYNOPSIS

    ---
    # description text
    - tag-name
    @ markl-id
    ! type-string
    ---

    body content

# DESCRIPTION

Hyphence (hyphen-fence) is a text-based serialization format that uses **---**
boundary lines to enclose a metadata section. It is the primary persistence and
interchange format for dodder objects: repository configs, blob store configs,
workspace configs, type definitions, and user-facing zettels all use hyphence.

# DOCUMENT STRUCTURE

A hyphence document has two parts:

1.  **Metadata section** (required): enclosed by two **---** boundary lines
2.  **Body section** (optional): content after the closing boundary

A minimal document:

    ---
    ! toml-type-v1
    ---

A document with inline body:

    ---
    # my description
    ! md
    ---

    Body content goes here.

A document with a blob reference (no inline body):

    ---
    # my description
    @ blake2b256-9ft3...
    ! md
    ---

A document must not have both a blob reference in the metadata and inline body
content after the closing boundary.

# BOUNDARY LINE

A boundary line is exactly three ASCII hyphen-minus characters followed by a
newline:

    ---\n

No trailing spaces, carriage returns, or additional hyphens.

# BODY SEPARATOR

When a body follows the metadata section, there must be exactly one empty line
between the closing **---** and the start of the body:

    ---
    ! md
    ---
                    <-- required blank line
    Body starts here.

Without this blank line, the body content is silently dropped during parsing.

# METADATA LINES

Each metadata line begins with a single-character prefix, a space, and the
content:

**! type-string**
:   Object type identifier. Determines how the body is decoded. May include a
    lock: **! type-string@markl-id**. Should be the last non-comment line.

**@ markl-id** or **@ file-path**
:   Blob reference. Alternative to inline body --- references content by
    digest or file path.

**# text**
:   Description. Multiple description lines are space-concatenated.

**- identifier**
:   Tag (bare name like **todo**) or object reference (contains **/**).
    References may include locks: **- object-id < markl-id**

**< object-id**
:   Explicit object reference. Same syntax as **-** references.

**% text**
:   Comment. Opaque, preserved during round-trips. Each comment is entangled
    with the non-comment line that follows it.

# CANONICAL LINE ORDER

When encoding, metadata lines should follow this order:

1.  Description lines (**#**)
2.  Locked object references
3.  Aliased object references
4.  Bare object references
5.  Tags (**-**)
6.  Blob line (**@**)
7.  Type line (**!**)

Lines may appear in any order during decoding --- order does not affect
semantics.

# VERSIONING

Hyphence has no version indicator. Format evolution uses type strings: new
versions introduce new type strings (e.g. **toml-blob_store_config-v2**
succeeds **-v1**) while old type strings retain their decoders. Old versions
remain decodable indefinitely.

# EXAMPLES

A zettel with tags:

    ---
    # purchase izipizi glasses
    - area-home
    - todo
    - urgency-2_week
    ! task
    ---

A blob store configuration:

    ---
    ! toml-blob_store_config-v3
    ---

    [blob-store]
    compression-type = "zstd"
    lock-internal-files = true

A type definition with a lock:

    ---
    @ blake2b256-76m5...
    ! toml-type-v1@ed25519_sig-1qxyz...
    ---

# SEE ALSO

**markl-id**(7), **organize-text**(7), **blob-store**(7)
