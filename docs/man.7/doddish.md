---
author:
- 
date: April 2026
title: DODDISH(7) Dodder \| Miscellaneous
---

# NAME

doddish - dodder query language

# SYNOPSIS

*predicate*\[*sigil*\]\[*genre*\]

# DESCRIPTION

Doddish is the query language used by dodder commands that accept object
predicates (**show**, **checkout**, **status**, **edit**, **push**, **pull**,
etc.). Query terms are separated by spaces and combined with AND semantics ---
an object must match all terms to be included in the result. The exception is
object IDs: because IDs are unique, multiple object ID terms are implicitly
OR-combined.

# QUERY TERMS

A query term has up to three parts:

*predicate*\[*sigil*\]\[*genre*\]

The predicate identifies what to match. The sigil selects which object set to
search. The genre restricts which object kinds to return.

## Predicates

A predicate can be:

**tag name**
:   A bare identifier matches objects tagged with that tag. It matches on the
    tag *string* carried by each object, whether or not a tag *object* of that
    name has been materialized. Example: **todo**, **priority-0_must**

**type filter**
:   A **!** prefix matches objects whose type is the given type. Example:
    **!md**, **!task**

**object ID**
:   A zettel ID (two parts separated by **/**). Example: **ceroplastes/midtown**

**empty**
:   When the predicate is empty, only the sigil and genre apply. Example: **:**
    lists all latest zettels.

## Sigils

Sigils control which object set is queried:

**:**
:   Latest version of objects in the store (default).

**+**
:   Historical versions (all inventory list entries).

**.**
:   External / checked-out objects in the workspace.

**?**
:   Hidden / dormant objects.

Sigils can be combined. For example, **:.** selects latest objects that are also
checked out.

## Genre Suffixes

Genre suffixes restrict results to a specific object kind:

**z**
:   Zettels (default when no genre is specified).

**t**
:   Types.

**e**
:   Tags (etiketten).

**b**
:   Inventory lists (bestandsaufnahme).

**k**
:   Repos (kasten).

A genre letter is matched as a prefix of the genre's full name, so any
unambiguous prefix selects the same genre. Repos accept two names, **kasten**
and **repo**, so **k**, **kasten**, **r**, and **repo** all resolve to the Repo
genre; **z** and **zettel** both resolve to zettels. The single letters above
are the canonical forms --- prefer them. The Blob genre has no short letter and
must be spelled **blob** in full.

Genres can be combined with commas: **:z,t,e** matches zettels, types, and tags.

The genre suffix is part of the query term, not a separate argument.

## Quoted Literals

Field values and other literal text can be double-quoted to include
spaces or reserved characters, e.g. **url="https://example.com/a b"**.
Inside a quoted literal, a backslash escapes the following character:
**\\n**, **\\t**, **\\r**, **\\a**, **\\b**, **\\f**, and **\\v** map to
their standard control characters (newline, tab, carriage return, bell,
backspace, form feed, vertical tab); **\\"** and **\\\\** round-trip as a
literal quote and a literal backslash. Any other character following a
backslash is passed through unchanged.

# DEFAULTS

When no sigil is given, **:** (latest) is assumed. When no genre is given, **z**
(zettels) is assumed. Therefore, a bare tag name like **todo** is equivalent to
**todo:z**.

# GROUPING

Terms can be grouped with brackets:

**\[!md,home\]:z**
:   Objects matching type !md OR tag home, restricted to zettels.

The **\^** operator negates a group or term:

**\^todo**
:   Objects NOT tagged todo.

# EXAMPLES

## Listing by Genre

**:**
:   List all latest zettels (default genre).

**:t**
:   List all type objects.

**:e**
:   List all tag objects. Only *materialized* tag objects are listed: a tag
    applied to objects purely as a string does not appear until it is authored
    as a tag object (**new -object-id <tag>** or **organize**). Materialization
    also gates meta-tags --- a tag carries meta-tags, and participates in
    meta-tag laddering, only once it exists as an object. Matching objects by a
    bare tag name (above) works either way.

**:k**
:   List all repo objects.

**:z,t,e**
:   All latest zettels, types, and tags.

## Filtering by Tag and Type

**todo**
:   Zettels tagged **todo**.

**!md**
:   Zettels whose type is **!md** (NOT the !md type object itself).

**!md:t**
:   The **!md** type object (genre suffix **:t** selects types).

**priority-0_must !task**
:   Tasks tagged **priority-0_must** (AND combination).

**=!md**
:   Exact type match (not prefix match).

## Sigil Combinations

**:.z**
:   Latest zettels that are checked out.

**!md?z**
:   Hidden/dormant zettels of type **!md**.

**one/uno+**
:   History of a specific zettel (all versions).

**one/uno.zettel**
:   A specific zettel's checked-out (external) version.

## Object IDs

**ceroplastes/midtown**
:   A specific zettel by ID.

**one/uno one/dos**
:   Multiple zettels (object IDs are OR-combined, unlike other terms).

**/repo:k**
:   A specific repo by ID.

## Negation

**\^todo**
:   Objects NOT tagged **todo**.

**\^\[test,house\] home**
:   Objects matching **home** but NOT matching **test** OR **house**.

**\[\^house,test\] home**
:   Objects matching **home** and either (NOT **house**) or **test**.

## Grouping

**\[!md,home\]:z**
:   Objects matching type **!md** OR tag **home**, restricted to zettels.

**\[test,house\] home wow**
:   Objects matching (**test** OR **house**) AND **home** AND **wow**.

## Dependent Tags

**-etikett-two.z**
:   Checked-out zettels with dependent tag **-etikett-two**.

# COMMON MISTAKES

To query a specific type object, the genre suffix goes on the predicate:

    dodder show '!img:t'     # correct: the !img type object
    dodder show :t '!img'    # wrong: two separate terms

The default genre is zettels, so **!md** finds zettels of type !md, not the type
object itself. Use **!md:t** to get the type object.

# SEE ALSO

**dodder**(1), **dodder-show**(1), **dodder-checkout**(1), **markl-id**(7)
