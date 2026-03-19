---
status: experimental
date: 2026-03-15
promotion-criteria: BATS tests pass for init-workspace -experimental-repo, push from workspace to parent, pull from parent to workspace with query filtering and edge reachability; existing lightweight workspaces work unchanged
---

# Workspace-as-Repo

## Problem Statement

Workspaces today are lightweight views into a parent repo — a
`.dodder-workspace` config file with a query filter and defaults, plus a
filesystem store for checked-out files. They have no independent commit history.
Every mutation happens in the parent repo's store.

This limits workspaces to ephemeral working copies. They cannot diverge from the
parent, track their own history, or sync selectively. There is no way to work on
a subset of a repo's objects with an independent commit timeline and later
reconcile changes.

Additionally, workspaces are locked to filesystem checkout (`store_fs`). There is
no mechanism to back a workspace with an alternative checkout store (browser,
CalDAV, WebDAV, git, etc.) that might better match the objects' domain.

Finally, agents and automated tools that mutate workspace objects have no
isolation boundary — changes propagate directly to the parent repo's store. There
is no review gate between agent mutations and the canonical data, risking data
loss from unreviewed automated changes.

## Design

### Overview

A new experimental workspace type backed by a full dodder repo. Created via
`init-workspace -experimental-repo`, it produces a CWD-rooted repo (`.dodder/`
directory) that clones a filtered subset of the parent repo's objects. The
workspace-repo has its own store, inventory lists, signing key, and commit
history.

Push and pull between workspace and parent use the `-direct` mechanism from
[FDR 0004](0004-bindingless-local-repo-transfer.md) — no stored remote object
required.

Three design goals beyond independent commit history:

1. **Pluggable checkout stores.** The workspace-repo's checkout store is
   configurable at init time (`store_fs` by default). Alternative stores
   (`store_browser`, CalDAV, WebDAV, git, etc.) can present the same objects
   through a domain-appropriate interface.
2. **Agent isolation.** Workspace-repos act as a scoping boundary for automated
   mutations. Agents interact with the workspace-repo, not the parent. Changes
   do not propagate upstream until a user explicitly pushes, providing a review
   gate that prevents unreviewed data loss.
3. **Fast divergence detection.** The workspace-repo must be able to quickly
   answer "did anything change since upstream?" without a full inventory
   comparison — enabling efficient status checks and review workflows.

### Identity

Workspace-repo identity is the pathname. There is a 1-to-1 mapping between
workspaces and XDG location types + paths. No separate repo ID is generated.

### Init Flow

`init-workspace -experimental-repo` with a query:

1. Resolve the parent repo (ancestor discovery or current repo context)
2. Create a CWD-rooted repo at the workspace path (`.dodder/` directory)
3. Store the parent's absolute path in the workspace config
4. Clone from the parent using `-direct`, filtered to the provided query
5. Pull in reachable objects (see Edge Reachability below)

### Query Filtering

Filtering applies at **clone time** and **pull time**, but **not at push time**.

- **Clone/Pull:** Only objects matching the workspace's query are transferred
  from parent to workspace. Additionally, objects reachable via edges from
  matching objects are included (see Edge Reachability).
- **Push:** All objects in the workspace-repo are pushed to the parent. No
  filtering — the workspace may have created objects outside the original query
  scope, and those should propagate back.

### Edge Reachability

Objects that do not match the query but are reachable via object references from
matching objects are also pulled in. This ensures referential integrity — a
zettel's tags, types, and other referenced objects travel with it.

**Depth limit:** 5 (maximum traversal depth from a matching object).

> **Open question:** The depth limit of 5 is a starting point. Future work
> should explore making this configurable (e.g. `-reachability-depth N`) and
> validating whether 5 is sufficient for real-world object graphs.

### Commit History

When pushing from workspace to parent, the parent **absorbs the workspace's
commits** into its own history (merge-style integration).

> **Open question:** Alternative approaches should be explored:
> - Object-only transfer (no commit history, parent makes its own commit)
> - Selective commit transfer (choose which workspace commits to push)
> - Rebase-style integration (replay workspace commits on top of parent HEAD)

### Checkout Store

The workspace-repo's checkout store is specified at init time and recorded in the
workspace config. Default is `store_fs` (filesystem checkout, current behavior).

Alternative stores present the same objects through a domain-appropriate
interface. The blob store and inventory lists are always local (`.dodder/`), but
the checkout representation varies:

- `store_fs` — files on disk (default)
- `store_browser` — browser-based UI
- CalDAV, WebDAV, git — domain-specific sync targets

The checkout store is orthogonal to the blob store and transfer protocol. Objects
are stored and transferred the same way regardless of how they are checked out.

> **Open question:** The store interface contract needs specification (likely a
> separate RFC). For the experimental phase, only `store_fs` is implemented.

### Zettel ID Provider Linking

Workspace-repos automatically discover and use the parent repo's zettel ID word
lists (Yin/Yang) when `-yin`/`-yang` flags are not explicitly provided. During
`init-workspace -experimental-repo`, the init flow:

1. Resolves the parent path to absolute
2. Checks for Yin/Yang flat files at the parent's known object\_ids directory
3. If found, passes them to genesis via `BigBang.Yin`/`BigBang.Yang`
4. Genesis copies the words into the workspace's own store (as blobs and flat
   files), creating an independent zettel ID index

The workspace's zettel ID index is then populated by:
- **Reset** — marks all `Yin × Yang` coordinate pairs as available
- **Pull** — each imported zettel calls `AddZettelId`, removing its ID from the
  available pool
- **CreateZettelId** — picks from remaining available IDs

This means workspace repos can create new zettels out of the box without the
user managing separate word lists.

**Collision risk with filtered clones:** When a workspace is cloned with a tag
filter (e.g. `project-alpha:z`), it only knows about the pulled zettels' IDs.
IDs used by unpulled parent zettels remain in the workspace's available pool,
creating collision risk on push. Unfiltered clones (`+zettel,typ,etikett`) avoid
this by syncing all parent zettel IDs.

**Explicit flags override:** If `-yin`/`-yang` are explicitly provided, the
parent's word lists are not used. This preserves backward compatibility and
allows workspace repos with custom ID spaces.

### Agent Isolation

The workspace-repo is the mutation boundary for agents and automated tools.
Agents read from and write to the workspace-repo. Changes accumulate in the
workspace's commit history and do **not** propagate to the parent until a user
explicitly pushes.

This provides a review gate: the user can inspect workspace changes (via
divergence detection or diff) before pushing to the parent. If agent mutations
are unwanted, the workspace can be reset or deleted without affecting the parent.

### Divergence Detection

The workspace-repo must quickly answer: "did anything change since upstream?"

The workspace config stores the parent's commit hash at the time of the last
pull or clone. Comparing the workspace's current HEAD against this stored
baseline identifies local changes. Comparing the stored baseline against the
parent's current HEAD identifies upstream changes.

This enables three states:
- **Clean:** workspace HEAD == baseline, parent HEAD == baseline
- **Local changes:** workspace HEAD != baseline
- **Upstream changes:** parent HEAD != baseline (pull needed)
- **Both diverged:** both differ from baseline (pull then push, or merge)

> **Open question:** Whether this should be a dedicated command
> (e.g. `workspace status`) or integrated into existing `show`/`status` output.

### Parent Discovery

The parent repo's absolute path is stored in the workspace config at init time.
Push and pull use this stored path with `-direct` — the user does not need to
specify the parent path on each operation.

## Interface

### CLI Surface

**Modified command:** `init-workspace` gains `-experimental-repo` flag.

```sh
# Create a repo-backed workspace filtered to a query
dodder init-workspace -experimental-repo -direct /path/to/parent \
  workspace-id project-alpha:z

# Push workspace changes back to parent (implicit parent discovery)
cd workspace-dir
dodder push

# Pull parent updates into workspace (implicit parent discovery)
cd workspace-dir
dodder pull

# Explicit -direct overrides stored parent path
dodder push -direct /other/path
dodder pull -direct /other/path
```

### Implicit Parent Transfers

When push/pull are run inside a workspace-repo with a V1 config, the stored
`ParentPath` is automatically used as the `-direct` target if no remote is
explicitly specified. This is implemented via `ResolveImplicitDirectPath` which
reads the parent path from the workspace config after the local working copy is
initialized.

The resolution is a no-op when:
- `-direct` is explicitly provided (explicit overrides implicit)
- The workspace has no V1 config (lightweight workspaces are unaffected)
- The stored parent path is empty

### Error Handling

- If the workspace path already contains a `.dodder/` directory, init fails with
  a clear error (repo already exists)
- If the parent repo path becomes invalid (moved/deleted), push/pull fail with
  `ErrNotInDodderDir` pointing to the stale path
- If the query matches no objects in the parent, the workspace-repo is created
  empty (not an error — objects may be created locally and pushed later)

## Implementation Status

### What's Built

All core workspace-repo functionality is implemented and tested:

- **`init-workspace -experimental-repo`** — single command creating a CWD-rooted
  repo, cloning filtered objects from parent, writing V1 workspace config
- **Implicit parent transfers** — push/pull auto-resolve stored parent path
  when no explicit `-direct` is provided
- **Edge expansion on filtered pull** — types, tags, and referenced objects
  reachable from matching objects are included (depth limit 5)
- **Query filtering** — clone/pull respect query filter; push transfers all
  objects unfiltered
- **Error cases** — existing repo detection, stale parent path, empty query
- **Divergence detection** — `check-workspace dirty` compares sync baseline
  (TAI + object digest stored in V1 config at clone/pull/push time) against
  current inventory list state. Local-only, no parent repo access. Exit codes:
  0 = dirty, 1 = clean, 2 = not in a workspace-repo. Designed for shell prompt
  use (quiet by default).
- **Zettel ID provider linking** — workspace repos automatically discover and
  copy the parent's Yin/Yang word lists when `-yin`/`-yang` are not explicitly
  provided, enabling zettel creation without separate word list management

### What's NOT Built

These are described in the Design section but not yet implemented:

- **Pluggable checkout stores** — only `store_fs` exists
- **Agent isolation workflows** — no structured review UI beyond manual
  inspection

### Key Files

| File | Purpose |
|------|---------|
| `go/internal/victor/commands_dodder/init_workspace.go` | `InitWorkspace` command with `runLightweight` and `runExperimentalRepo` paths |
| `go/internal/uniform/command_components_dodder/remote.go` | `ResolveImplicitDirectPath` — reads parent path from workspace config |
| `go/internal/echo/workspace_config_blobs/v1.go` | V1 config struct with `ParentPath`, `SyncTai`, `SyncDigest` fields |
| `go/internal/echo/workspace_config_blobs/main.go` | `ConfigWithParentPath`, `ConfigWithSyncBaseline` interfaces |
| `go/internal/echo/workspace_config_blobs/io.go` | Coder registration for V0 and V1 |
| `go/internal/november/env_workspace/main.go` | `GetParentPath()`, `CreateWorkspace` with V0/V1 type selection |
| `go/internal/bravo/ids/types_builtin.go` | `TypeTomlWorkspaceConfigV1` constant |
| `go/internal/victor/commands_dodder/pull.go` | Wires `ResolveImplicitDirectPath` |
| `go/internal/victor/commands_dodder/push.go` | Wires `ResolveImplicitDirectPath` |
| `go/internal/victor/commands_dodder/check_workspace.go` | `check-workspace dirty` command with exit-code-based status |
| `zz-tests_bats/current_version/workspace_repo.bats` | 10 integration tests covering all workspace-repo scenarios |
| `zz-tests_bats/current_version/check_workspace_dirty.bats` | 5 integration tests for `check-workspace dirty` |

### Key Types and Interfaces

- **`workspace_config_blobs.V0`** — original workspace config (query, defaults).
  Used by lightweight workspaces.
- **`workspace_config_blobs.V1`** — embeds V0, adds `ParentPath`, `SyncTai`,
  `SyncDigest` strings. Used by workspace-repos created with
  `-experimental-repo`.
- **`workspace_config_blobs.ConfigWithParentPath`** — interface with
  `GetParentPath() string`. Used to type-switch between V0/V1 in
  `CreateWorkspace` and `GetWorkspaceConfigTyped`.
- **`command_components_dodder.Remote.ResolveImplicitDirectPath`** — called after
  `MakeLocalWorkingCopy` in push/pull. Reads parent path from workspace config,
  sets `DirectPath` if no explicit `-direct` was provided.

### Gotchas

**Query storage vs. clone filter.** Positional args to `init-workspace
-experimental-repo` (e.g. `project-alpha:z` or `+zettel,typ,etikett`) are the
**initial clone filter only**. They are NOT stored as the workspace's default
query. The workspace default query comes from the `-query` flag (stored as
`workspace_config_blobs.V0.Query`). If positional args were stored as the
default query, push would filter outgoing objects through that query, causing it
to transfer zero objects when the query doesn't match workspace-created content.

**`ExcludeDefaultType` on Genesis.** `runExperimentalRepo` sets
`cmd.Genesis.BigBang.ExcludeDefaultType = true` before calling `OnTheFirstDay`.
Without this, the default `!md` type would be auto-created in the new repo
before the pull, potentially conflicting with the pulled type objects.

**Zettel ID conflicts with filtered clones.** When a workspace-repo is cloned
with a tag filter (e.g. `project-alpha:z`), it doesn't know about the parent's
full zettel ID space. Creating new zettels in the workspace may assign IDs that
already exist in the parent, causing conflicts on push. The unfiltered clone
(`+zettel,typ,etikett`) avoids this by syncing the full ID index.

## What Does NOT Change

- Lightweight workspace behavior (`init-workspace` without `-experimental-repo`)
- Transfer protocol (inventory list exchange, blob copying)
- `-direct` flag behavior (FDR 0004)
- Stored-remote workflow (`remote-add` + push/pull by repo-id)
- Object format, blob format, commit format

## Exploration: Workspace-Repos as the Only Checkout Mechanism

Today dodder has two distinct concepts for materializing objects:

1. **Checkout stores** (`checkout`, `checkin`) — the repo writes objects into a
   filesystem store (`store_fs`) and reads them back. The checkout store is a
   mutable view directly coupled to the parent repo's inventory. Changes
   propagate immediately on `checkin`.
2. **Workspace-repos** (`init-workspace -experimental-repo`, `push`, `pull`) — a
   full repo that pulls a filtered subset from a parent and pushes changes back.
   The workspace-repo has its own store, commit history, and isolation boundary.

These two concepts overlap significantly. Both answer the question "how do I
work with a subset of a repo's objects?" The checkout store answers it with
tight coupling and immediate propagation. The workspace-repo answers it with
isolation and explicit sync.

### What if checkout stores were workspace-repos?

Every `checkout` would become `init-workspace -experimental-repo` (or its
eventual non-experimental successor). Instead of materializing files from a
repo's store into a sibling directory, you'd create a workspace-repo that pulls
the objects you want to work with. `checkin` becomes `push`. `checkout` becomes
`pull`.

**What this unifies:**

- **One sync model.** Today `checkin`/`checkout` uses a different code path than
  `push`/`pull`. Workspace-repos would make all object exchange go through the
  same inventory-list-based transfer protocol.
- **Automatic history.** Every checkout gets its own commit history for free.
  Today, checking in a file directly mutates the parent's commit history — there
  is no record of intermediate states in the working copy.
- **Isolation by default.** Checkout stores have no isolation boundary — a bad
  `checkin` directly corrupts the parent. Workspace-repos require an explicit
  `push`, providing a natural review gate.
- **Pluggable stores become pluggable workspace-repos.** The "alternative
  checkout store" concept (browser, CalDAV, WebDAV, git) becomes "alternative
  workspace-repo type." The workspace-repo's type determines how objects are
  presented, but the sync protocol is always push/pull. No new store interface
  needed — each presentation layer is just a different workspace-repo
  implementation.

**What this simplifies:**

- `checkout`/`checkin` commands could be sugar over `push`/`pull` with implicit
  workspace-repo creation.
- The `store_fs` code path in `env_repo` becomes a workspace-repo type rather
  than a special case wired into the repo internals.
- Organize, which currently operates on the checkout store directly, would
  operate on the workspace-repo and push results back.

**What this complicates:**

- **Performance.** A lightweight checkout is fast — it writes files and updates
  an index. A workspace-repo requires creating a full `.dodder/` directory,
  generating a signing key, and maintaining a separate inventory. For quick
  edits this is heavyweight.
- **Ergonomics for simple cases.** `dodder checkout my-zettel && vim ... &&
  dodder checkin` is three commands. The workspace-repo equivalent is five:
  init-workspace, pull, edit, push, cleanup. Sugar could hide this but adds
  conceptual weight.
- **Workspace lifecycle.** Checkout stores are implicitly cleaned up (or persist
  as stale files). Workspace-repos are full repos that accumulate history and
  consume disk. Users would need to manage workspace-repo lifecycle (create,
  sync, delete).
- **Nested repo detection.** If every checkout creates a `.dodder/` directory,
  tools that walk the filesystem (including dodder itself) need to handle nested
  `.dodder/` directories without confusion.

**Possible middle ground:**

Ephemeral workspace-repos — workspace-repos that are created on-demand, have no
persistent commit history (single-commit lifecycle), and are cleaned up after
push. This preserves the isolation benefit without the lifecycle burden. The
implementation would be a workspace-repo with a flag like `-ephemeral` that
auto-pushes and self-destructs, making the user experience identical to
`checkout`/`checkin` while using the workspace-repo machinery underneath.

### Open questions

- Does the performance cost of workspace-repo creation matter in practice, or
  is it dominated by blob transfer time?
- Can `organize` work through push/pull, or does its interactive editing model
  require direct store access?
- Would ephemeral workspace-repos actually be simpler than the current checkout
  store, or would they just be checkout stores with extra steps?
- How does this interact with `store_fs` watching (inotify/FSEvents) for
  automatic `checkin`?

## Future Possibilities

### Workspace Query Evolution

Allow the workspace's filter query to be updated after creation, triggering a
re-sync (pull new matching objects, optionally prune objects that no longer
match).

### Nested Workspaces

A workspace-repo could itself have workspaces, enabling hierarchical filtering
(e.g. organization repo → team workspace → personal workspace).

### Batch Zettel ID Reservation

To eliminate collision risk between workspace and parent, the workspace could
reserve a batch of IDs from the parent at init/pull time. The parent marks those
IDs as consumed; the workspace uses them locally. This enables offline zettel
creation with guaranteed uniqueness.

### Fast-Forward Pull for Clean Workspaces

When a parent repo pulls from a remote and the workspace has a clean copy of the
changed objects (no local modifications), the workspace could fast-forward its
local state without a full pull. This requires tracking which objects are
unmodified relative to the parent's last sync baseline.

### Agent Review Workflows

Build on the divergence detection to provide structured review of agent
mutations — e.g. a diff view showing what the agent changed, with accept/reject
per object or per commit before pushing to parent.

## Rollback Strategy

### Dual-Architecture Period

The `-experimental-repo` flag is purely additive. Existing lightweight
workspaces are completely unchanged. Both types coexist — `init-workspace`
without the flag produces the current lightweight workspace.

### Promotion Criteria (experimental → testing)

The following criteria for `experimental` status are **all met** as of
2026-03-15:

- [x] BATS tests pass for repo-backed workspace init, push, and pull
- [x] Query filtering + edge reachability produces correct object sets
- [x] Commit history merges cleanly from workspace to parent
- [x] Existing lightweight workspace tests still pass unchanged
- [x] Error messages are clear for invalid parent path, existing repo, empty query

To advance to `testing`:

- [x] Zettel ID provider linking from parent (workspace can create zettels
  without explicit `-yin`/`-yang`)
- [ ] Zettel ID conflict resolution for filtered clones — planned approach is a
  remote zettel ID index that delegates allocation to the parent's index via
  `flock(2)`, eliminating collisions entirely. Depends on
  [FDR-0006](0006-two-stage-commit.md) (two-stage commit) and
  [ADR-0001](../decisions/0001-use-flock-for-fine-grained-resource-locking.md)
  (per-resource flock locking)
- [ ] Divergence detection (workspace HEAD vs parent HEAD baseline comparison)
- [ ] Real-world validation with a non-trivial object graph (100+ objects)

### Rollback Procedure

Remove the `-experimental-repo` flag and the repo-backed workspace init path.
No migration needed — lightweight workspaces are unaffected. Repo-backed
workspace directories can be deleted manually (they are self-contained `.dodder/`
directories).
