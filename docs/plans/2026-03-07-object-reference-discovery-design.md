# Object Reference Discovery Design

## Summary

Add type-driven reference discovery so that types can define how to extract
object references from blob content. When an object is committed, its type's
discovery script receives the blob on stdin and outputs discovered references on
stdout. These are merged into the object's `References` collection before
pre-commit hooks and lock finalization.

## Motivation

The referenced object locks feature (implemented on this branch) pins
signatures of referenced objects at commit time. But references must currently
be manually specified via `< ref` lines in the triple-hyphen metadata. There is
no way for a type to say "here's how to find references in my blob content."

For example, a markdown type might want to automatically discover `[[wiki-link]]`
references. A config type might want to discover blob store references. Without
type-driven discovery, users must manually maintain reference lists, which is
error-prone and defeats the purpose of automatic lock pinning.

## Type Blob Schema

Add an `[object-references]` section to `TomlV1` using a new config struct that
embeds `script_config.ScriptConfig`:

```go
type ObjectReferencesConfig struct {
    script_config.ScriptConfig
    Optional bool `toml:"optional,omitempty"`
}
```

On `TomlV1`:

```go
ObjectReferences *ObjectReferencesConfig `toml:"object-references,omitempty"`
```

Example type blob:

```toml
file-extension = 'md'
vim-syntax-type = 'markdown'

[object-references]
shell = ['bash', '-c']
script = '''grep -oP '\[\[(.+?)\]\]' | sed 's/\[\[//;s/\]\]//' '''
```

## Command Interface

**Input:** Blob content piped to stdin.

**Output:** One reference per line on stdout. Two formats:

```
one/dos
blog-template = one/uno
```

- `<object-id>` — reference without alias
- `<alias> = <object-id>` — reference with alias

Empty lines and `#`-prefixed lines are ignored. Lines are trimmed of leading and
trailing whitespace.

**Exit codes:**
- 0 — success; stdout parsed as references
- Non-zero — discovery failed

## Error Handling

Configurable per type via `optional` field:

- `optional = false` (default) — discovery failure blocks the commit. If a type
  defines discovery, the script must succeed.
- `optional = true` — discovery failure logs a warning and the commit proceeds
  without discovered references. Manual `< ref` lines still work.

## Pipeline Integration

Discovery runs as a new step in the commit pipeline, after `SaveBlob` and
before `tryPreCommitHooks`:

```
SaveBlob → discoverReferences → tryPreCommitHooks → WriteLockfile → CalculateObjectDigest
```

This ordering means:
1. Blob content is available (already saved)
2. Pre-commit hooks see the full reference set (manual + discovered)
3. Lock finalization pins signatures for all references
4. Digest calculation includes all reference metadata

## Merge Behavior

Discovery augments manual references — it never removes them.

For each discovered reference:
- If the same object ID is already in `References` (from manual `< ref`), skip
  it. Don't override the existing alias or lock value.
- If the object ID is new, add it to `References`.

On re-commit (edit + checkin), discovery runs again on current blob content. New
references are added, but existing ones with locked values are preserved (lock
finalizer skips if lock already set).

If the object's type has no `[object-references]` section, discovery is skipped
entirely.

## Future Work

**Authoritative mode:** Currently references are purely additive — discovery
adds but never removes. A future iteration should support an authoritative mode
where the discovery output IS the complete set of references. Stale references
(present in metadata but absent from discovery output) would be removed. This
requires careful handling of manually-specified references vs discovered ones.

**Lua backend:** A second discovery backend using Lua hooks with a streaming
blob reader API. No subprocess overhead since the Lua VM is pooled. Would
require a new Lua userdata type for blob reading.

## Rollback

Purely additive. To revert: remove the `discoverReferences` pipeline step.
Existing objects with references (manual or discovered) continue to work.
Types with `[object-references]` sections have an unused field. No
dual-architecture period needed.
