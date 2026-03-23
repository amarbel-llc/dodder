# init-workspace: Implicit Parent Discovery

## Problem

`init-workspace -experimental-repo` requires `-direct <path>` to specify the
parent repo. This is unnecessary friction --- for now there is only one
user-scoped XDG dodder repo, and CWD-scoped parents can be specified with a
flag. The `-direct` flag is also overloaded (used by clone, push, pull for
different purposes), and the recent CWD-routing fix exposed a bug where the
command tries to `mkdir` on the root filesystem when given a bad path.

## Design

### CLI Surface

    # Home repo as parent (default):
    der init-workspace my-workspace-id [query...]

    # CWD-scoped repo as parent:
    der init-workspace -parent /path/to/repo my-workspace-id [query...]

    # Lightweight workspace (no independent store, opt-out):
    der init-workspace -experimental-repo=false my-workspace-id

### Changes

- **Flip `-experimental-repo` default** from `false` to `true`. Repo-backed
  workspaces are the default; lightweight mode is opt-out via
  `-experimental-repo=false`.
- **Remove `-direct` requirement** from the experimental-repo path.
- **Add `-parent <path>` flag** (optional). When omitted, the parent is the
  user-scoped XDG home repo. When set, the parent is a CWD-scoped repo at that
  path.
- **Error if `-repo_id` is passed** with `-experimental-repo` (default or
  explicit). Workspace repos are always CWD-rooted.
- **Hard fail if parent repo doesn't exist:**
  - No `-parent`: resolve home XDG repo, fail with "no dodder repo found; run
    `dodder init` first"
  - `-parent /path`: fail if `/path` doesn't contain a valid dodder repo

### Validation

  Combination                              Result
  ---------------------------------------- ------------------------------
  No `-parent`                             Home XDG repo is parent
  `-parent /path`                          CWD-scoped repo at path
  `-repo_id` + `-experimental-repo`        Hard fail
  `-experimental-repo=false` + `-parent`   Hard fail
  Parent repo missing                      Hard fail with clear message

### Parent Zettel ID Discovery

`linkParentZettelIdProviders` needs two code paths:

- **Home repo parent**: yin/yang at `~/.local/share/dodder/object_ids/`
- **CWD-scoped parent**: yin/yang at `<path>/.dodder/local/share/object_ids/`
  (existing logic)

### Workspace Config (V1 blob)

`ParentPath` field semantics:

- **Empty** → parent is the home XDG repo (resolved at runtime)
- **Set** → parent is a CWD-scoped repo at that absolute path

`ResolveImplicitDirectPath` in the remote component handles the empty case by
resolving the home repo path at runtime, so push/pull without `-direct` works
for both parent types.

### Rollback

The `-experimental-repo=false` flag is the escape hatch. No dual-architecture
period needed --- the feature is already marked experimental.

## Files Changed

1.  `go/internal/victor/commands_dodder/init_workspace.go` --- flip default, add
    `-parent` flag, remove `-direct` requirement, add validation, update
    `linkParentZettelIdProviders`
2.  `go/internal/uniform/command_components_dodder/remote.go` ---
    `ResolveImplicitDirectPath` handles empty `ParentPath`
3.  `zz-tests_bats/current_version/workspace_repo.bats` --- update tests: remove
    `-direct`, remove `-repo_id .`, add home-repo and CWD-parent variants
