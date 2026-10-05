---
author:
-
date: October 2026
title: TRANSFORM-SCRIPT(7) Dodder \| Miscellaneous
---

# NAME

transform-script - dodder Lua inventory-list transform script API

# SYNOPSIS

    local list = dodder.list()        -- or dodder.list_v2()

    for object in list:each() do
      ...
    end

    return list

# DESCRIPTION

A transform script is a Lua chunk run by **dodder transform** and **dodder
init-from-lists**. It receives the selected objects as a list, mutates the
objects and the list's membership in place, and returns the list. Dodder then
commits the returned objects.

The chunk runs exactly once per invocation. Under a dry run no objects are
committed, but the script itself still runs.

# THE LIST HANDLE

The **dodder** global offers two bindings onto the same input objects. They
differ only in the key names of the per-object tables (see **OBJECT KEYS**).

**dodder.list()**
:   The default binding. Object tables carry the original German key names.

**dodder.list_v2()**
:   The opt-in binding. Object tables carry English key names.

Every call to one binding returns the same handle. A script must use exactly
one of the two and **return** that binding's handle: a script that calls both,
or returns anything else, is rejected and nothing is committed.

The handle has three methods:

**list:each()**
:   Returns an iterator over the object tables, for use as **for object in
    list:each() do**. Removed objects are skipped; objects added during
    iteration are visited.

**list:remove(***object***)**
:   Drops *object* from the output. The object gets no new revision; nothing is
    deleted from the store. Passing a table that is not an object from this
    list raises a Lua error.

**list:add()**
:   Creates a new zettel that was not in the input and returns its object
    table. Its object id is empty; an id is allocated when the output is
    committed.

# OBJECT KEYS

Each object is a Lua table. These keys differ by binding:

  **dodder.list()**     **dodder.list_v2()**   Meaning
  --------------------- ---------------------- --------------------------------
  **Gattung**           **Genre**              genre, e.g. **Zettel**
  **Kennung**           **ObjectId**           object id
  **Typ**               **Type**               type
  **Etiketten**         **Tags**               tags, a table of name to true
  **EtikettenImplicit** **TagsImplicit**       implicit tags, read-only
  **Bezeichnung**       **Description**        description

These keys have the same name in both bindings:

**Fields**
:   The object's projected fields, a table of name to string value. Assigning
    to an existing field rewrites it. A key that is not already a field of the
    object is ignored.

**Blob**
:   The blob digest as a markl id string. Assign the result of **blobs.write**
    to point the object at new content. An empty string clears the digest.

**References**
:   An array of object id strings the object references. Appending an id adds a
    reference; removing a reference is not expressible.

**SkipHooks**
:   Boolean, **false** by default. Set to **true** to commit this one object
    without its Lua hook stages. Reference discovery, fields projection, and
    field validation still run. The command reports how many objects skipped
    their hooks.

**Tai**, **TaiSortKey**, **TaiDate**
:   Read-only. The object's timestamp as the canonical tai string, as a
    fixed-width key that sorts lexicographically in time order, and as a local
    calendar date (**YYYY-MM-DD**). Assignments to these are ignored.

# MUTATION RULES

Assigning to an object table mutates the object in place.

To remove a tag, assign **nil** or **false** to its entry in the tags table.
To add one, assign **true**.

Assigning **nil** or an empty string to the genre, object id, or type key
leaves that part of the object unchanged. Clearing an object's type, or
blanking its id to request a new one, is not expressible; use **list:add()**
to create an object that needs an id.

Assigning to the description key replaces the description.

An output that names the same object id more than once is rejected.

# THE BLOBS GLOBAL

**blobs.read(***digest***)**
:   Returns the content of the blob with the given markl id as a Lua string.

**blobs.write(***bytes***)**
:   Stores *bytes* as a blob and returns its digest as a markl id string.

# EXAMPLES

Retype legacy tasks and drop a dead tag, with the default binding:

    local list = dodder.list()

    for object in list:each() do
      if object.Typ == "!task-legacy" then
        object.Typ = "!task"
      end

      object.Etiketten["newsblur"] = nil
    end

    return list

The same script with the English-keyed binding:

    local list = dodder.list_v2()

    for object in list:each() do
      if object.Type == "!task-legacy" then
        object.Type = "!task"
      end

      object.Tags["newsblur"] = nil
    end

    return list

Add a zettel that references an existing one and commits without hooks:

    local list = dodder.list_v2()
    local note

    for object in list:each() do
      if object.ObjectId == "one/uno" then
        note = object
      end
    end

    local task = list:add()
    task.Type = "task"
    task.Description = note.Description
    task.References[#task.References + 1] = note.ObjectId
    task.SkipHooks = true

    return list

# SEE ALSO

**dodder-transform**(1), **dodder-init-from-lists**(1), **markl-id**(7),
**doddish**(7)
