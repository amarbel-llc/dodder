---
status: exploring
date: 2026-03-15
promotion-criteria: BATS tests pass for direct push, pull, and clone between two local repos without a stored remote object; existing remote-add-based push/pull/clone still works unchanged
---

# Bindingless Local Repo Transfer

## Problem Statement

Push, pull, and clone currently require either a stored remote object (created
via `remote-add`) or, in clone's case, a remote type + connection arg that still
fetches the remote's config and public key (binding/handshake). There is no way
to do ad-hoc transfers between two local repos without first persisting a
tracked remote object.

This matters because the end goal is to unify workspaces and repos. Workspaces
will be backed by full repos filtered to the query provided at initialization
time, enabling independent commit history. Push/pull between a workspace-repo
and its parent must be lightweight and not require pre-registration — the
relationship is implicit (parent contains child directory).

## Interface

### CLI Surface

**New flag:** `-direct <path>` on `push`, `pull`, and `clone`.

When set:
- `<path>` is resolved to an absolute path
- A `TomlLocalOverridePathV0` blob is constructed inline with the resolved path
  and no public key
- `MakeRemoteFromBlob` is called (not `MakeRemoteFromBlobAndSetPublicKey` — no
  config fetch or key exchange)
- Push/pull skip the `repo-id` positional arg entirely
- Clone skips `MakeRemoteAndObject` and its type-arg pop

When not set: existing behavior is unchanged.

**Examples:**

```sh
# Pull from a local repo
dodder pull -direct /path/to/other/repo

# Push to a local repo
dodder push -direct /path/to/other/repo

# Clone a local repo
dodder clone -direct /path/to/source/repo new-repo-id
```

### Error Handling

If the target path does not contain an initialized dodder repository, the
existing `ErrNotInDodderDir` from `env_repo.Make` is wrapped with context:
"direct remote at `<path>`: not in a dodder directory."

No separate pre-flight check — the construction chain already validates repo
existence.

## Implementation

### Changes to Existing Code

**`command_components_dodder.Remote`** — Add a `DirectPath` string field.
Register `-direct` flag in `SetFlagDefinitions`. Add
`MakeDirectRemoteFromPath(req, local)` method that:
1. Resolves the path to absolute
2. Constructs `repo_blobs.TomlLocalOverridePathV0{OverridePath: absPath}`
3. Calls `MakeRemoteFromBlob`
4. Wraps any `ErrNotInDodderDir` with `-direct` context

**`push.go`** — Branch on `DirectPath != ""`: if set, call
`MakeDirectRemoteFromPath`; otherwise, existing `GetObjectFromObjectId` +
`MakeRemote` path.

**`pull.go`** — Same branching pattern.

**`clone.go`** — Same branching pattern, skipping `MakeRemoteAndObject`.

### What Does NOT Change

- Transfer protocol (inventory list exchange, blob copying)
- `remote_http` client/server
- `repo_blobs` types
- `env_repo` initialization or validation
- Stored-remote workflow (`remote-add` + push/pull by repo-id)

## Future Possibilities

### Positional Arg Path Detection

Instead of `-direct <path>`, detect that the argument is a filesystem path
(starts with `/`, `.`, or `~`) and construct the blob inline. This removes the
flag but adds parsing ambiguity with existing positional args (repo-ids, query
terms). Deferred until the ergonomics of `-direct` are validated.

### Workspace-as-Repo

The motivating use case. `init-workspace` would create a full dodder repo
(with its own store, inventory lists, signing key, commit history) filtered to
the query provided at initialization time. The workspace repo lives inside the
parent repo's working directory. Checkin/checkout between workspace and parent
become push/pull using the `-direct` (or its successor) mechanism.

This gives workspaces independent commit history while maintaining the
parent-child relationship through local transfers.

## Rollback Strategy

### Dual-Architecture Period

The `-direct` flag is purely additive. Existing remote-add-based push/pull/clone
is completely unchanged. Both paths coexist — users choose which to use per
invocation.

### Promotion Criteria

- BATS tests pass for direct push, pull, and clone between two local repos
- Existing remote-add-based tests still pass unchanged
- Error message is clear when target path is not an initialized repo

### Rollback Procedure

Remove the `-direct` flag registration, the `DirectPath` field, and the
`MakeDirectRemoteFromPath` method. No persistent state is affected — `-direct`
is a runtime mechanism, nothing is written to disk.
