-----

## status: exploring
date: 2026-03-20
promotion-criteria:

# Pluggable Checkout Stores

## Problem Statement

Workspace-repos (FDR-0005) decouple a workspace’s commit history from its
parent, but the checkout medium is still hardcoded to `store_fs` — files on
disk. Many object types have a natural home that isn’t the filesystem. Tasks
belong in a CalDAV server (tasks.org, Apple Reminders, Thunderbird). Bookmarks
belong in a browser. Notes might belong in a WebDAV folder or a wiki. Calendar
events belong in a CalDAV VEVENT collection.

Today, to interact with dodder tasks through a CalDAV client, you’d need an
external sync script that understands both dodder’s object model and the
iCalendar format. The mapping between a dodder zettel of type `!task` and a
VTODO is type-specific, bidirectional, and must survive round-trips through
external clients that add their own properties (tasks.org adds
`X-APPLE-SORT-ORDER`, Thunderbird adds `X-MOZ-GENERATION`, etc.).

There is no mechanism for a workspace to declare “my checkout medium is CalDAV”
and have checkout/checkin translate between dodder’s internal representation and
the external format, detect changes made by external clients, and reconcile
conflicts when both sides have mutated an object since the last sync.

## Design

### The Compilation Model

Checkout is compilation. Checkin is decompilation. The dodder object graph is the
intermediate representation (IR); the external format (iCalendar, HTML, WebDAV
properties, git objects) is the compilation target.

The type definition is the domain model — a `!task` object isn’t “a dodder object
that happens to map to a VTODO.” It *is* a task, expressed in dodder’s native
idiom. Fields like `summary`, `status`, `priority`, `due`, `rrule` are on the
type not because CalDAV demands them but because they are the semantically
correct fields for modeling tasks. The domain alignment between the dodder type
and the external format is what makes the transform stable, not clever adapter
code.

The workspace config declares which compilation target to use and how fields map
between dodder’s representation and the target format. The type defines what the
object *is*; the workspace defines how it’s *compiled*.

### Object Graph Decomposition

A VTODO is not atomic. It contains sub-structures: VALARMs, ATTACHments,
RELATED-TO references, ATTENDEEs. Dodder’s referenced objects and blob references
(FDR-0001) can decompose these into a normalized object graph:

|iCalendar construct           |Dodder representation                                                                                       |
|------------------------------|------------------------------------------------------------------------------------------------------------|
|VTODO itself                  |`!task` object (summary, status, priority, due, etc. as type-defined fields; blob is the DESCRIPTION body)  |
|VALARM                        |referenced `!alarm` object (trigger, action, description)                                                   |
|ATTACH                        |blob reference (`- attachment-name < @blake2b256-...`)                                                      |
|RELATED-TO;RELTYPE=PARENT     |referenced `!task` object (subtask relationship, already modeled in the caldav MCP package via `parent_uid`)|
|ATTENDEE                      |referenced `!contact` object or inline metadata field                                                       |
|X-* extensions, SEQUENCE, etc.|opaque properties preserved in the object’s blob                                                            |

On **compilation** (checkout), the workspace walks the task object’s references
and blob references, assembles a VCALENDAR document with the VTODO and its
VALARMs, ATTACHments, etc. On **decompilation** (checkin), the VTODO is parsed,
sub-structures are split back into their respective objects, references are wired
up, and new sub-objects are created in the workspace-repo as needed.

The graph decomposition also gives per-sub-object merge granularity (see
Three-Way Merge below).

### How Deep to Decompose

Not every VTODO sub-structure needs its own dodder object. The right depth
depends on whether the sub-structure is independently addressable and reusable:

- **VALARM → `!alarm` object:** Yes — alarms could be shared across tasks,
  queried independently (“show all my reminders”), or managed by agents.
- **ATTACH → blob reference:** Yes — attachments are content-addressable blobs,
  exactly what blob references are for.
- **RELATED-TO (subtasks) → referenced `!task`:** Yes — subtasks are
  independent objects with their own lifecycle.
- **ATTENDEE → `!contact`:** Maybe — depends on whether the workspace cares
  about contacts as first-class objects.
- **X-* extensions → inline blob content:** No — server-specific extensions are
  opaque metadata, not independent objects. Preserve them in the parent object’s
  blob.

The workspace config’s mapping declares the decomposition depth. A minimal
mapping treats the entire VTODO as a single `!task` with no sub-object
decomposition (X-* properties, VALARMs, and all are blob content). A richer
mapping decomposes VALARMs into `!alarm` objects and ATTACHes into blob
references. This is configurable per workspace, not hardcoded.

### Workspace Config

The workspace config (V2, extending V1 from FDR-0005) declares a checkout store,
its connection details, and type-to-schema mappings:

```toml
[checkout-store]
type = "caldav"

[checkout-store.caldav]
url = "https://caldav.example.com/dav/calendars/user/tasks/"
username = "alice"
# password from DODDER_CALDAV_PASSWORD or OS keychain

[checkout-store.mappings."!task"]
component = "VTODO"

[checkout-store.mappings."!task".fields]
summary = "description"         # dodder object description → VTODO SUMMARY
body = "blob"                   # dodder blob content → VTODO DESCRIPTION
tags = "categories"             # dodder tags → CATEGORIES
priority = "meta.priority"      # type-defined metadata → PRIORITY (0-9)
due = "meta.due"                # type-defined metadata → DUE
dtstart = "meta.dtstart"        # type-defined metadata → DTSTART
status = "meta.status"          # type-defined metadata → STATUS
location = "meta.location"      # type-defined metadata → LOCATION
rrule = "meta.rrule"            # type-defined metadata → RRULE
parent = "meta.parent_uid"      # type-defined metadata → RELATED-TO;RELTYPE=PARENT

[checkout-store.mappings."!task".decompose]
alarm = "!alarm"                # VALARM → referenced !alarm objects
attach = "blob"                 # ATTACH → blob references

[checkout-store.mappings."!event"]
component = "VEVENT"
# ... similar field mappings
```

The mapping lives in the workspace config, not the type definition. Rationale:
the same `!task` type checks out to CalDAV in one workspace, to `store_fs` in
another, and might check out to a Kanban board in a third. The type defines what
the object *is*; the workspace defines how it’s *compiled*.

> **Open question:** Should the mapping be versioned? If the workspace config’s
> mapping changes, existing objects with external GUID bindings were checked out
> under the old mapping. Options: (a) version the mapping and require re-sync
> on change, (b) treat mapping changes as append-only (new fields compile but
> old bindings don’t break), (c) store the mapping version in the sync state
> per object.

### External GUID Binding

Each dodder object that has been checked out to an external store carries the
external system’s identifier. For CalDAV, this is the VTODO’s UID (a globally
unique string, typically a UUID).

The binding is stored in the workspace’s sync state (see below), not on the
object itself. Rationale: the binding is between a workspace and an external
store instance, not between an object and an abstract external identity. The same
object may have different VTODO UIDs in two different CalDAV servers (one per
workspace). Keeping the binding in sync state means the object’s metadata stays
clean of checkout-store-specific fields and the type definition doesn’t need to
know about CalDAV.

The tradeoff: bindings don’t travel with objects on push to parent. If you delete
and recreate a CalDAV-backed workspace, the objects are re-checked-out with new
VTODO UIDs. External CalDAV clients see them as new tasks. For most workflows
this is acceptable — the workspace is the persistent sync relationship.

### Sync Cycle

Sync is bidirectional. The workspace-repo is the source of truth for dodder
metadata (locks, tags, type, commit history). The external store is the source of
truth for user-facing mutations made through external clients.

**Checkout (compilation: dodder → external):**

1. For each object matching the workspace query whose type has a mapping:
1. Walk the object’s references and blob references to build the sub-object graph
1. Compile the graph into the target format (VCALENDAR with VTODO, VALARMs, etc.)
1. If no external GUID binding exists, create a new resource in the external
   store (PUT with `If-None-Match: *`), record the GUID binding
1. If a binding exists, update the existing resource (PUT with `If-Match: <etag>`)
1. Store the hyphence text of each dodder object in the graph as the sync base

**Checkin (decompilation: external → dodder):**

1. Detect changed external resources (CalDAV sync-token / CTAG, or ETag
   comparison per resource)
1. Fetch the current external state for changed resources
1. Decompile: parse the VTODO into dodder fields and sub-object graph
1. For each object in the decompiled graph, three-way merge against the sync base
1. Create new dodder objects for new sub-structures (e.g., VALARM added
   externally → new `!alarm` object), wire references
1. Remove references for deleted sub-structures (e.g., VALARM removed externally)
1. Commit merged results to the workspace-repo as a batch
   (FDR-0006 two-stage commit)

**Discover (external → dodder, new resources):**

1. Find external resources with no GUID binding (created in CalDAV, not
   originating from dodder)
1. Infer dodder type from component type + workspace mappings (VTODO → `!task`)
1. Decompile into a new object graph
1. Allocate zettel IDs (FDR-0006 plan phase), commit, record binding

**Deletion handling:**

- External resource deleted → flag the dodder object for user decision (delete,
  re-checkout, or ignore). Not automatic — CalDAV deletion is a normal workflow
  in tasks.org, and silently deleting the dodder object would be surprising.
- Dodder object deleted → remove the external resource on next checkout.

### Three-Way Merge

Conflict resolution uses text-based three-way merge on the hyphence
representation of each object in the graph. The key insight: because the object
graph decomposes sub-structures into separate dodder objects, merge operates
per-object. This gives sub-structure-level conflict granularity without building
a custom per-field merge engine.

**Three versions per object:**

- **Base** — the hyphence text at last sync time, stored in sync state
- **Ours** — the current dodder object rendered to hyphence
- **Theirs** — the external resource decompiled to dodder and rendered to
  hyphence

**Merge procedure:**

1. Render all three versions to hyphence text
1. Run a standard three-way text merge (`diff3` / git’s `merge-file`)
1. If clean: parse the merged hyphence back into a dodder object, commit
1. If conflict: write the conflict-marked hyphence to a staging area, open in
   `$EDITOR` (like `organize`), parse the resolved text, commit

**Graph-level merge (no per-field logic needed):**

Because sub-structures are separate objects, concurrent edits to different parts
of a VTODO decompose into independent merges:

|CalDAV side changed|Dodder side changed          |Merge result                                                |
|-------------------|-----------------------------|------------------------------------------------------------|
|Added VALARM       |Changed description          |No conflict — new `!alarm` object + clean text merge on task|
|Changed due date   |Added a tag                  |No conflict — different lines in the task’s hyphence        |
|Changed due date   |Changed due date             |Conflict — same line in the task’s hyphence, user resolves  |
|Deleted VALARM     |(no change)                  |Clean — remove `!alarm` reference                           |
|Added ATTACH       |Added ATTACH (different file)|No conflict — two new blob references                       |

The hyphence format’s line-oriented structure makes text merge viable:
description is one line, each tag is one line, each metadata field is one line,
each reference is one line, blob content is the body after `---`. Fields that
changed only on one side merge cleanly. Fields that changed on both sides produce
conflict markers at the line level.

**Deterministic transform requirement:** The decompilation must be deterministic —
same iCalendar input must always produce the same hyphence output. This means
normalizing property order (sort alphabetically within each component), unfolding
lines (RFC 5545 line folding), and canonicalizing date formats before rendering
to hyphence. Without this, CalDAV server-side normalization (property reordering,
whitespace changes) produces phantom diffs.

**Future improvements** (not in this FDR):

- Per-field semantic merge (e.g., categories are sets — union rather than
  text-level merge)
- Timestamp-aware merge (prefer the more recent `LAST-MODIFIED`)
- Custom merge drivers per field type (analogous to git’s merge drivers)
- Automatic resolution policies configurable per workspace (e.g., “CalDAV wins
  for status changes, dodder wins for tags”)

### Opaque Property Preservation

iCalendar properties that have no dodder mapping (SEQUENCE, X-* extensions,
server-specific metadata) must survive the round-trip. The compilation model
handles this naturally: the task object’s blob stores the full iCalendar text (or
a normalized subset of it) alongside the structured dodder content. On
compilation, the structured fields are injected into the preserved iCal template.
On decompilation, unmapped properties stay in the blob.

This means the blob format for `!task` objects in a CalDAV-backed workspace
embeds iCalendar as a section — either as the entire blob (the type’s native
format *is* iCalendar) or as a fenced section within a richer format. The choice
affects how `store_fs` checkout renders the object:

- **Option A: iCalendar blob.** The blob is raw iCalendar. `store_fs` checkout
  writes `.ics` files. Simple, but the object looks like iCalendar even in
  non-CalDAV contexts.
- **Option B: Structured blob with iCal passthrough section.** The blob uses the
  type’s canonical format (markdown with TOML frontmatter, structured text, etc.)
  and has a designated section for opaque iCal passthrough. `store_fs` checkout
  writes readable files; compilation extracts the passthrough and merges it with
  the structured fields.
- **Option C: Passthrough as a blob reference.** The opaque iCal properties are
  stored as a separate blob (`- ical-passthrough < @blake2b256-...`). The main
  blob stays clean. Compilation reads both.

> **Open question:** Which option best balances round-trip fidelity, readability
> in `store_fs`, and complexity? Option B is the most ergonomic but requires the
> type’s formatter to understand the passthrough section. Option C is the most
> principled (uses existing blob reference infrastructure) but adds indirection.

### Sync State

The workspace stores sync state in `.dodder/sync-state/`:

```
.dodder/sync-state/
  <uid>.hyphence       # base text at last sync (for three-way merge)
  <uid>.etag           # external resource's ETag at last sync
  manifest.json        # uid → external-guid mapping, last sync timestamp
```

`manifest.json` tracks:

```json
{
  "store_type": "caldav",
  "last_sync": "2026-03-20T14:30:00Z",
  "sync_token": "http://example.com/sync/token-123",
  "bindings": {
    "ceroplastes/midtown": {
      "external_uid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "etag": "\"abc123\"",
      "sync_tai": "1742486400.0",
      "sub_objects": {
        "ceroplastes/alarm-1": "valarm",
        "ceroplastes/alarm-2": "valarm"
      }
    }
  }
}
```

Sub-object tracking in the manifest records which dodder objects were created by
decompilation, so that a deleted VALARM on the CalDAV side can be traced back to
the specific `!alarm` object to remove.

### Checkout Store Interface

The checkout store is a Go interface, not an MCP connection. MCP adds latency
and serialization overhead that make sense for remote tool invocation but not for
a tight sync loop processing hundreds of objects. The caldav MCP server
(`packages/caldav` in `amarbel-llc/bob`) and the checkout store share the same
`internal/caldav` client package — they are two consumers of the same CalDAV
client, not layered on top of each other.

```go
// CheckoutStore is the interface a workspace checkout medium must implement.
type CheckoutStore interface {
    // Compile writes a dodder object graph to the external store.
    // graph contains the root object and all referenced sub-objects.
    // Returns the external UID (new or existing).
    Compile(ctx context.Context, graph *ObjectGraph, mapping *TypeMapping) (externalUID string, err error)

    // Decompile reads the external resource and returns the object graph
    // as a set of hyphence texts keyed by role (root, alarm, attachment, etc.).
    Decompile(ctx context.Context, externalUID string, mapping *TypeMapping) (*ObjectGraphDiff, error)

    // Discover returns external resources that have no dodder binding
    // (created externally since last sync).
    Discover(ctx context.Context, mapping *TypeMapping) ([]ExternalResource, error)

    // Delete removes an external resource.
    Delete(ctx context.Context, externalUID string) error

    // SyncToken returns the current sync token for change detection.
    // Returns empty string if the store doesn't support sync tokens.
    SyncToken(ctx context.Context) (string, error)

    // Changes returns resources changed since the given sync token.
    Changes(ctx context.Context, syncToken string) ([]ChangedResource, error)
}
```

`ObjectGraph` contains the root object and its referenced sub-objects (alarms,
attachments), resolved from the workspace-repo’s store. `ObjectGraphDiff`
contains hyphence texts for new, updated, and deleted objects in the graph,
ready for three-way merge.

Implementations: `store_fs` (existing, adapted to the interface), `store_caldav`
(new, wraps `internal/caldav.Client`), future `store_webdav`, `store_git`, etc.

### Interaction with Workspace-Repo Isolation

The checkout store syncs with the workspace-repo, not the parent. The flow is:

```
parent repo ←—push/pull—→ workspace-repo ←—compile/decompile—→ CalDAV server
                                                                      ↕
                                                                tasks.org app
```

Agent mutations hit the workspace-repo. Explicit `sync` (or auto-sync on
workspace lock release) compiles to CalDAV. External CalDAV mutations are
decompiled on the next `sync` or `checkin`. The workspace-repo’s divergence
detection (FDR-0005) tracks changes relative to the parent; sync state tracks
changes relative to CalDAV.

This means `check-workspace dirty` (FDR-0005) now has two dimensions:

- **Dirty relative to parent** — existing, based on `SyncTai`/`SyncDigest`
- **Dirty relative to external store** — new, based on sync token / ETag
  comparison

### Interaction with Two-Stage Commit

Checkin (decompilation) is a batch mutation: it may create new objects (for new
sub-structures), update existing objects, and remove references. This maps
directly to FDR-0006’s two-stage commit:

1. **Plan phase:** decompile changed external resources, compute three-way merges,
   classify results (create, update, conflict), allocate zettel IDs for new
   sub-objects
1. **Commit phase:** commit the plan under LockSmith

The `-dry-run` capability from FDR-0006’s Builder unification applies here too:
`sync -dry-run` would decompile and show the merge results without committing.

## Examples

Initialize a CalDAV-backed workspace:

```
$ dodder init-workspace -experimental-repo \
    -checkout-store caldav \
    -caldav-url https://caldav.example.com/dav/calendars/user/tasks/ \
    -caldav-username alice \
    project-tasks '+task project-alpha'

workspace-repo created at .dodder/
checkout store: caldav (https://caldav.example.com/...)
pulled 47 tasks from parent
compiled 47 VTODOs to CalDAV (12 with alarms, 3 with attachments)
```

Sync after external changes (tasks.org marked 3 tasks complete, added a
reminder):

```
$ dodder sync
decompile: 4 external changes
  ceroplastes/midtown: STATUS NEEDS-ACTION → COMPLETED (clean merge)
  papilio/uptown: STATUS NEEDS-ACTION → COMPLETED (clean merge)
  bombyx/downtown: STATUS + description changed (clean merge)
  bombyx/downtown: new VALARM → created !alarm bombyx/alarm-1
compile: 0 local changes
sync complete: 3 updated, 1 created
```

Sync with graph-level non-conflict (different sub-objects edited on each side):

```
$ dodder sync
decompile: 1 external change
  ceroplastes/midtown: VALARM trigger changed (tasks.org changed reminder)
compile: 1 local change
  ceroplastes/midtown: description updated (dodder side)
merge:
  ceroplastes/midtown (task): clean — only dodder side changed
  ceroplastes/alarm-1 (alarm): clean — only CalDAV side changed
sync complete: 2 updated
```

Conflict (both sides edited the same object):

```
$ dodder sync
decompile: 1 external change
  ceroplastes/midtown: description changed on both sides
opening conflict in $EDITOR...

# ceroplastes/midtown
- project-alpha
! task@blake2b256-abc...
---
<<<<<<< dodder
Updated project timeline: Q3 launch confirmed.
=======
Updated project timeline: Q3 launch delayed to Q4.
>>>>>>> caldav
```

Dry-run to preview sync without committing:

```
$ dodder sync -dry-run
decompile: 2 external changes
  ceroplastes/midtown: STATUS NEEDS-ACTION → COMPLETED
  bombyx/downtown: new VALARM (would create !alarm)
compile: 1 local change
  papilio/uptown: tags updated
(dry run — no changes committed)
```

## Limitations

- Only `store_fs` exists today. `store_caldav` is the first non-filesystem
  implementation and defines the interface by example.
- Three-way merge on hyphence text is line-level, not semantic. Concurrent edits
  to the same line (e.g., both sides change the due date) produce a conflict
  even though a field-level merge could resolve it automatically.
- CalDAV servers vary in compliance. tasks.org, Radicale, Nextcloud, and
  Apple’s CalDAV server handle ETags, CTAG, and sync-token differently. The
  initial implementation targets Radicale (the simplest) and tasks.org (the
  most common Android client).
- External resources created in CalDAV (not originating from dodder) require
  type inference — Discover must infer dodder type from the iCalendar component
  type and the workspace’s mappings.
- Opaque property round-trip preservation depends on the CalDAV server not
  stripping or rewriting unknown properties. Most servers preserve X-*
  properties but this is not guaranteed by RFC 4791.
- Object graph decomposition depth is configured per workspace, not enforced
  globally. Two workspaces with different decomposition depths checking out the
  same parent objects will produce different sub-object graphs. Push to parent
  includes the sub-objects, so the parent repo may accumulate `!alarm` objects
  from one workspace that the other workspace doesn’t decompose.

## More Information

- [FDR-0001: Object Locks](0001-object-locks.md) — referenced objects and blob
  references used for sub-object graph decomposition
- [FDR-0005: Workspace-as-Repo](0005-workspace-as-repo.md) — workspace-repo
  architecture, pluggable checkout stores listed as future work
- [FDR-0004: Bindingless Local Repo Transfer](0004-bindingless-local-repo-transfer.md) —
  `-direct` mechanism for workspace ↔ parent sync
- [FDR-0006: Two-Stage Commit](0006-two-stage-commit.md) — plan-based batch
  commit used by decompilation/checkin
- CalDAV MCP server (`packages/caldav` in `amarbel-llc/bob`) — existing CalDAV
  client and iCalendar parser/serializer, shared code base for `store_caldav`