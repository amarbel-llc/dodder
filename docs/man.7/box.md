---
author:
-
date: April 2026
title: BOX(7) Dodder \| Miscellaneous
---

# NAME

box - dodder compact object representation format

# SYNOPSIS

\[*object-id* @*blob-digest* !*type* *tag1* *tag2* ...\] *description*

# DESCRIPTION

The box format is dodder's default output representation for objects. Each
object is rendered as a single line with metadata fields inside square brackets
and an optional description trailer. It is used by **dodder show** (default
format), **dodder status**, organize-text documents (see **organize-text**(7)),
and MCP resource listings.

# FIELD ORDER

Fields appear inside the brackets in this fixed order:

1.  **Object ID** --- the identifier. Zettels use **left/right** form (e.g.
    **thallium/golem**), types are **!**-prefixed (e.g. **!md**), tags are bare
    names (e.g. **todo**).

2.  **Blob digest** --- prefixed with **@** (e.g.
    **@blake2b256-9ft3...**). Omitted when the object has no blob.

3.  **Timestamp** --- TAI date. Only present when the **-print-time** flag is
    used.

4.  **Type** --- prefixed with **!** (e.g. **!task**, **!toml-type-v1**).
    Omitted when the object has no type.

5.  **Tags** --- bare identifiers sorted alphabetically. Tags prefixed with
    **%** are computed or derived at display time and are not persisted to the
    store.

6.  **Fields** --- optional **key=value** pairs.

**Description** appears as a trailer after the closing bracket, separated by a
space.

# QUOTING

Values containing spaces are Go-quoted with double quotes:

    [one/uno !md "wow the first" tag-3 tag-4]

Identifier-safe values (alphanumerics, hyphens, underscores) are unquoted.
Embedded quotes within quoted values use Go escape syntax:

    [one/uno !md "see these \"quotes\""]

The description trailer is never quoted.

# COMPUTED TAGS

Tags prefixed with **%** are not stored in the repository. They are generated
at display time by the object's type or other entities. For example:

    [!md %virtual_etikett new-etikett-for-all]

The **%** prefix is output-only and is not used in query syntax. To filter by a
computed tag, use the bare tag name in a doddish query.

# DESCRIPTION PLACEMENT

Two variants exist:

**Outside brackets (default)**
:   Description appears after the closing bracket. Used by CLI output and MCP
    responses.

        [thallium/golem !task todo] purchase izipizi glasses

**Inside brackets**
:   Description appears as the last field inside brackets. Used by archive
    representations.

# FORMAT VARIANTS

**Standard**
:   Color output with optional timestamps. Used by **dodder show**.

**No-color**
:   No ANSI escape codes, no timestamps. Used by MCP responses and programmatic
    access.

**Archive**
:   No color, timestamps enabled, description inside brackets.

# EXAMPLES

A zettel with type, tags, and description:

    [thallium/golem !task area-home urgency-2_week] purchase izipizi glasses

A zettel with blob digest:

    [ceroplastes/midtown @blake2b256-9ft3... !md project-2024-q3] meeting notes

A type object:

    [!md @blake2b256-76m5... !toml-type-v1]

A tag object with description:

    [project-2021-zit area-career] store, organize, query, and edit files

A minimal tag (no type, no tags, no description):

    [zz-inbox]

# SEE ALSO

**dodder-show**(1), **organize-text**(7), **doddish**(7)
