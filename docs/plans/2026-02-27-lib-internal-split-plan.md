# lib/internal Split Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task.

**Goal:** Split `go/src/` into `go/lib/` (62 domain-agnostic packages) and
`go/internal/` (remaining domain-specific packages), preserving the NATO
phonetic hierarchy in both trees.

**Architecture:** Two-phase approach. Phase 1 moves the `StoreVersion` interface
from `_/interfaces` to `alfa/domain_interfaces` (the only domain-specific
interface remaining in `_/interfaces`). Phase 2 renames `src/` to `internal/`,
creates `lib/`, moves packages, and rewrites all import paths.

**Tech Stack:** Go, git, sed/find for bulk import rewriting

**Design doc:** `docs/plans/2026-02-27-lib-internal-split-design.md`

---

## Phase 1: Extract StoreVersion from \_/interfaces

### Task 1: Move StoreVersion interface to domain\_interfaces

**Files:**
- Modify: `go/src/_/interfaces/store_version.go` (delete)
- Modify: `go/src/alfa/domain_interfaces/` (add StoreVersion)
- Modify: all files importing `interfaces.StoreVersion` (14 files)

**Step 1: Copy StoreVersion to domain\_interfaces**

Add to `go/src/alfa/domain_interfaces/store_version.go`:

```go
package domain_interfaces

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

// TODO combine with config_immutable.StoreVersion and make a sealed struct
type StoreVersion interface {
	interfaces.Stringer
	GetInt() int
}
```

**Step 2: Delete the old file**

```bash
rm go/src/_/interfaces/store_version.go
```

**Step 3: Update importers**

For each of the 14 files that reference `interfaces.StoreVersion`, update them
to import `alfa/domain_interfaces` and use `domain_interfaces.StoreVersion`
instead. The files are:

- `go/src/charlie/store_version/main.go`
- `go/src/charlie/store_version/errors.go`
- `go/src/echo/ids/types_builtin.go`
- `go/src/hotel/genesis_configs/main.go`
- `go/src/hotel/genesis_configs/toml_v0.go`
- `go/src/hotel/genesis_configs/toml_v1.go`
- `go/src/hotel/genesis_configs/toml_v2.go`
- `go/src/juliett/env_repo/main.go`
- `go/src/mike/inventory_list_store/main.go`
- `go/src/tango/store/mutating.go`
- `go/src/victor/local_working_copy/main.go`
- `go/src/yankee/commands_dodder/info.go`
- `go/src/yankee/commands_dodder/info_repo.go`

For each file: add `domain_interfaces` import, replace
`interfaces.StoreVersion` with `domain_interfaces.StoreVersion`.

**Step 4: Verify compilation**

```bash
cd go && go build ./...
```

Expected: compiles cleanly.

**Step 5: Run unit tests**

```bash
just test-go
```

Expected: all pass.

**Step 6: Commit**

```bash
git add -A
git commit -m "refactor: move StoreVersion interface to domain_interfaces"
```

---

## Phase 2: Rename src/ to internal/, create lib/, move packages

### Task 2: Rename src/ to internal/

**Step 1: git mv the directory**

```bash
cd /path/to/worktree
git mv go/src go/internal
```

**Step 2: Rewrite all import paths**

Use find+sed to replace every occurrence of
`code.linenisgreat.com/dodder/go/src/` with
`code.linenisgreat.com/dodder/go/internal/` in all `.go` files under `go/`:

```bash
find go/ -name '*.go' -exec sed -i '' \
  's|code.linenisgreat.com/dodder/go/src/|code.linenisgreat.com/dodder/go/internal/|g' \
  {} +
```

**Step 3: Verify compilation**

```bash
cd go && go build ./...
```

Expected: compiles cleanly (everything is now under internal/).

**Step 4: Run unit tests**

```bash
just test-go
```

Expected: all pass.

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: rename go/src to go/internal"
```

---

### Task 3: Create lib/ directory structure and move packages

**Step 1: Create lib tier directories**

```bash
mkdir -p go/lib/_
mkdir -p go/lib/alfa
mkdir -p go/lib/bravo
mkdir -p go/lib/charlie
mkdir -p go/lib/delta
mkdir -p go/lib/echo
```

**Step 2: Move underscore-tier packages (15 packages)**

```bash
git mv go/internal/_/bech32 go/lib/_/bech32
git mv go/internal/_/box_chars go/lib/_/box_chars
git mv go/internal/_/equality go/lib/_/equality
git mv go/internal/_/exec go/lib/_/exec
git mv go/internal/_/flag_policy go/lib/_/flag_policy
git mv go/internal/_/hecks go/lib/_/hecks
git mv go/internal/_/http_statuses go/lib/_/http_statuses
git mv go/internal/_/interfaces go/lib/_/interfaces
git mv go/internal/_/mcp go/lib/_/mcp
git mv go/internal/_/ohio_buffer go/lib/_/ohio_buffer
git mv go/internal/_/primordial go/lib/_/primordial
git mv go/internal/_/reflexive_interface_generator go/lib/_/reflexive_interface_generator
git mv go/internal/_/reset go/lib/_/reset
git mv go/internal/_/stack_frame go/lib/_/stack_frame
git mv go/internal/_/vim_cli_options_builder go/lib/_/vim_cli_options_builder
```

**Step 3: Move alfa-tier packages (11 packages)**

```bash
git mv go/internal/alfa/analyzers go/lib/alfa/analyzers
git mv go/internal/alfa/cmp go/lib/alfa/cmp
git mv go/internal/alfa/collections_coding go/lib/alfa/collections_coding
git mv go/internal/alfa/collections_map go/lib/alfa/collections_map
git mv go/internal/alfa/equals go/lib/alfa/equals
git mv go/internal/alfa/errors go/lib/alfa/errors
git mv go/internal/alfa/pool go/lib/alfa/pool
git mv go/internal/alfa/quiter_collection go/lib/alfa/quiter_collection
git mv go/internal/alfa/quiter_seq go/lib/alfa/quiter_seq
git mv go/internal/alfa/reflexive_interface_generator go/lib/alfa/reflexive_interface_generator
git mv go/internal/alfa/unicorn go/lib/alfa/unicorn
```

**Step 4: Move bravo-tier packages (12 packages)**

```bash
git mv go/internal/bravo/collections_slice go/lib/bravo/collections_slice
git mv go/internal/bravo/comments go/lib/bravo/comments
git mv go/internal/bravo/delim_reader go/lib/bravo/delim_reader
git mv go/internal/bravo/env_vars go/lib/bravo/env_vars
git mv go/internal/bravo/flags go/lib/bravo/flags
git mv go/internal/bravo/lua go/lib/bravo/lua
git mv go/internal/bravo/ohio go/lib/bravo/ohio
git mv go/internal/bravo/quiter go/lib/bravo/quiter
git mv go/internal/bravo/quiter_set go/lib/bravo/quiter_set
git mv go/internal/bravo/string_builder_joined go/lib/bravo/string_builder_joined
git mv go/internal/bravo/ui go/lib/bravo/ui
git mv go/internal/bravo/values go/lib/bravo/values
```

**Step 5: Move charlie-tier packages (15 packages)**

```bash
git mv go/internal/charlie/catgut go/lib/charlie/catgut
git mv go/internal/charlie/cli go/lib/charlie/cli
git mv go/internal/charlie/collections go/lib/charlie/collections
git mv go/internal/charlie/collections_ptr go/lib/charlie/collections_ptr
git mv go/internal/charlie/collections_value go/lib/charlie/collections_value
git mv go/internal/charlie/compression_type go/lib/charlie/compression_type
git mv go/internal/charlie/delim_io go/lib/charlie/delim_io
git mv go/internal/charlie/expansion go/lib/charlie/expansion
git mv go/internal/charlie/files go/lib/charlie/files
git mv go/internal/charlie/heap go/lib/charlie/heap
git mv go/internal/charlie/script_config go/lib/charlie/script_config
git mv go/internal/charlie/toml go/lib/charlie/toml
git mv go/internal/charlie/tridex go/lib/charlie/tridex
git mv go/internal/charlie/trie go/lib/charlie/trie
git mv go/internal/charlie/xdg_defaults go/lib/charlie/xdg_defaults
```

**Step 6: Move delta-tier packages (8 packages)**

```bash
git mv go/internal/delta/age go/lib/delta/age
git mv go/internal/delta/alfred go/lib/delta/alfred
git mv go/internal/delta/collections_delta go/lib/delta/collections_delta
git mv go/internal/delta/debug go/lib/delta/debug
git mv go/internal/delta/editor go/lib/delta/editor
git mv go/internal/delta/ohio_files go/lib/delta/ohio_files
git mv go/internal/delta/script_value go/lib/delta/script_value
git mv go/internal/delta/thyme go/lib/delta/thyme
```

**Step 7: Move echo-tier packages (1 package)**

```bash
git mv go/internal/echo/config_cli go/lib/echo/config_cli
```

**Step 8: Commit the directory moves (before import rewriting)**

```bash
git add -A
git commit -m "refactor: move 62 domain-agnostic packages to go/lib/"
```

---

### Task 4: Rewrite import paths for lib packages

**Step 1: Rewrite imports**

For each of the 62 moved packages, replace
`code.linenisgreat.com/dodder/go/internal/{tier}/{pkg}` with
`code.linenisgreat.com/dodder/go/lib/{tier}/{pkg}` in all `.go` files.

Run a sed command per package. The full list:

```bash
cd /path/to/worktree

# Underscore tier
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/bech32|go/lib/_/bech32|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/box_chars|go/lib/_/box_chars|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/equality|go/lib/_/equality|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/exec|go/lib/_/exec|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/flag_policy|go/lib/_/flag_policy|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/hecks|go/lib/_/hecks|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/http_statuses|go/lib/_/http_statuses|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/interfaces|go/lib/_/interfaces|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/mcp|go/lib/_/mcp|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/ohio_buffer|go/lib/_/ohio_buffer|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/primordial|go/lib/_/primordial|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/reflexive_interface_generator|go/lib/_/reflexive_interface_generator|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/reset|go/lib/_/reset|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/stack_frame|go/lib/_/stack_frame|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/_/vim_cli_options_builder|go/lib/_/vim_cli_options_builder|g' {} +

# Alfa tier
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/analyzers|go/lib/alfa/analyzers|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/cmp|go/lib/alfa/cmp|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/collections_coding|go/lib/alfa/collections_coding|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/collections_map|go/lib/alfa/collections_map|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/equals|go/lib/alfa/equals|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/errors|go/lib/alfa/errors|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/pool|go/lib/alfa/pool|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/quiter_collection|go/lib/alfa/quiter_collection|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/quiter_seq|go/lib/alfa/quiter_seq|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/reflexive_interface_generator|go/lib/alfa/reflexive_interface_generator|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/alfa/unicorn|go/lib/alfa/unicorn|g' {} +

# Bravo tier
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/collections_slice|go/lib/bravo/collections_slice|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/comments|go/lib/bravo/comments|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/delim_reader|go/lib/bravo/delim_reader|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/env_vars|go/lib/bravo/env_vars|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/flags|go/lib/bravo/flags|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/lua|go/lib/bravo/lua|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/ohio|go/lib/bravo/ohio|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/quiter|go/lib/bravo/quiter|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/quiter_set|go/lib/bravo/quiter_set|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/string_builder_joined|go/lib/bravo/string_builder_joined|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/ui|go/lib/bravo/ui|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/bravo/values|go/lib/bravo/values|g' {} +

# Charlie tier
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/catgut|go/lib/charlie/catgut|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/cli|go/lib/charlie/cli|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/collections|go/lib/charlie/collections|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/collections_ptr|go/lib/charlie/collections_ptr|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/collections_value|go/lib/charlie/collections_value|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/compression_type|go/lib/charlie/compression_type|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/delim_io|go/lib/charlie/delim_io|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/expansion|go/lib/charlie/expansion|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/files|go/lib/charlie/files|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/heap|go/lib/charlie/heap|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/script_config|go/lib/charlie/script_config|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/toml|go/lib/charlie/toml|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/tridex|go/lib/charlie/tridex|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/trie|go/lib/charlie/trie|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/charlie/xdg_defaults|go/lib/charlie/xdg_defaults|g' {} +

# Delta tier
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/age|go/lib/delta/age|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/alfred|go/lib/delta/alfred|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/collections_delta|go/lib/delta/collections_delta|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/debug|go/lib/delta/debug|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/editor|go/lib/delta/editor|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/ohio_files|go/lib/delta/ohio_files|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/script_value|go/lib/delta/script_value|g' {} +
find go/ -name '*.go' -exec sed -i '' 's|go/internal/delta/thyme|go/lib/delta/thyme|g' {} +

# Echo tier
find go/ -name '*.go' -exec sed -i '' 's|go/internal/echo/config_cli|go/lib/echo/config_cli|g' {} +
```

Note: order matters for packages whose names are prefixes of others. The sed
commands above use full package paths (including trailing package name) which
avoids false matches. However, watch out for these cases:

- `bravo/quiter` vs `bravo/quiter_set` — sed on `bravo/quiter` must not match
  `bravo/quiter_set`. The import paths include the full package name in quotes
  so `bravo/quiter"` won't match `bravo/quiter_set"`. No issue.
- `charlie/collections` vs `charlie/collections_ptr` / `charlie/collections_value`
  — same logic applies, safe.

**Step 2: Verify compilation**

```bash
cd go && go build ./...
```

Expected: compiles cleanly.

**Step 3: Run unit tests**

```bash
just test-go
```

Expected: all pass.

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: rewrite import paths for lib/ packages"
```

---

### Task 5: Update documentation and non-Go references

**Files:**
- `go/CLAUDE.md`
- `go/src/alfa/errors/SENTINEL_GUIDE.md` (now `go/internal/alfa/errors/`)
- `.claude/skills/design_patterns-hamster_style/SKILL.md`
- `.claude/skills/dodder-development/SKILL.md`
- `.claude/skills/dodder-development/references/nato-hierarchy.md`
- Any `docs/plans/` files with import path examples

**Step 1: Update CLAUDE.md files**

Update import path references from `go/src/` to `go/lib/` or `go/internal/` as
appropriate. Update the module organization section to describe the lib/internal
split.

**Step 2: Update skill files**

Update any import path examples in `.claude/skills/` to reflect the new paths.

**Step 3: Update SENTINEL\_GUIDE.md**

Update code examples to use the new `go/lib/alfa/errors` import path.

**Step 4: Commit**

```bash
git add -A
git commit -m "docs: update references for lib/internal split"
```

---

### Task 6: Run full test suite

**Step 1: Build**

```bash
just build
```

Expected: builds successfully.

**Step 2: Run all tests**

```bash
just test
```

Expected: all unit tests and integration tests pass.

**Step 3: Verify no stale src/ references remain**

```bash
grep -r 'go/src/' go/ --include='*.go' | head -20
```

Expected: no matches.

```bash
grep -r 'dodder/go/src/' . --include='*.md' | head -20
```

Expected: no matches (or only in historical plan docs).

---

## Cautions

### Sed Ordering for Prefix Packages

When rewriting imports, some package names are prefixes of others:

- `bravo/quiter` is a prefix of `bravo/quiter_set` and
  `bravo/quiter_collection` (but `quiter_collection` is in `alfa/`)
- `charlie/collections` is a prefix of `charlie/collections_ptr` and
  `charlie/collections_value`

Since import paths appear in Go source as quoted strings
(`"code.linenisgreat.com/dodder/go/internal/bravo/quiter"`), the trailing `"`
prevents false matches. The sed commands above replace the full path segment
including the package directory name, so `bravo/quiter|` (pipe = end of word)
won't accidentally match `bravo/quiter_set|`.

### go/internal/ and Go's internal package rule

Go enforces that packages under `internal/` can only be imported by code rooted
at the parent of the `internal/` directory. Since `go/internal/` is under `go/`
(the module root), all code within the `go/` module can import from
`go/internal/`. This is the desired behavior — `go/internal/` packages are
private to the dodder module but accessible to all code within it.

### Integration test binaries

The BATS integration tests run against compiled binaries. The `just build` step
compiles these binaries, so as long as compilation succeeds, the binary paths
don't change. No BATS test modifications should be needed.
