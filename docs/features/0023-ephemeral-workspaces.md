---
status: exploring
date: 2026-07-09
promotion-criteria: solution direction selected and a full Interface/Examples draft exists (exploring -> proposed); a working `edit`/`new`-from-anywhere path exists in the CLI with BATS coverage (proposed -> experimental)
---

# Ephemeral Workspaces

## Problem Statement

`edit` and `checkout` refuse to run outside a persistent workspace: when no
`.dodder-workspace` is found, dodder falls back to a *temporary workspace* that
`workspace(7)` defines as **read-only**, and the write commands
`AssertNotTemporaryOrOfferToCreate` — which, non-interactively (e.g. an Alfred
script, an MCP call, a hook), simply fails with "not in a workspace" because the
create-a-workspace prompt has no TTY to answer. This blocks the common
"edit any object from anywhere against a known repo" flow: a user who keeps
several workspaces under a parent directory, or who invokes dodder from a launcher
with no cwd inside any workspace, cannot check out and edit an object without
first `cd`-ing into (or hand-creating) a durable workspace.

The building blocks already exist. `new` does *not* require a workspace (it
writes objects to the repo store directly). The repo-backed workspace
([FDR-0005](0005-workspace-as-repo.md)) gives a workspace its own repo + history
that propagates to a parent only on `push`, and FDR-0005's "Possible middle
ground" section already names the missing piece: *ephemeral workspace-repos* —
"created on-demand, have no persistent commit history (single-commit lifecycle),
and are cleaned up after push ... making the user experience identical to
`checkout`/`checkin` while using the workspace-repo machinery underneath." This
FDR promotes that exploration into a focused, implementable feature scoped to the
"operate on a repo from outside any workspace" use case, rather than the broader
"checkout-stores-as-workspace-repos" unification FDR-0005 also contemplates.

## Interface

> This section is a sketch, not a committed design — status is `exploring`.
> The concrete flag/command surface is an open question (see Limitations).

The intended behavior: a write command (`edit`, and by extension `checkout`)
invoked with no discoverable `.dodder-workspace`, but with a resolvable repo
(via cwd-ancestor `.dodder`, an explicit repo target, or a configured default),
should be able to:

1. materialize an **ephemeral** working copy (a repo-backed workspace in a temp
   dir, blob storage pointing at the resolved repo per FDR-0005),
2. check out the queried objects into it,
3. open them in the editor,
4. on editor exit, commit + **push** the changes back to the resolved repo, and
5. tear the ephemeral working copy down.

Candidate surfaces (undecided):

- an `-ephemeral` flag on `edit`/`checkout` that opts into the above;
- promoting it to the *default* fallback when a write command hits a temporary
  workspace and a repo is resolvable (replacing today's interactive
  offer-to-create); or
- a distinct verb (e.g. `dodder edit -repo <id>`) that never touches the
  workspace-discovery path at all.

The repo the ephemeral workspace pushes to must be selectable, since the
motivating users have *several* repos/workspaces — a single baked path is
insufficient (this is the concrete gap behind the Alfred workflow, see
[#340](https://github.com/amarbel-llc/dodder/issues/340)).

## Examples

    # today, from a dir with no .dodder-workspace (non-interactive):
    $ dodder edit some/zettel
    error: not in a workspace          # offer-to-create prompt can't be answered

    # intended (illustrative — surface undecided):
    $ dodder edit -ephemeral some/zettel
    # → temp repo-backed workspace created, object checked out, editor opens,
    #   on exit the change is pushed to the resolved repo, temp workspace removed

## Limitations

- **Scope.** This is deliberately narrower than FDR-0005's "checkout stores as
  workspace-repos" unification. It targets only the "edit/checkout from outside
  any workspace" flow; it does not propose replacing the lightweight checkout
  path for in-workspace use.
- **Repo selection is unsolved here.** *Which* repo an ephemeral workspace pushes
  to, when several are reachable, is the open UX problem tracked in
  [#340](https://github.com/amarbel-llc/dodder/issues/340). This FDR assumes a
  resolved repo; it does not specify the selection mechanism.
- **Not yet designed.** Status is `exploring`: the flag/command surface, the
  single-commit push semantics, failure/rollback on a push conflict, and cleanup
  guarantees are all open. FDR-0005's own open questions (lines under "Possible
  middle ground" — whether ephemeral workspace-repos are simpler than the current
  checkout store, or "checkout stores with extra steps") apply directly.

## More Information

- [FDR-0005 Workspace-as-Repo](0005-workspace-as-repo.md) — origin of the
  "ephemeral workspace-repos" middle-ground this FDR promotes; the repo-backed
  workspace machinery (`init-workspace -experimental-repo`, blob pointer,
  implicit parent push/pull) is the substrate.
- [FDR-0004 Bindingless Local Repo Transfer](0004-bindingless-local-repo-transfer.md)
  — the `-direct` push/pull mechanism an ephemeral workspace would use to
  reconcile with its parent.
- `workspace(7)` — documents the temporary-workspace read-only contract this
  feature works around.
- [dodder#340](https://github.com/amarbel-llc/dodder/issues/340) — the motivating
  Alfred-workflow UX gap (dynamic workspace/repo selection) that surfaced this.
