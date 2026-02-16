# Claude Skills Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Ship a purse-first skill-only plugin with three Claude Code skills for dodder: development, onboarding, and usage.

**Architecture:** Skill-only purse-first plugin with `.claude-plugin/plugin.json` manifest and three skills under `skills/`. Each skill has a lean SKILL.md (~1,500-2,000 words) with detailed content in `references/` for progressive disclosure.

**Tech Stack:** Purse-first plugin framework, Claude Code skill system (YAML frontmatter + Markdown)

---

### Task 1: Create plugin scaffold

**Files:**
- Create: `.claude-plugin/plugin.json`
- Create: `skills/dodder-development/SKILL.md` (placeholder)
- Create: `skills/dodder-onboarding/SKILL.md` (placeholder)
- Create: `skills/dodder-usage/SKILL.md` (placeholder)

**Step 1: Create plugin manifest**

Create `.claude-plugin/plugin.json`:

```json
{
  "name": "dodder",
  "skills": [
    "./skills/dodder-development",
    "./skills/dodder-onboarding",
    "./skills/dodder-usage"
  ]
}
```

**Step 2: Create skill directories with placeholder SKILL.md files**

Create empty SKILL.md placeholders in each skill directory so the structure is valid.

Each placeholder:

```yaml
---
name: <Skill Name>
description: placeholder
---

TODO
```

**Step 3: Commit scaffold**

```bash
git add .claude-plugin/ skills/
git commit -m "feat: scaffold purse-first skill-only plugin with three skills"
```

---

### Task 2: Write dodder-development SKILL.md

**Files:**
- Create: `skills/dodder-development/SKILL.md`

**Step 1: Write the skill**

The SKILL.md must cover (~1,800 words):

1. **Frontmatter** — third-person description with trigger phrases:
   - "work on dodder", "add a new command", "add a type", "register a coder",
     "fix dodder code", "add a NATO module", "repool", "pool management",
     "dodder architecture", "dodder development"

2. **Overview** — dodder is a distributed zettelkasten written in Go with three
   binaries (dodder/der user-facing, madder low-level). Content-addressable
   blob store with automatic IDs.

3. **Build & test cycle** — exact commands from justfile:
   - `just build` (debug + release to `go/build/debug/`, `go/build/release/`)
   - `just test` (all), `just test-go` (unit), `just test-bats` (integration)
   - `just test-bats-targets clone.bats` (specific files)
   - `just test-bats-tags migration` (by tag)
   - `just check` (vuln + vet + repool analyzer)
   - `just codemod-go-fmt` (goimports + gofumpt)

4. **NATO phonetic module hierarchy** — brief table of layers (alfa through
   yankee) with one-line descriptions. Rule: each layer may only depend on
   layers below it. Point to `references/nato-hierarchy.md` for full details.

5. **Critical rules** (imperative form):
   - Never dereference `sku.Transacted` pointers — use `ResetWith`
   - Always call `repool()` exactly once after `GetWithRepool()`
   - Use `//repool:owned` comment to suppress analyzer when intentionally
     discarding repool
   - Read `MIGRATION_LEARNINGS.md` before touching ObjectId, stream index,
     or store version code

6. **Codebase navigation** — where to find things:
   - `go/cmd/` — binary entry points
   - `go/src/alfa/` through `go/src/yankee/` — NATO layers
   - `zz-tests_bats/` — integration tests
   - `docs/plans/` — design and implementation docs

7. **Reference pointers** — explicit mentions of reference files

**Step 2: Verify word count**

Run: `wc -w skills/dodder-development/SKILL.md`
Expected: 1,500-2,000 words (excluding frontmatter)

**Step 3: Commit**

```bash
git add skills/dodder-development/SKILL.md
git commit -m "feat: write dodder-development skill"
```

---

### Task 3: Write dodder-development references

**Files:**
- Create: `skills/dodder-development/references/nato-hierarchy.md`
- Create: `skills/dodder-development/references/pool-management.md`
- Create: `skills/dodder-development/references/testing-workflow.md`

**Step 1: Write nato-hierarchy.md**

Full layer breakdown with:
- Each NATO layer (alfa through yankee) with key packages
- What each layer contains (2-3 sentences per layer)
- Dependency rules with examples of correct and incorrect imports
- How to add a new package to the right layer

Content source: the exploration data gathered (26 layers from alfa to yankee
plus the `_` vendor layer).

**Step 2: Write pool-management.md**

Cover:
- `GetWithRepool()` lifecycle (get, use, repool)
- Three-layer safety system:
  1. Static analyzer (`just check-go-repool`)
  2. Runtime debug poisoning (debug build tag)
  3. CI lint check
- `sku.Transacted` rules with code examples:
  - WRONG: `value := *object`
  - CORRECT: `sku.TransactedResetter.ResetWith(&dst, src)`
  - CORRECT: `obj.CloneTransacted()` + `defer pool.Put(cloned)`
- Common pitfall: hash lifetime in blob writers
- The `//repool:owned` suppression comment

**Step 3: Write testing-workflow.md**

Cover:
- BATS test structure (`zz-tests_bats/`)
- Fixture lifecycle:
  1. Code changes alter store format
  2. `just test-bats-update-fixtures` regenerates
  3. Review diff
  4. Commit fixtures before tests pass
- Writing new BATS tests (file naming, setup/teardown, assertions)
- Running subsets: `-targets`, `-tags`
- Store version and fixture versioning (`migration/v{N}/.xdg/`)
- Timeout: 5 seconds per test
- Debug: `DODDER_BIN`, debug binary on PATH

**Step 4: Commit**

```bash
git add skills/dodder-development/references/
git commit -m "feat: add development skill reference docs"
```

---

### Task 4: Write dodder-onboarding SKILL.md

**Files:**
- Create: `skills/dodder-onboarding/SKILL.md`

**Step 1: Write the skill**

The SKILL.md must cover (~1,500 words):

1. **Frontmatter** — triggers:
   - "get started with dodder", "set up dodder", "install dodder", "new to
     dodder", "what is dodder", "dodder tutorial", "learn dodder", "how does
     dodder work"

2. **What is dodder** — distributed zettelkasten, content-addressable blob
   store, automatic two-part IDs, flat hierarchy, full version tracking. Like
   Git for knowledge.

3. **Core concepts at a glance** — brief table:
   - Zettel: a note/blob with an automatic ID
   - Object ID: two-part human-friendly identifier (e.g., `one/uno`) from
     user-supplied yin/yang lists
   - Tag: organizational label (e.g., `project`, `-dependent`, `%virtual`)
   - Type: content format (e.g., `!md`, `!txt`, `!toml-config-v2`)
   - Blob: content-addressed by SHA hash
   - Store: `.dodder/` directory, like `.git/`

4. **Installation** — nix-based: `nix build` or devshell via
   `nix develop`. Binary is `dodder` (alias `der`).

5. **First steps** — walkthrough:
   ```bash
   dodder init -yin <(echo -e "one\ntwo\nthree") \
     -yang <(echo -e "uno\ndos\ntres") my-repo
   dodder new -edit=false
   dodder show :z
   ```

6. **Mental model** — flat (no directories), everything versioned, working copy
   vs. store (like Git), tags organize, types define format.

7. **Next steps** — point to dodder-usage skill for day-to-day workflows. Point
   to `references/concepts.md` for deeper understanding.

**Step 2: Verify word count**

Run: `wc -w skills/dodder-onboarding/SKILL.md`
Expected: 1,200-1,500 words

**Step 3: Commit**

```bash
git add skills/dodder-onboarding/SKILL.md
git commit -m "feat: write dodder-onboarding skill"
```

---

### Task 5: Write dodder-onboarding references

**Files:**
- Create: `skills/dodder-onboarding/references/concepts.md`

**Step 1: Write concepts.md**

Deeper explanation of:
- **Object ID system**: yin/yang lists, two-part combination, how new IDs are
  auto-assigned, the `peek-zettel-ids` debug command
- **ID genres**: zettel (`z`), tag (`t`), type (`e`), blob, repo (`r`),
  konfig (reserved)
- **Type system**: built-in types (`!md`, `!txt`, `!bin`, `!toml-config-v2`,
  `!toml-type-v1`), user-defined types, type versioning
- **Tags**: plain tags, dependent tags (`-prefix`), virtual tags (`%prefix`)
- **Content-addressable storage**: SHA hashing, deduplication, blob store
  structure (buckets by SHA prefix)
- **Versioning**: every change tracked, inventory lists as source of truth,
  stream index for fast access
- **Working copy vs store**: `.dodder/` (store), filesystem (working copy),
  checkout states (Internal, CheckedOut, Untracked, Recognized, Conflicted)

**Step 2: Commit**

```bash
git add skills/dodder-onboarding/references/
git commit -m "feat: add onboarding skill reference docs"
```

---

### Task 6: Write dodder-usage SKILL.md

**Files:**
- Create: `skills/dodder-usage/SKILL.md`

**Step 1: Write the skill**

The SKILL.md must cover (~2,000 words):

1. **Frontmatter** — triggers:
   - "create a zettel", "organize zettels", "dodder checkout", "dodder push",
     "dodder pull", "query dodder", "find zettels", "sync dodder", "tag a
     zettel", "edit a zettel", "dodder commands", "der", "dodder status",
     "dodder show"

2. **dodder and der** — same tool, `der` is a shortname. `madder` is excluded
   (low-level, separate skill).

3. **Common workflows** (imperative form, with command examples):

   **Creating content:**
   ```bash
   dodder new                         # create empty, open editor
   dodder new -edit=false             # create empty, no editor
   dodder new -edit=false file.txt    # import from file
   dodder new -tags project -type md  # with metadata
   ```

   **Viewing and querying:**
   ```bash
   dodder show :z                     # all zettels
   dodder show one/uno                # specific zettel
   dodder show -format text one/uno   # detailed view
   dodder show tag-name:z             # by tag
   ```

   **Editing:**
   ```bash
   dodder edit one/uno                # open in editor
   dodder checkin one/uno             # save changes to store
   ```

   **Checkout/working copy:**
   ```bash
   dodder checkout :z                 # sync all to filesystem
   dodder checkout one/uno            # sync specific zettel
   dodder status                      # show checked-out state
   ```

   **Organizing:**
   ```bash
   dodder organize :z                 # bulk edit (text UI)
   dodder organize -mode output-only  # preview only
   ```

   **Remote sync:**
   ```bash
   dodder clone new-id remote:///path # clone repository
   dodder push remote-id              # push to remote
   dodder pull remote-id              # pull from remote
   ```

4. **Query syntax overview** — brief table of sigils and genres, point to
   `references/querying.md` for full syntax.

5. **Working copy vs store** — `.dodder/` is the store, filesystem is the
   working copy. `checkout` syncs store to filesystem, `checkin` syncs
   filesystem to store.

6. **Tags and types** — tags organize (like labels), types define format (like
   file extensions). Both are first-class objects with their own IDs.

7. **Reference pointers** — explicit mentions of reference files.

**Step 2: Verify word count**

Run: `wc -w skills/dodder-usage/SKILL.md`
Expected: 1,800-2,200 words

**Step 3: Commit**

```bash
git add skills/dodder-usage/SKILL.md
git commit -m "feat: write dodder-usage skill"
```

---

### Task 7: Write dodder-usage references

**Files:**
- Create: `skills/dodder-usage/references/commands.md`
- Create: `skills/dodder-usage/references/querying.md`

**Step 1: Write commands.md**

Command reference organized by workflow:

- **Repository management**: `init`, `clone`, `deinit`, `info`, `info-repo`
- **Workspace**: `init-workspace`, `info-workspace`, `status`, `clean`
- **Creation**: `new` (all flags: `-edit`, `-count`, `-tags`, `-type`,
  `-description`, `-shas`, `-filter`, `-delete`, `-organize`)
- **Viewing**: `show` (all flags: `-format`, `-before`, `-after`, `-repo`),
  `cat`
- **Editing**: `edit` (flags: `-mode`, `-delete`, `-organize`), `checkin`
  (aliases: `add`, `save`; flags: `-ignore-blob`, `-each-blob`, `-delete`,
  `-organize`)
- **File operations**: `checkout`, `organize` (flags: `-mode`, `-prefix-joints`,
  `-refine`, `-filter`)
- **Remote sync**: `push`, `pull`, `pull-blob-store`, `clone`, `remote-add`
- **Maintenance**: `reindex`, `fsck`, `repo-fsck`, `find-missing`, `diff`,
  `revert`, `last`
- **Dormant objects**: `dormant-add`, `dormant-edit`, `dormant-remove`
- **Global flags**: `-dir-dodder`, `-ignore-workspace`, `-debug`,
  `-predictable-zettel-ids`, `-comment`

Each command should have: name, description, key flags, one-line example.

**Step 2: Write querying.md**

Full query syntax reference:

- **Basic selection**: `one/uno` (exact), `:` (all), `:z` (zettels), `:z,t,e`
  (multiple genres)
- **Genres**: `z` (zettel), `t` (tag), `e` (type), `r` (repo), `b`
  (inventory_list), `konfig`
- **Sigils**: `:` (latest), `:?` (including dormant), `:+` (including external)
- **Filtering by tag**: `tag-name:z` (zettels with tag)
- **Filtering by type**: `!md:z` (zettels of type md)
- **Compound queries**: `one/uno,tag-3,!md`
- **Practical examples**:
  - "Show all markdown zettels": `dodder show !md:z`
  - "Show zettels tagged 'project'": `dodder show project:z`
  - "Show hidden zettels": `dodder show :?z`
  - "Export types and konfig": `dodder export :t,konfig`

**Step 3: Commit**

```bash
git add skills/dodder-usage/references/
git commit -m "feat: add usage skill reference docs"
```

---

### Task 8: Validate plugin structure

**Step 1: Verify all files exist**

Check the complete file tree:

```bash
ls -R .claude-plugin/ skills/
```

Expected:
```
.claude-plugin/:
plugin.json

skills/:
dodder-development/  dodder-onboarding/  dodder-usage/

skills/dodder-development/:
SKILL.md  references/

skills/dodder-development/references/:
nato-hierarchy.md  pool-management.md  testing-workflow.md

skills/dodder-onboarding/:
SKILL.md  references/

skills/dodder-onboarding/references/:
concepts.md

skills/dodder-usage/:
SKILL.md  references/

skills/dodder-usage/references/:
commands.md  querying.md
```

**Step 2: Validate each SKILL.md has proper frontmatter**

Check each skill has `name:` and `description:` in YAML frontmatter. Verify
descriptions use third person and include specific trigger phrases.

**Step 3: Validate plugin.json references valid skill paths**

Each path in the `skills` array should point to a directory containing
`SKILL.md`.

**Step 4: Verify writing style**

Spot-check that SKILL.md bodies use imperative/infinitive form (verb-first),
not second person ("you should").

**Step 5: Commit any fixes**

If validation reveals issues, fix and commit.
