---
status: proposed
date: 2026-06-07
promotion-criteria: repo-id grammar (name + scope prefix + dot-depth)
  parses and round-trips in madder env_dir with dodder consuming it;
  on-disk layout matches madder's scoped layout (repos/<name>/ under
  each scope) with legacy single-repo trees readable as the `default`
  repo; all BATS tests pass with the new resolution; new tests cover
  user-scoped named repos, two same-named CWD repos disambiguated by
  dot-depth, and MCP repo_id addressing; MCP resource URIs accept the
  repo segment and the CWD-auto sugar
---

# Scoped Repo Resolution

## Problem Statement

Dodder can address at most one repo per location: the XDG user tree
or a `.dodder/` in CWD, selected by a location-only `-repo_id` (`.`,
`/`, or empty — FDR-0003). There is no name portion, so a user cannot
keep several user-scoped repos side by side, cannot address a parent
directory's repo from a child checkout, and the MCP server is bound
to exactly one repo for its whole lifetime.

Madder already solved this shape of problem for blob stores: the ID
itself does the scoping (`default` = XDG user, `.archive` = CWD,
`/system` = XDG system), repeated leading dots walk up to ancestor
directories (`..default` is the parent's store), and any number of
named stores coexist per scope. Repos should follow the same
strategy — same grammar, same scoping, and a matching on-disk
layout — so the two ID systems feel like one.

## Interface

### Repo ID Grammar

A repo id is an optional scope prefix followed by a name. The name
charset is `[a-zA-Z0-9_-]+`, identical to madder blob-store ids.

| Form | Scope | Resolution |
|---|---|---|
| *(empty)* | Auto | Nearest CWD-scoped repo on the walk-up; otherwise the user-scoped repo named `default` |
| `name` | XDG user | `$XDG_DATA_HOME/dodder/repos/<name>/` |
| `.name` | CWD | Nearest ancestor `.dodder/` tree containing a repo named `name` |
| `..name` | CWD (depth 1) | Second-nearest ancestor `.dodder/` tree containing a repo named `name` |
| `/name` | Remote, then system | A remote named `name` defined in the current (auto-resolved) repo; if no such remote exists, the system-scoped repo named `name` |
| `//name` | XDG system | System data dir, `dodder/repos/<name>/` — explicit, never a remote |

Rules carried over from madder's `blob_store_id`:

- **Dot-depth disambiguation.** N leading dots = the Nth-nearest
  ancestor (1-indexed) whose `.dodder/` tree holds a repo with that
  name. This is how two same-named CWD-scoped repos in nested
  directories are told apart. The walk-up is bounded by the existing
  ceiling-directory mechanism.
- **Canonical form is single-dot.** Anything persisted (configs,
  pointers, wire formats) collapses dot-depth to one dot, so stored
  references stay stable across CWD changes. Depth is a runtime
  CLI/MCP rendering and input concern only (madder #145 precedent).
- **Unprefixed = user scope.** `String()` omits the prefix for
  user-scoped ids; `~name` is accepted on parse as a
  disambiguation/compat alias and never emitted.

One deliberate divergence from madder's grammar:

- **`/name` is remote-first.** Resolving `/name` first looks up the
  remotes (repo-genre objects) defined in the current repo — which
  must itself be auto-resolved first — and only falls back to the
  system-scoped repo named `name` when no matching remote exists.
  `//name` forces the system scope and never matches a remote. This
  preserves FDR-0003's reservation of `/<repo-id>` for remote
  selection while still giving system-scoped repos an unambiguous
  spelling. (In madder, `/name` is always the system store; madder
  has no remote concept to collide with.)

Two repos with the same name in different scopes (`notes` vs
`.notes`) are different repos at different filesystem locations.

### On-Disk Layout

This is a layout change to match how madder lays out blob stores.
Today a repo *is* the whole per-scope XDG tree
(`$XDG_DATA_HOME/dodder/`, or the remapped tree under `$PWD/.dodder/`).
After this change, each scope hosts N named repos under a `repos/`
level, the same way madder hosts N stores under `blob_stores/`:

| Scope | Madder blob stores (existing) | Dodder repos (new) |
|---|---|---|
| XDG user | `$XDG_DATA_HOME/madder/blob_stores/<name>/` | `$XDG_DATA_HOME/dodder/repos/<name>/` |
| CWD | `$PWD/.madder/local/share/blob_stores/<name>/` | `$PWD/.dodder/local/share/dodder/repos/<name>/` |
| XDG system | system data dir | system data dir, `dodder/repos/<name>/` |

A repo's config/cache/state trees move under the same `repos/<name>/`
keying in their respective XDG category dirs, so two named repos
share nothing.

**Legacy compatibility.** Existing single-repo trees (the current
`$XDG_DATA_HOME/dodder/` contents, and `.dodder/` trees without a
`repos/` level) are recognized on read and treated as the repo named
`default` in their scope. An explicit migration (`dodder
migrate-repo-layout`, exact name TBD) moves a legacy tree into
`repos/default/`. No silent rewriting: until migrated, legacy trees
stay readable in place.

### CLI

- `-repo_id` (global) and `DODDER_REPO_ID` accept the full grammar
  above. Today's values stay valid: `.` and `/` without a name mean
  "the `default` repo in that scope".
- `dodder init` accepts the repo id (scope + name) for the repo it
  creates; `init-default` keeps deriving the name from the directory
  basename.
- A listing surface (e.g. `dodder info-repo repos` or a dedicated
  subcommand) enumerates every repo visible from CWD with its
  disambiguated id — the repo analogue of madder's store listing.
- `complete` learns the new grammar.

### MCP

The MCP server becomes multi-repo. Today it binds one repo at
startup and neither tools nor resource URIs carry a repo dimension.

- **Tool parameter.** Every repo-touching tool gains an optional
  `repo_id` accepting the grammar above.
- **Resource URI segment.** Resources gain a repo-scoped form (the
  emitted canonical shape uses the triple-slash empty-authority form):
  `dodder:///repos/<repo-id>/objects/...`,
  `dodder:///repos/<repo-id>/query/...`, etc. The repo-id segment is
  the CLI spelling (leading dots are URI-safe). The collection root
  `dodder:///repos` lists the repos in scope (the MCP analog of
  `info-repo repos`), and `dodder:///repos/<repo-id>` is a per-repo
  overview linking its objects/types/tags/indexes.
- **CWD-auto sugar.** Omitting `repo_id` (or using the existing
  un-segmented `dodder://...` URIs) resolves exactly like the empty
  CLI id: nearest CWD-scoped repo from the server's working
  directory, else the user-scoped `default`. Existing clients keep
  working unchanged; the sugar also gives "the repo here" without
  spelling its name.

## Examples

Two same-named repos on the walk-up, from
`~/eng/repos/dodder/.worktrees/calm-willow`:

    # nearest ancestor with a repo named "notes"
    dodder show -repo_id .notes :z

    # same name, one ancestor further up
    dodder show -repo_id ..notes :z

User-scoped repos side by side:

    dodder init -repo_id work ...
    dodder init -repo_id personal ...
    dodder show -repo_id work :t

Scoping a shell session:

    export DODDER_REPO_ID=work
    dodder show :z          # hits the user-scoped "work" repo

Remote-first `/name`, explicit system with `//name`:

    dodder show -repo_id /backup :z    # remote "backup" of the current
                                       # repo if defined, else the
                                       # system-scoped repo "backup"
    dodder show -repo_id //backup :z   # always the system-scoped repo

MCP, addressing a sibling repo without restarting the server:

    read dodder:///repos/work/objects/some/zettel
    query(["!task", "todo"], repo_id: "..notes")

    # CWD-auto sugar: same as today's single-repo behavior
    read dodder://objects/some/zettel

## Limitations

- **Remote transport stays out of scope.** `/name` reserves the
  remote-first resolution order, but actually operating against a
  remote (transport, auth, caching) remains separate work — until it
  lands, a `/name` that matches a defined remote errors
  not-implemented rather than silently falling back to system.
  Remotes are unbounded in cardinality and defined as repo-genre
  objects inside repos; only their *selection* syntax is specified
  here.
- **Dot-depth, not `../`.** The parent operator is repeated leading
  dots, exactly as in madder. A literal `../name` is not part of the
  grammar: `/` is the system-scope prefix and is excluded from the
  name charset.
- **No cross-repo queries.** A command or MCP call addresses exactly
  one repo. Fan-out across repos is a client concern.
- **`ids.RepoId` (the repo genre) is untouched.** The genesis
  config's repo id, remotes, and `-kasten` continue to use the
  existing object-genre type. This FDR only changes the *location*
  selector (`env_dir.RepoId`).
- **One `default` per scope is still special.** Empty-id resolution
  needs a deterministic target; `default` is it. Renaming the
  default repo means updating `DODDER_REPO_ID` or passing ids
  explicitly.

## Implementation Status

Landed (master):

- **P1 layout + grammar.** `scoped_id`-based `-repo_id`, default-named
  repos, the `repos/<name>/` metadata nest under each scope (madder
  blob env stays flat), and `repo_id.CheckSupported` as the uniform
  scope gate on the working-copy/serve/info paths.
- **P3 user surface.** `info-repo repos` listing and `-repo_id`
  completion of the in-scope repo names.
- **MCP repo_id, Phase A (tools).** Every repo-touching tool takes an
  optional `repo_id`; the bridge resolves it per call (startup pin
  restricts, unpinned routes per-call), `CheckSupported`-gated.
- **MCP repo_id, Phase B core (resources).** The resource surface is
  repo-scoped: `dodder:///repos/<repo>/...` reads route to the
  addressed repo (bridge- and per-repo-index-routed), `dodder:///repos`
  lists the in-scope repos, `dodder:///repos/<repo>` is a per-repo
  overview, and the legacy un-segmented `dodder://...` URIs stay as
  CWD-auto sugar. The HTTP MCP surface emits the same scheme.

Deferred (tracked follow-ups):

- **RepoManager (#278).** Three MCP surfaces still hold only the
  server's startup repo — the `edit` and `reset-lock` tools and the
  blob-format *listing* (`getBlobFormatIds`, the one handler reading
  `store`/`typeBlobCoder` directly). A repo-scoped read/call of a
  non-default repo for these returns a clear "per-repo not yet
  supported (RepoManager follow-up)" error. Making them per-repo needs
  a stateful repo-open-by-id with the lazy per-call env cache described
  under Tuning Levers.
- **Both-scope repo listing (#276).** `dodder:///repos` (and
  `info-repo repos`) currently enumerate the bare directory names under
  the startup repo's scope `repos/` dir, so a CWD-scope repo is listed
  as `default` rather than its routable `.default` spelling — the
  emitted child URI does not round-trip back to the CWD repo. Listing
  both scopes with scope-correct spellings is #276.
- **P2 madder walk-up / multi-dot** and **#274** (delegate
  `EffectiveName` to a madder shared resolver, then drop the XDGSystem
  `CheckSupported` reject) remain.

## Tuning Levers

| Lever | Current | Rationale | Change signal |
|---|---|---|---|
| Auto-resolution order | CWD walk-up, then user `default` | Matches today's empty-id behavior; least surprise | Users routinely shadowed by unexpected ancestor repos |
| Legacy tree handling | Read-in-place as `default`, explicit migration command | No surprise rewrites of user data | Compat shim cost dominates env construction, or all known repos migrated |
| MCP repo cache | One env per repo-id, built lazily per call | Startup stays cheap; matches GetReadBlobStore's build-per-call precedent | Per-call env construction shows up in MCP latency |
| Dot-depth in completions | Offer up to the ceiling | Walk-up is already bounded | Completion noise in deep worktree hierarchies |

## More Information

- FDR-0003 (repo disambiguation) — introduced location-only
  `-repo_id`; this FDR supersedes its "one repo per location"
  constraint while preserving its reservation of `/name` for remote
  selection (system scope gets the explicit `//name` spelling).
- FDR-0015 (multi-store blob lookup) — the madder walk-up and
  `.`/`..` enumeration semantics this FDR mirrors.
- FDR-0016 (blob-store config in mutable config) — the konfig's
  `blob-stores` list; per-repo configs continue to own their store
  order under the new layout.
- Madder `blob_store_id` (`go/internal/alfa/blob_store_id/`) and
  `env_dir.RepoId` (`go/internal/echo/env_dir/repo_id.go`) — the
  types this grammar extends. The `RepoId` change lands in madder
  first; dodder consumes it via `just go/update-flake-input madder`.
- blob-store(7) — the name-charset and prefix table this grammar
  reuses.
