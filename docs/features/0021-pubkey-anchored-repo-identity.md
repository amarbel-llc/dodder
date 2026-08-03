---
status: proposed
date: 2026-06-28
revised: 2026-08-03
promotion-criteria: the repo public key is the sole absolute repo identity
  — the remote-sync handshake and object provenance key off it, not a human
  name; the location+xdg-scope id and the in-graph repo-genre object are both
  defined as convenience resolvers that map a handle to a pubkey; the
  config-seed standalone repo id no longer drives any identity decision
  (info-repo id, handshake, provenance), with the on-disk field retained and
  decodable for back-compat; BATS coverage proves a repo is resolvable to one
  pubkey by both its host-local location id and an in-graph repo object, and
  that a legacy config-seed name differing from the location id no longer
  produces contradictory identities
---

# Repo Identity: Pubkey-Anchored, Resolver-Addressed

## Revision note (2026-08-03): pubkey demoted to the trust layer

RFC-0007's investigation invalidated this FDR's central premise: the
pubkey is **not** collision-proof as a sole identity, because a
hardware-backed key (a YubiKey PIV slot) can legitimately back more than
one logical repo. The confirmed model (RFC-0007's Revision section is
the normative statement; summary here):

- **The absolute identity is a repo uuid (uuidv7, markl-formatted)** in
  the immutable genesis config, alongside the pubkey — minted at
  genesis, immutable, requiring a genesis-config version bump. Existing
  repos get an explicit mint/migration command; never lazy minting.
- **The pubkey becomes the v2 trust layer**: it *signs* the uuid
  (references upgrade from digest pins to digest+sig), attesting "the
  key holder claims this repo identity" — which is what a pubkey can
  actually promise. Handshake and provenance ultimately key off the
  uuid, with the signature as the attestation over it.
- **This FDR's resolver framing survives unchanged**: the
  location+scope handle (now termed the repo **name**, per the revised
  FDR-0019 terminology) and the in-graph repo object both remain
  resolvers — they now resolve to the uuid rather than to the pubkey.
- **The big open question below ("how do the three handles reconcile")
  is substantially closed**: all handles resolve to one uuid; the
  config-seed standalone id's deprecation stands as written.

The body below is retained as originally written; read "pubkey" as
"identity anchor" through the lens of this note, with the uuid in that
role and the pubkey as its attestor.

## Problem Statement

A dodder repo's identity is currently smeared across redundant, drifting
human names with no immutable anchor underneath them. The `config-seed`
stores a standalone repo id (`config.GetRepoId()`) that is treated as the
repo's identity by `info-repo id`, by the remote-sync handshake
(`sierra/remote_proto`), and by object provenance (`box_format`/`sku`) — yet
it is anchored to nothing and routinely drifts from the location+xdg-scope id
that actually addresses the repo. A home repo whose location id is `default`
but whose config-seed id is a different name is then unaddressable by that
name, and reports an identity no discovery surface shows. The repo already
*has* an immutable ground truth — its public key — but nothing treats it as
the identity; instead several mutable labels each pretend to be one.

## Interface

Repo identity is modeled as **one absolute identifier plus two convenience
resolvers**. The resolvers are addressing conveniences; neither is the
identity. Both resolve *to* the absolute identifier.

### Absolute identity — the repo public key

The repo's ed25519 **public key** (`config.GetPublicKey()`,
`ed25519_pub-…`) is its immutable, absolute, globally-unique identity. It is
the only identifier that survives clone/move, never collides across hosts,
and cannot drift. Everything that must know *which repo, truly* keys off the
pubkey:

- **Remote-sync handshake** (`sierra/remote_proto/{client,server}.go`)
  exchanges the pubkey as peer identity, not `config.GetRepoId().String()`.
  A human name was never a safe peer key — two hosts can both pick `default`.
- **Object provenance** (`golf/box_format/checked_out.go`,
  `foxtrot/sku/{transacted,checked_out}.go`, `golf/store_abbr`) stamps the
  originating repo by pubkey for new objects; legacy name stamps stay
  readable.

### Convenience resolver #1 — location + xdg-scope id (host-local)

The FDR-0019 grammar (`name` user-scope, `.name` cwd-scope, dot-depth for
ancestors) is a **per-host** handle that resolves to a repo on disk, and so
to its pubkey. It is not portable and is not the identity — it is an
addressing convenience valid only on the host whose filesystem holds those
repos. This is already how `-repo_id` and discovery work; this FDR reframes
it as *resolution to a pubkey*, not as identity.

### Convenience resolver #2 — the in-graph repo-genre object

Within the dodder graph, a repo is named by a **repo-genre object**
(`ids.RepoId`) — e.g. a remote definition. This is a second convenience id:
it resolves, *within the graph*, to a repo's pubkey. It travels with the
graph (shareable in the way graph objects are), but like the location id it
is a resolver to the pubkey, not the identity itself.

### Deprecation — the config-seed standalone id

The genesis-config/`config-seed` standalone repo id is **deprecated as an
identity source**. It stops driving `info-repo id`, the handshake, and
provenance. It becomes, at most, a suggested default *local handle* when a
repo is created — never authoritative, never embedded as identity. The
on-disk field is retained and decodable (old store versions must remain
readable); "deprecated" means "no longer authoritative," not "removed."

Net effect: `info-repo id` surfaces the absolute identity (the pubkey);
addressing uses a resolver (`-repo_id <location>` on a host, a repo object
in the graph); and the drift bug disappears because the human handles are
defined as *resolvers to one pubkey* rather than as competing identities.

## Examples

One repo, one pubkey, reached by either resolver:

    # host-local resolver → on-disk repo → pubkey
    der show -repo_id default :z          # location handle (this host)
    der info-repo id                      # ed25519_pub-<repo-pubkey>  (absolute)

    # in-graph resolver → pubkey (e.g. a remote naming the same repo)
    der show -repo_id /backup :z          # repo-genre object → its pubkey

Sync compares pubkeys, not names:

    # two hosts each call their local repo "default"; the handshake still
    # distinguishes them because identity is the pubkey, not the handle

Before this change (the reported bug): `info-repo id` reports a config-seed
name (e.g. one that differs from the location id `default`), discovery lists
only `default`, and `-repo_id <config-seed-name>` resolves nothing — the same
repo presents contradictory identities.

## Open Questions

The big unresolved question is **how the three handles should be reconciled**
— the xdg location id, the config-seed id, and the in-graph repo-object id.
The pubkey-as-absolute-anchor is settled; how the three human handles relate
to it and to each other is **not yet decided**. Specifically:

- Is the config-seed id retired entirely, or does it survive as the *default*
  local handle (i.e. the seed for a repo's location name) — and if it
  survives, what keeps it from drifting again?
- Should the location id and the in-graph repo-object id be the *same*
  namespace viewed from two sides (a repo-object simply records a location
  handle + pubkey), or two genuinely independent resolver namespaces that
  both happen to map to a pubkey?
- When a repo is referred to by a handle that no longer resolves to a known
  pubkey (renamed, moved, never-seen remote), what is the failure/fallback?

These are open by design at `proposed` status; the model above commits only
to the pubkey anchor and to *both* human ids being resolvers, not to the
precise reconciliation among them.

## Limitations

- **Provenance migration is the hard, design-open part.** Objects already
  carry name-based provenance stamps. This FDR proposes pubkey stamps for
  new provenance and keeping legacy stamps readable, but the reconciliation
  of mixed-id histories (rewrite on next commit? display by pubkey or legacy
  label?) is deferred to implementation design.
- **Resolvers are not identity, and have different reach.** The location id
  is host-local and non-portable by design; the in-graph repo object is
  portable only within the graph that carries it. Cross-host, cross-graph
  identity is the pubkey's job alone.
- **Not a wire-protocol break.** The handshake gains a pubkey identity
  field; the config-seed id field is not removed in a way that breaks older
  peers within the supported-version window.
- **The `config-seed` field is deprecated, not deleted.** It stays on disk
  and decodable for back-compat and migration.

## More Information

- FDR-0003 (repo disambiguation) — introduced the location-only `-repo_id`.
- FDR-0019 (scoped repo resolution) — established the location+xdg-scope
  resolver grammar and explicitly carved the genesis-config repo id
  (`ids.RepoId`) out of scope; this FDR takes up that carve-out and anchors
  both resolvers to the pubkey.
- FDR-0020 (config as non-object) — related reshaping of repo config.
- Issue #294 — the discoverability/scoping report whose root cause is this
  multi-identity smear.
- `design_patterns-markl_id` — the pubkey/markl id format that serves as the
  absolute identity here.
- Code touchpoints: `config.GetPublicKey()` (absolute id),
  `charlie/repo_config_cli` + `bravo/genesis_config_blobs` (config-seed id
  source), `bravo/ids/repo_id.go` (`ids.RepoId` repo-genre resolver),
  `env_dir.RepoId` / `scoped_id` (location resolver),
  `sierra/remote_proto/{client,server}.go` (handshake identity),
  `golf/box_format/checked_out.go` + `foxtrot/sku/*` (provenance),
  `uniform/commands_dodder/info_repo.go` (`info-repo id`/`repos`),
  `tango/mcp_dodder/resources.go` (`dodder:///repos` discovery).
