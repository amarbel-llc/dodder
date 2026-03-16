# Blob References Design

**Goal:** Extend the reference system to support blob references alongside
object references, enabling types to discover both object and blob dependencies
from content.

**Context:** FDR-0001 (Object Locks) documents blob references as future work.
Object references are fully implemented with type-driven discovery via
`[object-references]` in type TOML. This design adds blob references using the
same discovery mechanism.

## Data Model

New `BlobReferences` collection on metadata, separate from `ContainedObjects`
because the key type is `markl.Id` (content-addressed digest), not `SeqId`
(object identifier).

```
blobReferenceEntry {
    Key   markl.Id    // blob digest (e.g., blake2b256-abc...)
    Alias string      // optional alias (e.g., "hero-image")
}
```

No lock value is needed — the digest IS the pin. Blobs are immutable and
content-addressed, so the digest itself serves as the version pin.

## Type TOML

Rename `[object-references]` to `[references]`. Same `ScriptConfig` + `Optional`
shape. No migration needed — `[object-references]` has not shipped to consumers.

```toml
[references]
shell = ['bash', '-c']
script = "grep -oP '\\[\\[(.+?)\\]\\]' | sed 's/\\[\\[//;s/\\]\\]//'"
optional = false
```

## Discovery Script Contract

**Input:** Blob content piped to stdin.

**Output:** One reference per line on stdout. Lines starting with `@` (or with
`@` after `= `) are blob references; others are object references.

```
one/dos                            # object reference
blog-template = one/uno            # object reference with alias
@blake2b256-abc...                 # blob reference
hero-image = @blake2b256-abc...    # blob reference with alias
# comments and blank lines ignored
```

**Exit code:** 0 = success, non-zero = failure (blocked or warned depending on
`optional`).

`parseReferenceOutput` detects `@` prefix to dispatch between `AddReference`
(object) and `AddBlobReference` (blob).

## Serialization

| Format             | Object ref          | Blob ref                 |
|--------------------|---------------------|--------------------------|
| Box/inventory list | `<one/uno@sig`      | `<@blake2b256-abc...`    |
| Box with alias     | `alias<one/uno@sig` | `alias<@blake2b256-abc...` |
| Hyphence           | `- one/uno@sig`     | `- @blake2b256-abc...`   |
| Hyphence with alias| `- alias < one/uno@sig` | `- alias < @blake2b256-abc...` |

Parsing rule: `<` is the separator. Text before `<` is the alias. After `<`, if
it starts with `@` it is a blob reference; otherwise it is an object reference.

## Metadata Interface

New methods on `Metadata` / `MetadataMutable`:

- `AllBlobReferences() Seq[markl.Id]`
- `AddBlobReference(markl.Id) error`
- `SetBlobReferenceAlias(markl.Id, string) error`
- `GetBlobReferenceAlias(markl.Id) string`

## GC Implication

Blob references prevent GC of referenced blobs. No GC changes in this
implementation — the data is stored for when GC needs it.

## Rollback

Purely additive. Revert the commits. No migration needed.
