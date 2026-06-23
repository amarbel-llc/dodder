---
status: draft
date: 2026-06-23
promotion-criteria: TBD — see Open Questions. At minimum: a repo can
  declare an identity domain in its genesis config; the domain's
  PAPI-published signing roots can be fetched and pinned; an object's
  signature chain can be verified to terminate at a domain-published
  root; and a mismatch (object signed by a key the domain does not
  publish) is detected and surfaced. Promotion blocked until the open
  questions below are resolved with the maintainer.
---

# Domain-Scoped Repos

## Problem Statement

A dodder repo today is identity-anchored only to itself. Every object
carries, in `objects.metadata`, a **repo public key** (`pubRepo`,
`GetRepoPubKey`) and an **object signature** (`sigRepo`,
`GetObjectSig`), and each object commits to its predecessor through a
**mother signature** (`sigMother`, `GetMotherObjectSig`) — the
hash-link that the `mother` / `merkle-mother` formatters walk to read
the previous object by its signature. Those three fields already form a
signed, hash-linked chain: a per-repo merkle structure rooted at the
repo's own keypair, minted at genesis (`genesis_configs`,
`object_finalizer.FinalizeAndSign`).

What is missing is any anchor *above* the repo. The repo's signing key
is its own root of trust — there is no published, externally-resolvable
authority that says "these are the keys allowed to sign objects in this
repo's lineage." A consumer who pulls a repo cannot answer "is this
object signed by a key its owner actually published?" without
out-of-band key exchange. There is also no way for one identity to span
several repos, or for several people/devices to contribute to one
identity's object history under a shared, publicly-verifiable key set.

Separately, FDR-0019 (Scoped Repo Resolution) scopes repos by
**location** (user / cwd / system, plus name and dot-depth). That is an
addressing axis. This FDR proposes an orthogonal **identity** axis:
scoping a repo to a **domain** — a DNS-resolvable hostname whose PAPI
instance publishes the pubkeys that define the roots of that repo's
signature merkle tree.

The first PAPI instance lives on `linenisgreat.com`
(`api.linenisgreat.com` serves the document; `code.linenisgreat.com` is
the code domain), so it is the first concrete domain a repo could be
scoped to.

## Background: the breadcrumbs this builds on

- **`objects.metadata` signature fields** (`go/internal/delta/objects/`):
  - `pubRepo` — the repo public key that signed the object
    (`FinalizeAndSign` copies `config.GetPublicKey()` into it).
  - `sigRepo` — the object signature over the object digest, produced by
    the genesis private key (`privateKey.Sign(objectDigest, …)`).
  - `sigMother` — the predecessor object's signature, making the object
    history a hash-linked chain (a merkle chain). `merkle-mother` emits
    `objectDigest -> motherSig` edges; the `mother` formatter resolves a
    predecessor by `ReadOneMarklId(motherSig, …)`.
- **`genesis_configs`** — owns the repo keypair (`GetPrivateKey` /
  `GetPublicKey`) and `RepoId`. It has **no domain field today**; this
  FDR would add one.
- **`object_finalizer`** — `FinalizeAndSign` / `Verify` are the choke
  points where a domain-root check would attach.
- **PAPI** (`amarbel-llc/papi` RFC-0001; served by
  `friedenberg/linenisgreat` at `api.linenisgreat.com`) — already a
  domain-anchored, self-describing key publisher. `papi.json` carries
  `person.domains[]`, `piggy.encryption_recipients[]`,
  `piggy.ssh_authorized_keys[]`, `proofs[]`, and an OPTIONAL document
  `signatures[]`. RFC-0001 §10 already frames the domain as a trust
  root ("whoever controls the domain") and makes the document
  self-certifying via published markl-id keys. PAPI does **not** yet
  publish a dodder **signing-root** key set — that is the new surface
  this FDR needs (see Open Questions).

## Interface (proposed)

### 1. A repo declares its identity domain

`genesis_configs` gains an optional `identity-domain` (name TBD) — a
hostname string, e.g. `linenisgreat.com`. Empty means today's behavior:
a self-rooted repo with no external anchor (fully backward compatible).

A domain is **DNS-resolvable**: the hostname resolves, and its
`/.well-known/papi` discovery resolves the serving host, exactly as the
existing `papi verify-receipt --domain` flow does in
`friedenberg/linenisgreat` (the pinned `papi` follows discovery from
the bare domain to `api.linenisgreat.com`).

### 2. The domain publishes the signing roots

A domain's PAPI publishes a set of **dodder signing-root pubkeys** — the
markl-id public keys permitted to sit at the root of a domain-scoped
repo's signature chain. These are the roots of the merkle tree of
dodder signatures for every repo that claims the domain.

This is a new PAPI resource (shape TBD — see Open Questions). It is the
dodder-signing analogue of the existing `piggy-ids` /
`ssh-authorized-keys` endpoints, and would be advertised in
`/.well-known/papi` discovery `resources` the same way `proofs` and
`caches` are (only when present).

### 3. Verification anchors the chain to a domain root

When a repo is domain-scoped, `object_finalizer.Verify` (and `fsck`)
gains a check: the chain's root `pubRepo` MUST be a key the domain
publishes. Walking `sigMother` back to genesis yields the root object,
whose `pubRepo` is checked against the (fetched, pinned) domain root
set. A signature by a key the domain does not publish is a verification
failure surfaced to the user, not a silent accept.

Multiple published roots let one domain identity span repos and
devices: any repo whose genesis key is in the domain's published set
verifies under that domain.

### 4. linenisgreat as the first domain

`linenisgreat.com` is the reference deployment: the existing PAPI there
would additionally publish its dodder signing root(s), and a repo could
set `identity-domain = linenisgreat.com` to be verifiable against them.
This dovetails with `linenisgreat`'s live-dodder backend
(`DodderHttpDataSource`) and the card-enrollment receipt flow
(papi FDR-0001), which already gates *adding* a published key behind an
attestation by an already-trusted key.

## Examples (illustrative — exact spellings TBD)

Declare a domain at genesis:

    dodder init -repo_id work -identity-domain linenisgreat.com ...

Verify a pulled repo against its domain's published roots:

    dodder fsck    # now also checks: chain root pubkey ∈ linenisgreat.com's
                   # PAPI-published dodder signing roots

Fetch what a domain publishes (PAPI side):

    GET https://api.linenisgreat.com/papi/<dodder-roots-resource>
    # → the markl-id pubkeys that may root a linenisgreat.com-scoped
    #   repo's signature chain

## Relationship to FDR-0019

Orthogonal axes, deliberately:

- **FDR-0019 = location/addressing.** Where a repo lives and how you
  name it (`name`, `.name`, `//name`, dot-depth). Changes
  `env_dir.RepoId`.
- **FDR-0021 (this) = identity/trust.** Which external domain authority
  defines the signing roots a repo verifies against. Changes
  `genesis_configs` (a domain field) and `object_finalizer` /
  verification (a root check), plus a new PAPI resource.

A single repo can be both: located by FDR-0019's grammar and scoped to a
domain by this FDR.

## Open Questions (for maintainer review)

These are unresolved and block promotion. They are the reason this doc
is `status: draft`.

1. **Which repo am I, really? — domain vs. `RepoId` vs. genesis key.**
   The genesis config already has a `RepoId` and a keypair. Is the
   identity domain a *third* coordinate, or does it subsume/qualify one
   of these? Concretely: is a domain-scoped repo's identity
   `(domain, repo-pubkey)`, or `(domain, RepoId)`, or just `domain`
   with the pubkey set being the membership test?

2. **PAPI resource shape for dodder signing roots.** Is this a brand-new
   endpoint (e.g. `/papi/dodder-roots`), or do dodder signing keys ride
   inside the existing `piggy.*` key sets with a new `purpose`/`scheme`?
   RFC-0001 changes live in `amarbel-llc/papi`; this needs an RFC
   amendment there. What markl purpose tag identifies a "dodder repo
   signing root" pubkey (the object-sig keys today are
   `markl.PurposeObjectSigV2` / ed25519 — do roots reuse that, or a PIV
   slot-9A-style key like the PAPI document signatures in §10)?

3. **"Merkle tree" vs. "merkle chain."** Today `sigMother` gives a
   linear hash-**chain** per repo. The task describes a "merkle tree …
   with the roots defined by PAPI." Is the tree (a) the existing
   per-repo chain whose single genesis root must be domain-published, or
   (b) a genuine tree where a domain's multiple published roots fan out
   over many repos/devices that merge into one identity history? If (b),
   how do independently-rooted repos merge — is there a join/merge
   object, and does pull need to reconcile multiple roots?

4. **Trust bootstrap and pinning.** How does a consumer first obtain and
   pin a domain's root set — TOFU on first pull, the PAPI document
   `signatures[]` (§10) as the self-certifying anchor, the `_papi` DNS
   TXT proof, or DNSSEC? What happens on root rotation: does the
   card-enrollment receipt flow (papi FDR-0001) extend to dodder roots
   so a new root is admissible only when attested by an existing one?

5. **Offline / air-gapped verification.** Domain resolution implies
   network. Can a repo cache the domain root set (a pinned snapshot) so
   `fsck`/`verify` works offline, and how is staleness vs. revocation
   handled?

6. **Revocation and history.** If a root key is compromised and pulled
   from PAPI, do already-signed historical objects retroactively fail
   verification, or is there a validity-window / "valid at signing
   time" model? (Compare RFC-0001 §10.3's "signed-but-invalid is worse
   than unsigned.")

7. **Migration / backward compatibility.** Existing repos have no
   domain and a self-rooted chain. Confirm the intended default is
   "domain optional, absence = self-rooted, no behavior change," and
   decide whether an existing repo can *adopt* a domain after the fact
   (its genesis root would need to already be in the domain's published
   set, or be added via the receipt flow).

8. **Where does this FDR live / who owns each half?** The dodder half
   (config field + verification) is here. The PAPI half (root-publishing
   resource + discovery advertisement) needs an RFC-0001 amendment in
   `amarbel-llc/papi` and a serving change in `friedenberg/linenisgreat`.
   Should this be one cross-repo FDR (this doc) plus a papi RFC, or
   split into paired feature docs in each repo?

9. **Naming.** `identity-domain`? `signing-domain`? `papi-domain`? And
   the PAPI resource name. Deferred until shape is settled.

## More Information

- FDR-0019 (Scoped Repo Resolution) — the orthogonal location/addressing
  axis; this FDR is the identity/trust axis.
- FDR-0003 (Repo Disambiguation) — the location-only `-repo_id` lineage.
- `go/internal/delta/objects/` — `pubRepo` / `sigRepo` / `sigMother`
  metadata fields (the merkle-chain breadcrumbs).
- `go/internal/golf/object_finalizer/` — `FinalizeAndSign` / `Verify`,
  where a domain-root check would attach.
- `go/internal/charlie/genesis_configs/` — the repo keypair + `RepoId`;
  the proposed home of the domain field.
- `amarbel-llc/papi` RFC-0001 §9 (proofs) and §10 (document signature) —
  PAPI's existing domain-as-trust-root framing and self-certifying keys.
- `friedenberg/linenisgreat` (`api/protected/lib/PersonalApi.php`,
  `DodderHttpDataSource`, `papi-verify-keys` receipt gate) — the first
  live PAPI domain and the enrollment-receipt trust flow this would
  extend.
