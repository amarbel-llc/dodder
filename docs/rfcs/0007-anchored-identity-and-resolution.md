---
date: 2026-07-12
status: draft
---

# Anchored Identity and Resolution for Repos and Blob Stores

## Abstract

Both a madder blob store and a dodder repo currently derive their "identity"
from a human-chosen name (`default`, `dodder-v8-take3`, a config-seed repo
id) resolved fresh, per invocation, by walking the filesystem from the
current working directory. Nothing anchors that name to a single physical
thing. Two independent resolvers in the same codebase can each be
individually correct for their own caller's intent and still disagree about
what `default` *is* — and for dodder repos, the deepest available anchor
(the signing pubkey) is not actually collision-proof either, because a
hardware-backed key (a YubiKey PIV slot) can legitimately back more than one
logical repo.

This document proposes a shared pattern across three repos —
`amarbel-llc/piggy`, `amarbel-llc/madder`, and `amarbel-llc/dodder` — of
**content-anchored local identity**, **local scope as a resolver to that
identity, never the identity itself**, and a **global registry** (spinclass-
style locally, PAPI-backed across hosts) that lets any code path discover
every known local instance regardless of which walk-up policy found it. A
further, more exploratory layer considers domain-qualified repo naming and
cross-domain federated trust. This is a design discussion draft, not a
committed plan; open questions are called out explicitly, and an
independent review pass (see below) has already revised part of the
proposal.

**Status note (2026-07-12):** this RFC grew out of scope for its
originating incident. The immediate bug (dodder#359) is being worked
around pragmatically outside this document; this RFC captures the full
design exploration for whenever the underlying architecture work is picked
up.

## Motivating incident

[dodder#359](https://github.com/amarbel-llc/dodder/issues/359): a blob store
named `default` resolved to two different, independent physical directories
depending on which madder resolution mechanism a given dodder code path
used:

- `~/.madder/local/share/blob_stores/default` — reached via madder's
  operate-time ancestor walk (`directory_layout.FindAllCwdOverridePaths`,
  which merges every `.madder/` found walking up from cwd via read
  fallback).
- `~/.local/share/madder/blob_stores/default` — reached via madder's
  init-time, walk-up-*immune* XDG-user resolution
  (`env_dir.MakeWithHomeAndInitialize`), which `init-workspace`'s
  home-parent-repo logic uses deliberately ("NO walk-up — matching the
  former `MakeWithHomeAndInitialize` behavior",
  `command_components_dodder/parent_backed_workspace.go`).

A blob written via the first was invisible to the second. Both resolvers are
individually intentional (madder's own code comments distinguish them,
citing madder#153); the bug is that nothing detects the divergence.

This surfaced during a live repair (rewriting a corrupted config_log blob,
[dodder#358](https://github.com/amarbel-llc/dodder/issues/358)) where the
repair write landed in the ancestor-walk store and `init-workspace`
subsequently couldn't find it.

## Two related but distinct identity gaps

### 1. Blob store identity (madder) — bookkeeping, not trust

Blob store TOML configs (`blob_store_configs.TomlV1/V2/V3` in madder) have
no unique-per-instance field — only user-chosen settings (`hash_buckets`,
`hash_type-id`, `compression-type`, ...). Two stores created with identical
settings produce byte-identical config content and therefore an identical
config digest. Nothing downstream cryptographically trusts a store's
identity, so this is a discoverability/bookkeeping gap: it makes divergence
like dodder#359 silent instead of loud.

### 2. Repo identity (dodder) — a real trust boundary

[FDR-0021](../features/0021-pubkey-anchored-repo-identity.md) anchors repo
identity on the ed25519 pubkey, treating it as settled/absolute, with the
xdg-location id and in-graph repo-object id as mere resolvers to it. That
document does not consider the case where the same pubkey legitimately
backs more than one repo — which is real when the private key lives on
hardware (a YubiKey PIV slot can be, and per piggy's own
`doc/piggy-piv-slots.7.scd`, is designed to be, reused across many logical
purposes via its 20 retired Key Management slots).

Unlike the store case, pubkey *is* what remote-sync handshake and object
provenance actually trust (FDR-0021: "Remote-sync handshake exchanges the
pubkey as peer identity... Object provenance stamps the originating repo by
pubkey"). A bolt-on disambiguator here can't just be a free-floating config
field the way a store UUID can — an unattested value is forgeable
independent of the private key, so it wouldn't actually restore the trust
guarantee FDR-0021 wants. It has to be bound to the key itself. (This
framing was later refined — see "Repo identity: revised direction" below.)

## Proposed pattern

### a. Content-anchored local identity

- **Blob stores**: a new config version generates a random UUID once, at
  `blob_store-init` time, and bakes it into the config. Two stores can no
  longer collide on identity even if every other setting matches.
- **Repos**: see "Repo identity: revised direction" below — the original
  hardware-sealed-key-per-repo mechanism sketched in early drafts of this
  RFC is now the *stronger, deferred* option, not the primary proposal.

### b. Local scope as resolver only

Both the xdg-location id (madder store name, dodder `-repo_id`) and any
in-graph reference remain valid, ergonomic *addressing* — this is not about
removing cwd-scoped or named lookup. It's about making explicit that they
resolve *to* the anchored identity above, and are never themselves treated
as the identity, matching FDR-0021's own framing extended to stores.

### c. Global registry, spinclass-style locally / PAPI-backed across hosts

spinclass gives `sc list` a global view of cwd-scoped worktree sessions by
having each session's local `<worktree>/.spinclass/state.json` register a
symlink into `$XDG_STATE_HOME/spinclass/index/<sha256(worktree-path)[:8]>`
— one canonical index, collision-free because the key is derived from a
stable path hash, readable regardless of which worktree you're standing in.
(Independent review confirmed this is already implemented in spinclass, not
just proposed, despite its own FDR still reading `status: proposed`.)

The proposed analog for a single host: at creation, a local store or repo
instance registers itself into a single per-host index keyed by its own
anchored identity. Any code path — regardless of which walk-up policy it
used to *find* the instance locally — can consult one canonical index to
answer "what are all the stores/repos actually on this host" and detect
when two different local resolutions produced two different identities
under the same name, instead of silently picking whichever one its own
walk-up policy preferred. This directly targets the class of bug in
dodder#359.

**Extension discussed 2026-07-12: PAPI-backed cross-host registration.** A
purely per-host registry can't see divergence *across* hosts (e.g. the same
pubkey used to back different repos on two machines). PAPI is already a
general key/identity directory — extending it to also register
`(pubkey, discriminator)` tuples (see below) at genesis would give a
canonical, cross-host answer to "what repos are known to exist under this
pubkey," and would let a discriminator surfacing during remote-sync that
was *never registered* function as a
real anomaly signal (unauthorized repo creation with a copied key), not
just "unknown peer." This is additive to, not a replacement for, the local
per-host registry — mechanism and schema not yet specified.

Note spinclass's *own* blob-store integration (the accepted
`docs/features/0003-per-worktree-madder-blob-store.md` in spinclass) does
the opposite for storage itself — it deliberately isolates per-worktree
stores via `MADDER_CEILING_DIRECTORIES`, requiring an explicit `madder
sync` to reconcile with the parent. The symlink-registry trick borrowed
here is spinclass's *session-bookkeeping* pattern, not its blob-store
pattern — the two are different problems (session global-visibility vs.
storage isolation) that happen to both be spinclass concerns.

## Repo identity: revised direction (signed discriminator over hardware-sealed key)

The original proposal (provision a fresh, hardware-sealed, F9-attested
ed25519 identity per repo via a piggy retired PIV slot) is a *strictly
stronger* guarantee than what FDR-0021 actually needs, at meaningfully
higher cost: new piggy-side retired-slot provisioning and attestation
plumbing, a hard ceiling of 20 repos per YubiKey, a forced hardware
dependency at genesis (or a software fallback that needs its own design),
and — per independent review — a collision with piggy's own documented
`MaxAuthTries 6` SSH-agent concern once many hardware-backed identities are
surfaced through one agent.

An independent review of this RFC (see below) surfaced a materially
cheaper alternative that better fits the actual trust requirement:

**Signed discriminator.** At genesis, generate a random per-repo value (the
*discriminator*) and sign it with whatever private key the repo already
uses — hardware-backed or not, shared across other repos or not.
`sig = Sign(privkey, discriminator)`. Store `(discriminator, sig)` alongside
the pubkey. Repo identity becomes the tuple `(pubkey, discriminator)`, and
remote-sync handshake / provenance stamping verify and compare that tuple,
not pubkey alone.

This is a different category from the "free-floating config field" the
original framing dismissed as forgeable: producing a *valid* discriminator
claim requires the private key, so a third party without the key cannot
forge one. What it does **not** provide is proof that the key itself is
exclusive to one repo, or that it was freshly generated on hardware for
this purpose — only the full hardware-attested mechanism gives that
stronger guarantee. The discriminator approach needs zero new piggy or
madder capability (dodder already signs routinely), has no slot ceiling,
imposes no hardware requirement at genesis, and doesn't create a new
single point of failure (repo identity isn't newly bound to one physical
token). Its own weak point is a software-correctness dependency: genesis
must always draw the discriminator from a real CSPRNG, which needs
deliberate test coverage (e.g. asserting two genesis runs never produce the
same discriminator) rather than relying on a hardware-structural guarantee.

**Decision as of 2026-07-12: this is the preferred direction.** The
hardware-sealed-key mechanism remains documented above as a stronger,
deferred option if the threat model later requires hardware-level
provenance rather than software self-attestation.

## Extension (exploratory, less developed): domain-qualified naming and federated trust

Discussed 2026-07-12, not yet designed in detail. The layers above
(pubkey → discriminator → PAPI registration) could be extended with a
further layer: a human-readable, domain-qualified repo namespace (mirroring
Go's import-path convention, e.g. `code.linenisgreat.com/dodder`), plus
trust that is transitive across domains — a repo trusts content from its
own domain by default, and can extend trust to other domains, but only
meaningfully if the trusted domain also runs a PAPI instance and signs its
trust declarations (so the extension is attributable and verifiable, not
an unverifiable assertion).

This lands in the same problem space as several well-trodden systems worth
studying before designing further, specifically for their known failure
modes:

- **SSH CA-signed host certificates** — a CA vouches for many hosts;
  relevant analogue for "a domain's PAPI vouches for many repos."
- **DNS-based mail trust (SPF/DKIM)** — a domain's own DNS records vouch
  for traffic claiming to be from it; relevant analogue for domain-anchored
  trust declarations.
- **PGP web of trust** — peer-signed trust graphs, notoriously difficult to
  get transitivity and revocation right in practice.

Known open problems inherited from those systems, not yet addressed here:
**trust-chain transitivity** (does trusting domain B that trusts domain C
imply trusting C?), **revocation timeliness** (how fast does a rotated
PAPI key or a compromised discriminator propagate to everyone who cached
the earlier trust declaration?), and **bootstrapping** (how does the first
trust relationship between two domains get established without an existing
anchor to vouch for it?).

## Independent review (2026-07-12)

A Fable-model review agent (read-only tools only) reviewed this RFC against
the actual dodder/madder/piggy/spinclass source. Full findings on file in
session history; summarized here for the permanent record.

**Verified accurate:** the motivating-incident mechanics, madder's two
intentional resolvers (citing madder#153), the deterministic blob-store
config claim, piggy's `RegisterSSHEd25519Format` SSH-agent pattern, the
`piggy-piv-slots.7.scd` retired-slot documentation, and — notably —
spinclass's registry pattern is already *implemented*
(`internal/session/session.go`'s `writeIndexSymlink`), not merely proposed
as its own FDR's frontmatter claims.

**Corrections applied to this document:** the struct name
`TomlLocalHashBucketedV1/V2/V3` (nonexistent; corrected to `TomlV1/V2/V3`
above) and an overstated claim that "PIV does not natively support
ed25519" — piggy's own Rust crate already defines a `PivAlgorithm::Ed25519`
byte for YubiKey 5.7+ firmware, so this is true only for baseline NIST PIV,
not the hardware piggy actually targets.

**A framing error on the reviewer's brief, not the RFC itself:** the
review was briefed that "madder and piggy don't depend on each other" —
false; madder's `go.mod` requires piggy directly (`markl` import). This is
favorable for the design: a shared registry package could live in piggy or
in the shared `purse-first/dewey` library (which all three repos already
consume, and which already owns XDG/ceiling-directory logic) without
creating a new dependency edge.

**Composition risk flagged:** the registry itself would be resolved
through the *same* XDG/ceiling-directory machinery responsible for
dodder#359 in the first place — a "per-host singleton" registry needs an
explicit answer for sandboxed test lanes (bats, nix builds) and
spinclass's own deliberate `MADDER_CEILING_DIRECTORIES` per-worktree
isolation, or it inherits the exact ambiguity it's meant to resolve.

**Gaps not covered by this RFC's own open questions, per review:**
advisory-vs-mandatory registry consultation (without mandatory checks, the
bug class in dodder#359 survives even with a registry present); registry
staleness/GC as stores/repos move or get deleted; system-scope
(`LocationTypeXDGSystem`) and multi-user store ownership/permissions; where
F9 attestation would actually be verified (locally at genesis only, or
exchanged during remote-sync handshake); and the `MaxAuthTries 6` SSH-agent
collision noted above, which piggy's own doc already flags but this RFC
had not connected to the per-repo-hardware-identity proposal.

**Alternatives the review considered** (see "Repo identity: revised
direction" above for the one now adopted): an on-demand divergence
diagnostic with no new schema (cheapest, retrofits to existing stores,
converts a silent failure into a loud one, but doesn't prevent divergence
or give global enumeration — "arguably the right first step regardless");
collapsing to a single resolution policy everywhere (structurally prevents
dodder#359 with zero new machinery, but reverses a deliberate design
distinction and breaks legitimate cwd-scoped-store UX).

## Open questions

- **Slot exhaustion** (if the hardware-sealed-key option is ever revisited):
  a YubiKey has only 20 retired slots.
- **Migration**: existing repos/stores that already share an identity (like
  the repo this session repaired) need a transition path — this proposal
  describes *new* genesis/init behavior, not a retrofit.
- **Registry schema and location**, including how it avoids inheriting the
  XDG/ceiling-directory ambiguity it's meant to resolve (see Composition
  risk above).
- **Shared implementation vs. shared pattern**: madder and dodder are
  separate Go modules, released independently (dodder also ships a
  standalone madder binary as a pinned flake input — CLAUDE.md already
  warns that skew between them produces "cryptic wire-form mismatches").
  Any shared registry format needs to be explicitly versioned and
  forward-tolerant across independently-released binaries, not just
  independently-compiled modules.
- **Divergence UX**: when the registry (or a diagnostic) finds two physical
  stores/repos under one name, what actually happens — hard error,
  warning-and-pick-one, or interactive resolution? Which commands check,
  and when (every resolve? only at init? only on explicit demand)?
- **PAPI registration mechanism and schema**: not yet specified — what
  gets sent, when, and how a client authenticates to PAPI to register.
- **Domain-trust bootstrapping, transitivity, and revocation**: see the
  Extension section above; unresolved.

## References

- [dodder#358](https://github.com/amarbel-llc/dodder/issues/358) — the
  corrupted config blob that motivated the repair which surfaced this
- [dodder#359](https://github.com/amarbel-llc/dodder/issues/359) — the
  concrete store-divergence incident
- [dodder#360](https://github.com/amarbel-llc/dodder/issues/360) — capture
  issue for this design exploration
- [FDR-0015 — Multi-Store Blob Lookup](../features/0015-multi-store-blob-lookup.md)
- [FDR-0021 — Pubkey-Anchored Repo Identity](../features/0021-pubkey-anchored-repo-identity.md)
- `piggy-piv-slots(7)` (`amarbel-llc/piggy`, `doc/piggy-piv-slots.7.scd`)
- spinclass `docs/features/0001-worktree-local-session-state.md` (proposed)
  and `docs/features/0003-per-worktree-madder-blob-store.md` (accepted)
