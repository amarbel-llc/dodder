---
name: design_pattern-snob_naming
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

### Skill Names

| Name | Part Breakdown |
|------|----------------|
| `design_pattern-horizontal_versioning` | `design_pattern` \| `horizontal_versioning` |
| `design_pattern-pool_repool` | `design_pattern` \| `pool_repool` |
| `design_pattern-typed_error_sentinels` | `design_pattern` \| `typed_error_sentinels` |

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
| `design_pattern-tripleHyphenIo` (camelCase) | `design_pattern-triple_hyphen_io` |
| `designPattern-tripleHyphenIo` (camelCase both) | `design_pattern-triple_hyphen_io` |
| `design-pattern-triple-hyphen-io` (all kebab) | `design_pattern-triple_hyphen_io` |
