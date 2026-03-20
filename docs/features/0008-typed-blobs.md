# Typed Blob References

## Context

The git bridge workspace requires dodder to represent a git repository as a
top-level object that presents a repo at a given commit on a given branch. The
tree structure underlying a commit --- directories mapping names to blobs and
subtrees --- needs to be expressible in dodder's object graph.

This immediately surfaces a structural limitation: blob references in objects
cannot themselves contain blob references. Blobs are leaf nodes. There is no
primitive for "a content-addressed node whose job is to map names to other
content-addressed nodes," which is exactly what a git tree (or a filesystem
directory, or an archive manifest) is.

## Design Progression

### Blob tree as a third primitive

The initial approach considered was a new core primitive type --- the blob tree
--- sitting between blobs and objects. It would be content-addressed, leaf-like
(no object metadata), and its content would be a list of named references to
blobs or other blob trees. This mirrors git's tree object exactly.

The problem: a third primitive type expands every layer of dodder that currently
assumes a two-kind world --- hashing, storage, traversal, the Wasm plugin
interface, WIT (Wasm Interface Type) definitions. The cost propagates broadly.

### Blob tree as a non-indexed structural node

A refinement recognized that the blob tree shouldn't be user-facing. Objects are
semantic, user-facing entities. A tree used internally by a workspace to
organize content is structural plumbing --- it should be reachable for graph
traversal and GC (Garbage Collection) purposes but not queryable in the index.
This is analogous to how git tree objects exist in the packfile but don't appear
in `git log`.

This resolved the visibility concern but still implied a third primitive. The
hash function is horizontally versioned on the primitive inventory list type, so
adding a new primitive is mechanically straightforward (a new row in the table),
but the conceptual expansion remains.

### Typed blobs as the generalization

The key insight: blob references in objects already carry type information.
Rather than introducing a new primitive, the design generalizes this by
requiring that all blob references carry an associated type. The blob itself
stays opaque bytes. The reference declares how to interpret the target.

A "blob tree" is then just a blob whose type declares that its content is a list
of named, typed references to other blobs. A "git commit" blob is just a blob
whose type declares fields for parent refs, a tree ref, author, and message. No
new primitive is needed. The storage layer and hash function are unchanged. The
two-primitive model (objects and blobs) is preserved.

## Design

### Core invariant

Every blob reference must have an associated type. There are no untyped blob
references.

This is the single new constraint. It ensures the graph walker can always
determine how to discover further references inside any blob, without
understanding the blob's content directly.

### Type-driven ref discovery

The type system becomes load-bearing infrastructure for graph operations. A type
definition must declare which fields within a blob's content are references, so
that traversal, GC reachability analysis, and sync can follow edges without
parsing blob content in an ad-hoc way.

The contract between a typed blob and dodder core is narrow: the type tells
dodder "here are the edges in this node." The core doesn't need to understand
the rest of the blob's content. This preserves the separation between structural
concerns (where are the edges) and semantic concerns (what does this data mean).

### Blob identity and hashing

Typed blobs are still blobs from the perspective of storage and hashing. The
type is carried on the reference, not baked into the blob's hash. Two references
to the same blob bytes can declare different types --- though in practice this
would be unusual and potentially an error worth flagging.

The primitive inventory's horizontal versioning is unaffected. No new rows are
added to the hash function dispatch table.

### Visibility and indexing

Typed blobs used as structural plumbing (e.g., tree blobs in a git workspace)
are reachable in the graph but not indexed as queryable entities. They exist for
traversal and GC but are invisible to end users. The workspace exposes its own
interface to users; the internal blob organization is an implementation detail.

Types may expose blob tree references to allow external graph traversal and
reachability testing, but this doesn't promote the blob to an indexed, queryable
object.

## Application: Git Bridge Workspace

With typed blob references, the git bridge workspace is composed of:

A **repo object** (a regular dodder object) serving as the user-facing entity.
It carries commit metadata needed to produce the next git commit:

- `branch` --- the active branch name
- `parents` --- list of git OIDs (Object IDs) for parent commits (list, not
  singular, to support merge representation)
- `tree` --- a dodder ref to a tree-typed blob representing the repo root
- `author` / `committer` --- identity strings
- `message` --- commit message (empty until an explicit commit action)

A **tree-typed blob** for each directory level, where each entry contains a
name, a type tag (tree or file), and a ref to either another tree-typed blob or
a content blob. The type definition for tree entries declares the ref field,
enabling dodder to traverse the tree without understanding git semantics.

**Content blobs** for file data, carrying dual digests: the dodder-native hash
and the original git OID. Unchanged blobs on the write path can emit their
original git OID directly into the reconstructed git tree, avoiding rehashing.

### Write path

When the workspace produces a git commit, it walks the tree-typed blob hierarchy
back into git tree/blob objects. The dual-digest model means only new or
modified blobs need to be hashed into git's scheme. The repo object's `parents`
field provides lineage. After the commit is created and pushed, a new repo
object is minted with updated parent OIDs and a fresh tree ref.

### Commit lifecycle

The `message` field on the repo object is empty until an explicit commit action.
Mutations to the tree mark the workspace as dirty. A commit requires a non-empty
message and flushes the current tree state to git. This mirrors git's staging
model.

## Implications

### Type registry scope expansion

The type registry, previously responsible for object field schemas, now also
describes blob content layouts --- specifically, where refs live within a blob's
byte stream. This is a narrowly scoped IDL (Interface Definition Language) for
blob internals: it declares edge locations without needing to specify full
deserialization.

### Audit requirement

The invariant that all blob references must be typed requires auditing the
existing codebase for any untyped blob references. Any current bare blob ref is
a violation of the new contract and needs a type annotation.

### Future workspace applications

Typed blob references are not git-specific. Any workspace that organizes
hierarchical content-addressed data benefits: filesystem workspaces, archive
workspaces, package manager workspaces. Per-entry metadata (mode bits,
permissions, symlink flags) can be encoded in the type definition for the entry,
keeping the blob primitive clean while allowing workspace-specific semantics at
the type layer.
