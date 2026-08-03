---
status: exploring
date: 2026-06-07
revised: 2026-08-03
promotion-criteria: resolution conforms to madder's normative
  one-resolver engine FDR (single id -> location function shared by
  every command path); the conformance suite — madder's engine suite
  plus dodder-side bats — passes using non-`default` repo names
  throughout; auto-id hybrid discovery (solo repo wins, `default`
  tiebreak, loud error on unresolved ambiguity) is implemented and
  tested; resolution errors satisfy the fail-fast error contract
  (state what was asked, what was searched, what was found, what to
  do); legacy layouts produce the specific migration-pointing errors
---

# Scoped Repo Resolution

## Revision note (2026-08-03): one-resolver redesign

This FDR was revised in place after a design review confirmed that its
original resolution prose had diverged from the implementation — and
that the implementation itself comprises several independent resolution
engines whose disagreements produced a family of real bugs (#359, #283,
#341, #196; audit umbrella #383). The grammar below is unchanged and
stays. What changed:

- **The resolution engine is no longer specified here.** madder — which
  owns `scoped_id`, `env_dir`, and `directory_layout` — has a normative
  engine FDR: **madder FDR-0010** (scoped-id resolution,
  `docs/features/0010-scoped-id-resolution.md`) defining the single
  resolver `Resolve(id, cwd, env) -> Location`: one id, one physical
  location, for every command; only `.`-prefixed CWD ids walk up
  (store-aware, deepest-first, match-ranked — today's
  `ResolveNthAncestorMatch` semantics, adopted as the one CWD walk);
  init distinguished only by being allowed to find nothing at the
  resolved location and create there (at `$PWD` for CWD scope —
  multi-dot `..name` is addressing-only, rejected at init); a fail-fast
  error contract; and the retirement of implicit ancestor union-merge
  in favor of explicit digest-pinned multi blob-store configs. This FDR
  is the **dodder policy layer** over that engine: repo auto-discovery,
  `repos/<name>/` nesting, and the `/name` remote-first reservation.
- **Auto-id resolution is now specified as hybrid discovery** (see the
  grammar table and "Resolution semantics" below), replacing both the
  original prose ("nearest CWD-scoped repo on the walk-up" — which
  implied discovery the implementation never performed) and the actual
  implemented behavior (substitute the literal name `default`, then
  resolve that fixed name — `repo_id.EffectiveName`). Neither prior
  reading survives: the prose overclaimed, the code under-delivered,
  and the divergence went unnoticed because every bats test uses the
  name `default`, on which the two readings coincide.
- **Known divergences of the current implementation from this revision
  are documented explicitly** (see "Known divergences") rather than
  living as internal footnotes.

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
| *(empty)* | Auto | Hybrid discovery: nearest scope containing any repo; solo repo wins regardless of name; multiple repos → `default` if present, else error listing candidates. See "Resolution semantics". |
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

### Resolution semantics (normative, this revision)

The engine — how an id maps to a physical location — is specified by
madder FDR-0010 (scoped-id resolution) and summarized here only as far
as dodder policy depends on it:

- **One resolver, one answer.** Every dodder command resolves a repo id
  through the same function; the same id names the same physical
  location in every command class. There is no init-time vs.
  operate-time divergence: `init` uses the same resolved answer and is
  distinguished only by being allowed to find nothing there and create.
  For CWD scope, `init` creates at `$PWD` — you `cd` to where you want
  the repo (git's model). Multi-dot `..name` is an addressing-only
  spelling; init never derives a creation location from dot-depth.
- **Auto (empty id) is hybrid discovery.** Resolution proceeds:
  1. Walk CWD ancestors (ceiling-bounded) to the nearest scope
     containing any repo. Within that scope: exactly one repo → it wins,
     regardless of its name; several repos → the one named `default`
     wins if present; several repos, none named `default` → **error**,
     listing the candidate ids so the user can pass `-repo_id`.
  2. If no CWD scope holds any repo, apply the same rule to the XDG
     user scope.
  3. If neither scope holds any repo → error stating both scopes were
     searched and suggesting `dodder init`.

  This preserves every previously-working workflow (`default` still
  wins wherever it won before) while fixing the naming trap: a solo
  CWD repo named anything is reachable by bare commands.
- **Fail-fast error contract.** A resolution miss or ambiguity errors
  at resolve time — never silently falls back beyond the two auto
  steps above, never writes to a store other than the one resolved.
  Every resolution error states: the id asked for, the scopes/paths
  searched, what was found, and what to do next.
- **No implicit union-merge.** An id resolves to exactly one store or
  repo. Multi-store read/write behavior exists only via explicit
  digest-pinned multi blob-store configs (FDR-0016's write_through
  multis); ancestor stores are never implicitly visible to reads or
  writes — listing/completion may still enumerate them (enumeration is
  not resolution; madder FDR-0010's framing). A repo's default blob
  store comes from its own config (repo_configs V3's pinned id) —
  blob-side "default" needs no discovery at all.
- **Legacy layouts get diagnosis, not compat.** The resolver carries no
  legacy branches (#363 stands). A legacy flat tree produces the
  specific error naming `migrate-repo-layout`; conformance fixtures
  assert those exact errors. Migration tooling gaps are filed as
  issues, not folded into resolution.

### Known divergences (current implementation vs. this revision)

Documented so conformance work has a precise starting inventory; the
madder engine FDR carries the engine-level list.

1. **Auto-id substitutes a fixed name instead of discovering.**
   `repo_id.EffectiveName` (delegating to madder `scoped_id`) maps the
   empty id to the literal name `default` and resolves that — no
   discovery of what actually exists. A CWD repo with any other name is
   invisible to bare commands. Masked by the test corpus's universal
   use of the name `default`.
2. **Init and operate resolve `..name` differently.** Literal-init
   paths (genesis, `MakeEnvRepo`) root at the literal Nth parent;
   operate paths (`MakeOperateEnvDir` → `ResolveNthAncestorMatch`)
   resolve the Nth *matching* ancestor. They coincide only when no
   non-matching `.dodder/` sits between matches. Under this revision
   the split dissolves: init creates at `$PWD`, `..name` is
   addressing-only.
3. **A third, walk-up-immune resolution exists** (`init-workspace`'s
   home-parent lookup, the `MakeWithHomeAndInitialize` lineage), which
   can disagree with the walk-up-sensitive paths about where `default`
   physically is — the #359 incident. (madder's own twin of this bug,
   madder#227, is fixed point-wise — unprefixed init is home-pinned —
   but the structural shape, a separate constructor rather than a
   branch of one resolver, remains and is what #359 tracks on the
   dodder side.)
4. **Blob-store discovery union-merges every ancestor `.madder/`**
   (`FindAllCwdOverridePaths`, two-phase `MakeBlobStores`), so which
   physical store serves a read depends on which discovery path ran
   (#196, #341).

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
`repos/` level) are **not** automatically readable by the current
binary — every repo-opening command expects the `repos/<name>/`
nesting and fails against a flat legacy tree. An explicit migration,
`dodder migrate-repo-layout` (landed), copies a legacy tree into
`repos/<name>/`, never modifying the source. Opening a legacy tree
directly surfaces a distinct error naming `migrate-repo-layout`
rather than a generic failure. No silent rewriting: a legacy tree is
left untouched, and stays unreadable, until migrated — a deliberate
scope decision (#363) against the read-in-place fallback originally
promised here, since the explicit-migrate path already resolves the
practical blocker (an unopenable repo) at far lower ongoing
complexity than a permanent legacy-layout compat check on every
repo-open path.

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

User-scoped repos side by side (FDR-0021: `init` names the new repo by its
location-handle positional; `-repo_id` is for addressing existing repos):

    dodder init work ...
    dodder init personal ...
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
- **`default` is the ambiguity tiebreaker, not the only auto target.**
  Under hybrid discovery a solo repo of any name resolves bare;
  `default` only decides among several repos in one scope. A scope
  holding several repos none of which is named `default` requires an
  explicit id (the error lists the candidates).

## Implementation Status

Everything below landed against this FDR's **pre-revision** semantics
(fixed-`default` auto, the init/operate split). The grammar, layout,
listing, and MCP work all survive the revision unchanged; the
resolution behaviors do not — conformance against the revised
"Resolution semantics" is pending and tracked by the madder engine
FDR's suite plus #383's audit. This list is kept as the historical
record of what exists, not as a claim of conformance.

Landed (master):

- **P1 layout + grammar.** `scoped_id`-based `-repo_id`, default-named
  repos, the `repos/<name>/` metadata nest under each scope (madder
  blob env stays flat), and `repo_id.CheckSupported` as the uniform
  scope gate on the working-copy/serve/info paths.
- **P3 user surface.** `info-repo repos` listing and `-repo_id`
  completion of the repo names.
- **Both-scope listing (#276).** `info-repo repos` and `-repo_id`
  completion list both scopes together — cwd repos spelled `.name`, XDG-
  user repos spelled `name` — so every candidate is a directly-usable
  `-repo_id`. The active scope is the cwd walk-up; when it resolves to a
  cwd repo, the user scope is enumerated too via a no-walk-up env_dir
  (`env_dir.MakeStandardXDGUser`). The MCP `dodder:///repos` listing does
  the same — the server captures the user-scope `repos/` dir at startup
  (also via `MakeStandardXDGUser`) and lists both scopes with their
  scope-correct spellings.
- **MCP repo_id, Phase A (tools).** Every repo-touching tool takes an
  optional `repo_id`; the bridge resolves it per call (startup pin
  restricts, unpinned routes per-call), `CheckSupported`-gated.
- **MCP repo_id, Phase B core (resources).** The resource surface is
  repo-scoped: `dodder:///repos/<repo>/...` reads route to the
  addressed repo (bridge- and per-repo-index-routed), `dodder:///repos`
  lists the in-scope repos, `dodder:///repos/<repo>` is a per-repo
  overview, and the legacy un-segmented `dodder://...` URIs stay as
  CWD-auto sugar. The HTTP MCP surface emits the same scheme.
- **MCP per-repo edit / reset-lock / blob-format listing (#278).** The
  three remaining store-backed stragglers now open the addressed repo
  per call via `bridge.OpenRepo` — a fresh `*local_working_copy.Repo`
  built from the repo_id (same pin/`CheckSupported` resolution as the
  tool path). `edit` and `reset-lock` gain an optional `repo_id`;
  blob-format listing routes by the resource URI's repo. No persistent
  cache: mirrors how the bridge already builds a fresh repo per tool
  call, so no lock-holding or index staleness. The open-repo build
  duration is emitted as a stats-me timer (`dodder.mcp.open_repo`) so
  the build-per-call cost is observable; if it ever dominates MCP
  latency, switch to the lazy per-repo cache (the MCP repo cache lever
  below).
- **EffectiveName delegation (#274).** `repo_id.EffectiveName`,
  `EffectiveId`, and `DefaultName` delegate to madder's
  `scoped_id.EffectiveName` / `EffectiveId` / `DefaultName` (shipped in
  madder v0.3.41) — one source of truth, so the empty-id→default
  resolution cannot diverge between the two repos.
- **System scope `//name` (#280).** A forced-system id roots under a
  configured system root instead of no-op'ing to the user tree (madder's
  `rootAtSystem` only fires when `Config.SystemRoot` is set, which dodder
  never did). dodder now injects `SystemRoot` in `env_dir.configFor` — the
  madder blob slot at `madder_env.DefaultSystemRoot` (`/var/lib/madder`),
  the dodder metadata slot at `dodder_env.DefaultSystemRoot`
  (`/var/lib/dodder`); the `DODDER_SYSTEM_ROOT` env var overrides both (for
  relocation or a test sandbox). The operate path (`MakeLocalWorkingCopy`)
  routes `//name` through `MakeDefaultAndInitialize`, as `env_repo`/genesis
  already did, and the XDGSystem `CheckSupported` reject is dropped.
  Consumes the madder `//name` init resolver released in `go/v0.3.43` /
  `go/v0.3.44` — already present in dodder's pinned madder rev. The
  remote-first `/name` spelling stays gated (see below).
- **Multi-dot cwd `..name` (#281).** A multi-dot id resolves the Nth
  same-named ancestor. dodder honors the two cwd-resolution models it
  already has, one per existing depth-0 behavior (madder's deliberate
  literal-init vs store-aware-operate split, `echo/env_dir/AGENTS.md`):
  the **literal-init** paths (`genesis`, `MakeEnvRepo`) root at the literal
  Nth parent — they get multi-dot for free once `repo_id.EffectiveId` (and
  the madder blob-slot strip in `env_dir.MakeDefaultAndInitialize`) stop
  flattening `cwdDepth` to 0; the **nearest-operate** paths
  (`MakeLocalWorkingCopy`, serve's `MakeEnv`/`FromEnvLocal`, `info`'s
  env/xdg display) resolve the Nth *matching* ancestor store-aware via the
  shared `command_components_dodder.MakeOperateEnvDir` helper, which walks
  `directory_layout.ResolveNthAncestorMatch` (deepest-first, ceiling-bounded,
  skipping ancestors that don't host a `notes` repo, erroring on overflow)
  and roots both env slots at the result. The `cwd-depth>0`
  `CheckSupported` reject is dropped. For nested same-named repos at every
  level the two models coincide; they diverge only when a non-matching
  `.dodder/` sits between matches (literal counts it, store-aware skips it).
  `..name` completion is a tracked follow-up (#282).
- **Legacy-layout migration, explicit-only (#363).** `dodder
  migrate-repo-layout` copies a legacy flat `.dodder` tree into the
  `repos/<name>/` nested layout (pure filesystem copy; source never
  modified). The read-in-place fallback originally promised in
  "Legacy compatibility" above was explicitly rejected in favor of
  this explicit-migrate-only approach — opening a legacy tree now
  fails with a distinct error naming the migration command instead of
  transparent read-in-place recognition.
- **Explicit XDG-user name is scope-pinned (#294).** The operate and
  literal-init dispatchers (`MakeOperateEnvDir`, `MakeEnvRepo`) originally
  routed an explicit `LocationTypeXDGUser` bare name through `MakeDefault`'s
  cwd walk-up, so inside a `.dodder/` workspace `-repo_id name` was hijacked to
  the workspace-local repo (or failed outright when the workspace had no repo
  of that name) instead of resolving to `$XDG_DATA_HOME/dodder/repos/<name>/`.
  Both dispatchers now route explicit `XDGUser` (alongside `Cwd`/`XDGSystem`)
  through `MakeDefaultAndInitialize`, which preserves the scoped_id LocationType
  and pins to the user home; only the auto/empty id (`LocationTypeUnknown`)
  still walks up. This matches the FDR grammar (bare `name` = XDG-user scope
  unconditionally) and the discovery enumeration (`MakeStandardXDGUser`).

Deferred (tracked follow-ups):

- **Remote-first `/name`** stays `CheckSupported`-rejected: it means
  "consult the repo's remotes first, fall back to the system-scoped name,"
  but dodder has no remote transport and can't tell whether `name` is a
  defined remote before opening — so it errors rather than silently
  treating it as the system repo (FDR remote-transport limitation).

## Tuning Levers

| Lever | Current | Rationale | Change signal |
|---|---|---|---|
| Auto-resolution order | Hybrid discovery: CWD walk-up (solo wins, `default` tiebreak, error on unresolved ambiguity), then the same rule at XDG user (spec; impl still substitutes the fixed name `default` — see Known divergences) | Fixes the named-solo-repo trap while preserving every workflow where `default` already won | Ambiguity errors prove too noisy in practice, or solo-wins surprises users with unexpected ancestor repos |
| Legacy tree handling | Explicit migration only (`migrate-repo-layout`); a distinct error names the command when a legacy tree is detected | Read-in-place would add a permanent compat check to every repo-open path for a case the explicit migration already fixes in one command (#363) | Read-in-place demand becomes common enough to justify the ongoing complexity |
| MCP repo cache | Fresh repo built per call (no cache); `dodder.mcp.open_repo` stats-me timer tracks build duration | No lock-holding or index staleness; matches the bridge's per-call repo build | The timer shows open-repo build dominating MCP latency — then memoize one env per repo-id |
| Dot-depth in completions | Nearest cwd (`.name`) + user (`name`) only; multi-dot `..name` resolves but isn't yet offered (#282) | Resolution shipped first; completion ergonomics split out to stay focused | #282 lands — then offer `..name` up to the ceiling |

## More Information

- **madder FDR-0010 (scoped-id resolution)** — the normative
  one-resolver specification this revision layers policy over
  (`docs/features/0010-scoped-id-resolution.md` in madder), plus the
  engine-level conformance suite (Go `resolution_conformance_test.go`
  in `scoped_id`/`env_dir`, bats `resolution_conformance.bats`) whose
  expected-fail cases are each annotated to the divergence they
  document. madder's legacy-layout error-contract composes with the
  open madder#175 (legacy-rename error UX); madder issues live on
  code.linenisgreat.com (Forgejo), not the GitHub mirror.
- Issues #359, #283, #341, #196 — the resolver-disagreement bug family
  motivating the revision; #383 — the exhaustive id-resolution audit.
- FDR-0016 (blob-store config in mutable config) — the explicit
  write_through multi mechanism that replaces implicit ancestor-store
  visibility; FDR-0015's implicit sibling-wrap is slated for
  supersession once the engine spec is accepted.
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
