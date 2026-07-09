---
status: proposed
date: 2026-07-09
promotion-criteria: a working `edit -ephemeral` / `checkout -ephemeral` path exists in the CLI with BATS coverage — temp repo-backed workspace created, object checked out + edited, pushed to the resolved repo, torn down on success, and the temp repo preserved on push failure (proposed -> experimental); the dodder Alfred workflow's edit path switched to `-ephemeral` and exercised end-to-end (experimental -> testing)
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

A `-ephemeral` boolean flag on `edit` (and `checkout`). When set, the command
does not require — or discover — a persistent `.dodder-workspace`. Instead, for
a resolvable target repo (cwd-ancestor `.dodder`, an explicit `-parent`/`-direct`
path, or the home repo), it:

1. materializes an **ephemeral** repo-backed workspace in a temp dir, blob
   storage pointing at the resolved repo (the FDR-0005 `TomlPointerV1` — no blob
   copy),
2. checks out the queried objects into it,
3. opens them in the editor,
4. on editor exit, commits and **pushes** the changes back to the resolved repo,
   then
5. tears the ephemeral workspace down — **only** on push success; on push
   failure the temp repo is preserved and its path surfaced, so no edit is lost.

`-ephemeral` is opt-in: it changes nothing about the existing in-workspace
`edit`/`checkout` path, and it does not replace today's temporary-workspace
offer-to-create prompt for the non-`-ephemeral` case. This keeps the blast
radius small and sidesteps the perf question (a genesis + filtered pull per
invocation) that would gate making it the default.

**First consumer — the dodder Alfred workflow.** The workflow
(`zz-alfred/`) currently `cd`s into a single baked workspace before invoking
`edit`, which fails for users with several workspaces under a parent dir
([#340](https://github.com/amarbel-llc/dodder/issues/340)). Switching its edit
triggers to `edit -ephemeral` against a resolved repo removes the baked-workspace
requirement: the launcher can edit any object from anywhere. The repo the
ephemeral workspace targets still has to be chosen when several are reachable —
that selection UX is #340; this flag is the mechanism it will build on.

## Examples

    # today, from a dir with no .dodder-workspace (non-interactive):
    $ dodder edit some/zettel
    error: not in a workspace          # offer-to-create prompt can't be answered

    # with -ephemeral, from anywhere with a resolvable repo:
    $ dodder edit -ephemeral some/zettel
    # → temp repo-backed workspace created (blob pointer to the resolved repo),
    #   some/zettel checked out, editor opens; on exit the change is committed +
    #   pushed to the repo and the temp workspace is removed.

    # push failure preserves the work rather than discarding it:
    $ dodder edit -ephemeral some/zettel
    # ... edit ..., push conflicts →
    # error: push failed; ephemeral workspace kept at /tmp/dodder-ephemeral-XXXX

    # the Alfred workflow's edit trigger becomes (illustrative):
    #   ./run.bash edit -ephemeral -parent <resolved-repo> "$@"
    # so it no longer needs to cd into a single baked workspace.

## Limitations

- **Scope.** This is deliberately narrower than FDR-0005's "checkout stores as
  workspace-repos" unification. It targets only the "edit/checkout from outside
  any workspace" flow; it does not propose replacing the lightweight checkout
  path for in-workspace use.
- **Repo selection is unsolved here.** *Which* repo an ephemeral workspace pushes
  to, when several are reachable, is the open UX problem tracked in
  [#340](https://github.com/amarbel-llc/dodder/issues/340). This FDR assumes a
  resolved repo; it does not specify the selection mechanism.
- **Open semantics (status `proposed`, not yet built).** The surface is decided
  (`-ephemeral` on `edit`/`checkout`), but the single-commit push semantics
  (does the parent absorb the workspace commit or make its own — FDR-0005
  "Commit History" open question) and the exact push-failure landing behavior
  are still to be pinned in implementation. FDR-0005's perf open question —
  whether an ephemeral repo is cheaper than the current checkout store or just
  "checkout stores with extra steps" — is why this is opt-in rather than the
  default fallback.

## Implementation Notes

> Grounded in the current `runExperimentalRepo` path
> (`commands_dodder/init_workspace.go`). Recorded to show the mechanism is a
> lifecycle wrapper over existing pieces, not new store machinery — but the
> surface and the teardown/rollback semantics remain open (status `exploring`).

The durable repo-backed-workspace init already performs every step an ephemeral
workspace needs; the only additions are (a) a temp-dir root and (b) a
teardown. Mapping today's `runExperimentalRepo` onto an ephemeral lifecycle:

| Ephemeral step | Existing mechanism (init_workspace.go) |
|---|---|
| resolve target repo | `resolveParentPath` / `makeParentRemote` (home repo or `-parent` path) |
| root the workspace in a temp dir | today it is CWD-rooted (`repo_id.CwdDefault()`, `OnTheFirstDay`); ephemeral would root at a `mktemp -d` instead |
| share the target's blobs | `setupParentPointerBlobStore` writes a `TomlPointerV1` (no blob copy) — reused verbatim |
| pull the objects to edit | `PullQueryGroupFromRemote` with the query = the object(s) being edited |
| edit | existing `edit`/`checkout` `MakeCheckout` path, now against the temp workspace |
| push back | the FDR-0005 implicit-parent `push` (`ResolveImplicitDirectPath`, stored `ParentPath` + `ParentPubkey`) |
| tear down | **new** — `rm -rf` the temp repo after a successful push |

Open mechanism questions this raises (beyond the surface choice):

- **Push-failure rollback.** If the push conflicts or fails, the temp workspace
  holds the only copy of the edit. Teardown must be gated on push success, and
  the failure path needs a durable landing spot (keep the temp repo? surface its
  path?) rather than silently discarding work.
- **Single-commit lifecycle.** FDR-0005 frames ephemeral repos as
  single-commit; whether the parent absorbs that commit or makes its own
  (FDR-0005 "Commit History" open question) applies here unchanged.
- **Cost.** Each invocation does a genesis (`.dodder/` + signing key) + filtered
  pull. FDR-0005's open question — whether this is cheaper than the current
  checkout store or "checkout stores with extra steps" — is the gating perf
  question for making this the *default* fallback vs. an opt-in.

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
