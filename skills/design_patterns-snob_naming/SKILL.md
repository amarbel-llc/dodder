---
name: design_patterns-snob_naming
description: >
  Use when naming packages, directories, type strings, skills, or any
  identifiers that need structured multi-part names. Also applies when
  reviewing names for convention compliance or encountering naming
  inconsistencies involving hyphens and underscores.
triggers:
  - naming convention
  - snob naming
  - package name
  - directory name
  - type string name
  - skill name
  - hyphen vs underscore
---

# Snob Naming

## Overview

Snob naming (snake + kebab) is the naming convention used across dodder and
related projects. Hyphens separate distinct semantic parts. Underscores join
words within a single semantic part. This creates a two-level hierarchy where
splitting on `-` yields major components and splitting on `_` yields the words
composing each component.

## The Rule

```
<compound_noun>-<compound_noun>-<compound_noun>
       │                │               │
  words_joined     words_joined    words_joined
  by_underscores   by_underscores  by_underscores
```

- **Hyphens** (`-`) separate distinct semantic parts
- **Underscores** (`_`) join words within a single semantic part

## Examples

### Package Names

| Name | Parts |
|------|-------|
| `blob_store_configs` | single part: "blob store configs" |
| `triple_hyphen_io` | single part: "triple hyphen io" |
| `type_blobs` | single part: "type blobs" |
| `inventory_list_coders` | single part: "inventory list coders" |

### Type Strings

| Name | Part Breakdown |
|------|----------------|
| `!toml-blob_store_config-v0` | `toml` \| `blob_store_config` \| `v0` |
| `!toml-blob_store_config_sftp-ssh_config-v0` | `toml` \| `blob_store_config_sftp` \| `ssh_config` \| `v0` |
| `!toml-repo-dotenv_xdg-v0` | `toml` \| `repo` \| `dotenv_xdg` \| `v0` |
| `!toml-workspace_config-v0` | `toml` \| `workspace_config` \| `v0` |
| `!inventory_list-v1` | `inventory_list` \| `v1` |
| `!zettel_id_list-v0` | `zettel_id_list` \| `v0` |

### Skill Names (in `skills/` directory — plural category)

| Name | Part Breakdown |
|------|----------------|
| `design_patterns-horizontal_versioning` | `design_patterns` \| `horizontal_versioning` |
| `design_patterns-pool_repool` | `design_patterns` \| `pool_repool` |
| `design_patterns-typed_error_sentinels` | `design_patterns` \| `typed_error_sentinels` |

## Pluralization

Plurals are contextual — they reflect the relationship between the name and
its container.

**Inside a plural container, use plural.** Skills live in `skills/`, so the
category part is plural to match: `design_patterns-pool_repool`. Packages in
Go are often plural when they hold collections of things: `blob_store_configs`,
`type_blobs`, `inventory_list_coders`.

**At the top level or in type strings, prefer singular.** Type string
identifiers name a single thing: `!toml-blob_store_config-v0` (not
`blob_store_configs`). Standalone identifiers default to singular unless the
name inherently refers to a collection.

| Context | Example | Why |
|---------|---------|-----|
| Type string | `!toml-blob_store_config-v0` | Names one config schema — singular |
| Package name | `blob_store_configs` | Contains multiple config versions — plural |
| Skill in `skills/` | `design_patterns-pool_repool` | Lives inside `skills/` — plural category |
| Standalone identifier | `design_pattern` | Refers to the concept itself — singular |

## How to Apply

When naming something with multiple semantic parts:

1. **Identify the distinct concepts.** Each concept becomes a hyphen-separated
   part. In `toml-blob_store_config-v0`: the format (`toml`), the thing being
   configured (`blob_store_config`), and the version (`v0`) are three distinct
   concepts.

2. **Join multi-word concepts with underscores.** "blob store config" is a
   single concept composed of three words, so it becomes `blob_store_config`.

3. **Separate concepts with hyphens.** The format, the thing, and the version
   are different concepts, so they get hyphens between them.

4. **Choose singular or plural based on context.** Inside a plural container
   (like `skills/`), use plural for the category part. For type strings and
   standalone identifiers, prefer singular.

## Common Patterns in Type Strings

Type strings follow a consistent structure:

```
!<format>-<thing>-<variant>-<version>
```

| Segment | Examples |
|---------|---------|
| format | `toml`, `lua`, `json` |
| thing | `config`, `blob_store_config`, `tag`, `type` |
| variant (optional) | `pointer`, `inventory_archive`, `ssh_config` |
| version | `v0`, `v1`, `v2` |

## Common Mistakes

| Mistake | Correct |
|---------|---------|
| `blob-store-configs` (all kebab) | `blob_store_configs` (single compound noun) |
| `triple_hyphen_io-toml_v0` (wrong split) | Depends on semantics — split on concept boundaries |
| `design_pattern-tripleHyphenIo` (camelCase) | `design_patterns-triple_hyphen_io` |
| `designPattern-tripleHyphenIo` (camelCase both) | `design_patterns-triple_hyphen_io` |
| `design-pattern-triple-hyphen-io` (all kebab) | `design_patterns-triple_hyphen_io` |
| `design_pattern-pool_repool` in `skills/` (singular in plural container) | `design_patterns-pool_repool` |
