---
author:
- 
date: April 2026
title: ORGANIZE-TEXT(7) Dodder \| Miscellaneous
---

# NAME

organize-text - dodder organize text format

# SYNOPSIS

The organize text format is a structured text representation used by the
**organize** command to batch-edit object metadata (tags, types, descriptions,
fields) in a text editor or via stdin.

# DESCRIPTION

When **dodder organize** is invoked, it generates a text document listing
objects grouped under headings. The user (or a script) edits this document to
add, remove, or reassign tags, change descriptions, or edit field values. The
edited document is then parsed back, and changes are committed to the store.

The format has three components: **metadata** (document-level type and tags),
**headings** (hierarchical tag groups), and **object lines** (individual objects
in box format).

# DOCUMENT STRUCTURE

An organize text document begins with optional metadata in hyphence format,
followed by object lines and headings:

    ---
    ! task
    ---

    # project-alpha

    - [one/uno !task status=todo] fix the login bug
    - [one/dos !task status=done] update dependencies

    # project-beta

    - [two/uno !task status=todo] review PR

## Metadata

The document metadata specifies a default type and tags applied to all objects.
It uses the hyphence format (triple-hyphen delimiters):

    ---
    ! task
    - project-alpha
    ---

The type (`! task`) sets the default type for new objects. Tags
(`- project-alpha`) are applied to all objects in the document.

## Settings Fields

Document-level, behavior-affecting settings are spelled as `_`-reserved
metadata fields (cutting-garden RFC 0015), not comments — a leading
underscore marks a field as document/operation-scoped rather than a tag or
type applied to every object in the document:

    ---
    - _dry-run=true
    ---

**`_dry-run=true`** mirrors the **-dry-run** CLI flag: when active,
**organize** parses and validates changes without writing them to the
store. It is generated in output whenever **-dry-run** is passed on the
command line, and can be set explicitly in a hand-edited document, though
note that today this only takes effect when **organize** was already
invoked with **-dry-run** on the command line (the field is not yet a way
to activate dry-run mode from a cold start).

The older comment spelling, **`% dry-run:true`**, is still accepted when
reading a document (a deprecated alias), but is no longer generated.

## Headings

Headings use markdown-style `#` syntax. Each heading level defines a tag scope:

    # tag-a
    - [one/uno !md] first

    ## tag-b
    - [one/dos !md] second

Objects under `## tag-b` inherit both `tag-a` and `tag-b`. Heading depth
corresponds to the number of `#` characters.

Multiple tags on a heading are space-separated, forming a conjunction (a
ground trellis term — cutting-garden RFC 0015): every object under the
heading carries all of the listed tags.

    # tag-a tag-b

A comma in a heading is a parse error — comma is disjunctive elsewhere in
the query grammar (`doddish`(7)), so a comma-separated heading would invert
the intended AND semantics. There is no legacy acceptance of the older
comma-separated spelling.

## Object Lines

Each object line starts with a prefix indicating its type assignment:

**`-`** (direct type)
:   The object has a known, directly assigned type.

**`%`** (unknown type)
:   The object's type is inferred or virtual.

After the prefix, the object is rendered in box format (see **BOX FORMAT**
below):

    - [object-id !type tag1 tag2 field=value] description

# BOX FORMAT

Object lines use the box format within square brackets. Fields appear in order:

1.  **Object ID** --- e.g., `one/uno`, `!md`, `konfig`
2.  **Blob digest** --- prefixed with `@` (e.g., `@blake2b256-...`), present
    when the object has blob content
3.  **Type** --- prefixed with `!` (e.g., `!task`, `!md`)
4.  **Tags** --- bare identifiers, sorted alphabetically
5.  **Fields** --- `key=value` pairs for type-defined fields (e.g.,
    `status=todo`)

The description appears after the closing bracket as a trailer.

Field values are unquoted when they contain only identifier-safe characters.
Values with spaces or reserved characters are Go-quoted
(`key="value with spaces"`).

# EDITING

## Adding Tags

Move an object under a heading, or add a tag to the heading line:

    # new-tag
    - [one/uno !md] my zettel

## Removing Tags

Remove a tag from a heading or move the object out from under it.

## Changing Descriptions

Edit the description text after the closing bracket:

    - [one/uno !md] updated description here

## Changing Fields

Edit the field value within the brackets:

    - [one/uno !task status=done] my task

When a field value is changed, the **fields-writer** script (defined in the type
blob) is invoked to update the blob content. The blob digest changes
accordingly.

## Creating Objects

Add a new object line. The organize system will create it on commit:

    - new object description

## Removing Objects

Delete the object line from the document. This is **not** a no-op and does
**not** delete the object from the store — it is a real metadata write: the
tag(s) belonging to the query used to invoke **organize** (the document
metadata's tag set, see **METADATA** above) are removed from the object, and
the reduced tag set is committed as a new object version. For example, after
`dodder organize tag-5`, deleting an object's line removes `tag-5` from that
object; the object no longer matches that query on a subsequent `show`.

Tags implied by headings or **-group-by** grouping are a separate mechanism
(see **HEADINGS**) and are **not** affected by line deletion — an object that
was nested under `# priority-1` keeps the `priority-1` tag even after its
line is deleted, because only the document-level query-selection tags are
removed, not the per-heading tag set.

This is scoped to the tags declared at the document-metadata level; it says
nothing about the grouped (**-group-by**) dimension specifically, and should
not be read as "clears the object's group membership" or "removes it from
dodder entirely."

# INTERNAL AND EXTERNAL FORKS

When organize computes changes, each object exists as two forks:

**Internal fork**
:   The object as stored in the index. Carries full metadata including
    **TypeBlobDigest** on fields, blob digest, type signature, and other
    computed values.

**External fork**
:   The object as parsed from the organize text. Carries user-edited values
    (description, tags, field values) but lacks computed metadata like
    TypeBlobDigest.

During change resolution, the external fork's edited values are overlaid onto
the internal fork's structure. For fields, this means the internal fork provides
the field definitions (TypeBlobDigest) while the external fork provides the
edited values.

# MODES

The **organize** command supports three modes via the **-mode** flag:

**interactive** (default)
:   Generate organize text, open in editor, commit on save.

**commit-directly**
:   Generate organize text internally, read replacement from stdin, commit. Used
    for scripted workflows.

**output-only**
:   Generate organize text and write to stdout. No changes are committed.

# SEE ALSO

**dodder-organize**(1), **doddish**(7), **markl-id**(7)
