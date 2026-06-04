---
date: 2026-06-04
promotion-criteria: A decision is reached on whether domain-anchored public-key
  declaration uses DNS TXT records, HTTPS .well-known documents, or both; and a
  BATS test resolves a remote repo by a domain name, fetches the domain's
  declared repo public key, and verifies a transfer's challenge-response
  signature against that declared key (rather than against a key learned by
  trust-on-first-use).
status: exploring
---

# Dodder Domains

## Problem Statement

A dodder repository already carries a cryptographic identity: an Ed25519
keypair, stored in the genesis config as markl IDs with purposes
`dodder-repo-public-key-v1` and `dodder-repo-private-key-v1`
(`charlie/genesis_configs/toml_v2.go`). That key signs objects
(`dodder-object-sig-v2`), signs HTTP challenge responses
(`dodder-request-repo-sig-v1`), and signs response bodies via trailers. The
key *is* the repo's identity; everything else is addressing.

Addressing is where the gap is. A remote is stored as a TOML blob
(`alfa/repo_blobs/`: `TomlUriV0`, `TomlXDGV0`, `TomlLocalOverridePathV0`) that
pairs a transport location with an **optional** `PublicKey markl.Id`. The key
is learned by trust-on-first-use: `remote-add` calls
`MakeRemoteFromBlobAndSetPublicKey`, which connects to the remote, reads the
key it advertises in `X-Dodder-Repo-Public_Key`, and pins it into the blob.
Subsequent transfers verify the challenge-response
(`sierra/remote_http/round_tripper_wrapped_signer.go`) against that pinned
key. TOFU itself is still a `TODO` --- there is no prompt, no out-of-band
anchor, and no story for what happens when the key changes.

This has three weaknesses:

1.  **First contact is unauthenticated.** Whoever answers on the wire at
    first connect defines the identity. A MITM at `remote-add` time pins the
    attacker's key permanently.
2.  **Identity is welded to location.** The trust anchor is a URL / XDG path /
    filesystem path. Move the repo to a new host and every peer's pinned
    binding is stale; there is no name that survives relocation.
3.  **Rotation has no path.** Rotating the repo key means re-handshaking every
    peer that pinned the old key, with no way to publish "the new key is
    authentic" except re-running the same unauthenticated TOFU.

FDR-0003 (Repo Disambiguation) deferred remote selection (`/<repo-id>`)
precisely because it "requires user-config-wide repo definitions that exist
outside of individual repos." A **domain** is the natural shape for that
external definition. A domain name is human-memorable, location-independent,
and already sits under an ownership-and-delegation hierarchy. If a domain can
*authoritatively declare the public key(s) of the dodder repo it speaks for*,
then `remote-add example.com` becomes a verifiable bootstrap instead of blind
TOFU, the trust anchor outlives any particular host, and rotation is a matter
of republishing a record.

This FDR examines **how a domain declares its repo public key**, weighing two
mechanisms --- a **DNS** record versus an **HTTPS `.well-known`** document ---
against the Let's Encrypt / ACME lineage that has spent a decade refining
exactly this DNS-vs-HTTP tradeoff.

## Goals

- A domain (e.g. `example.com`) can declare one or more dodder repo public
  keys it vouches for, in a form a fresh client can fetch and verify before
  trusting any wire-level key advertisement.
- Replace TOFU-on-`remote-add` with domain-anchored verification: the key
  advertised in `X-Dodder-Repo-Public_Key` is accepted only if it matches a
  key the domain declares.
- Support key **rotation** and **overlap** (old + new key valid during a
  cutover window) without re-handshaking every peer.
- Reuse the existing markl ID / Ed25519 / signing machinery; do not introduce a
  new key type or a new signature scheme.

## Non-goals

- Issuing X.509 certificates or running an ACME CA. Dodder domains are
  **self-authoritative**: the domain publishes *its own* repo key. There is no
  third-party CA in the trust path (see "Relationship to ACME" below).
- Choosing the human-facing remote-addressing syntax (`@example.com`,
  `dodder://example.com`, `/example.com`). That is downstream of this FDR and
  belongs with the deferred FDR-0003 remote work.
- Discovery of *which* repos a domain hosts beyond a single default. Multi-repo
  fan-out under one domain is noted in Open Questions, not designed here.

## Prior Art

The "prove something about a domain by publishing under DNS or under
`.well-known`" pattern is well-trodden. The relevant lineage:

| Mechanism | Where it lives | What it asserts | Persistence |
|---|---|---|---|
| ACME **HTTP-01** | `http://<d>/.well-known/acme-challenge/<token>` | control of the web root on port 80 | ephemeral, per-issuance |
| ACME **DNS-01** | `_acme-challenge.<d>` TXT | control of the zone | ephemeral, per-issuance |
| ACME **DNS-PERSIST-01** | `_validation-persist.<d>` TXT | a *persistent* authorization bound to a CA + ACME account | durable, "validate once" |
| **did:web** | `https://<d>/.well-known/did.json` | the domain's public keys (`verificationMethod`) | durable declaration |
| **JWKS** | `https://<d>/.well-known/jwks.json` | a set of signing keys | durable declaration |
| **DKIM** | `<sel>._domainkey.<d>` TXT | a public key for signing mail | durable declaration |
| **CAA** | `<d>` CAA RR | which CAs may issue, optionally bound to an `accounturi` | durable policy |

Two distinctions sharpen the design:

- **Challenge vs. declaration.** HTTP-01 and DNS-01 prove *control at a moment*
  by answering a freshly-issued nonce. did:web, JWKS, and DKIM *declare a
  standing fact* (here are my keys) that anyone can read at any time. Dodder
  domains want the latter: a durable, cacheable declaration, not a
  challenge-response handshake. (The wire-level challenge-response dodder
  already has is a separate, complementary layer --- see "Two-layer trust.")

- **Self-authoritative vs. CA-mediated.** ACME's records bind the domain to an
  *account at a CA*; the CA, not the domain, ultimately vouches for the cert.
  Dodder has no CA --- the domain publishes its own repo key directly, so for
  *content* the closer analogs are DKIM / JWKS / did:web (publish your own
  key). What dodder borrows from Let's Encrypt is the hard-won **mechanism**
  knowledge: the operational tradeoffs between a DNS record and a `.well-known`
  file, and especially the **DNS-PERSIST-01 insight** that a *persistent,
  account-bound* DNS record is strictly better than a repeated challenge when
  the thing you are publishing is stable.

### The DNS-PERSIST-01 lesson

DNS-PERSIST-01 (Let's Encrypt, 2026-02-18; CA/Browser Forum SC-088v3 "DNS TXT
Record with Persistent Value", IETF `draft-ietf-acme-dns-persist`) replaced
"answer a fresh challenge on every issuance" with one durable record:

    _validation-persist.example.com.  IN  TXT
      "letsencrypt.org; accounturi=https://acme-v02.api.letsencrypt.org/acme/acct/1234567890"

The TXT value is a CA identifier plus an `accounturi=` binding (with optional
`persistUntil=` expiry and `policy=` flags). The record says, durably, "this
account at this CA is authorized for this name." Validate once; reissue
forever. That is exactly the shape a dodder domain wants --- substitute "this
*repo public key* is authoritative for this name" for the CA/account binding.

## What a Domain Declares

Independent of transport, a domain's declaration carries:

- **`v`** --- declaration format version (`dodder1`), for forward evolution.
- **`k`** --- one or more repo public keys, as markl IDs with purpose
  `dodder-repo-public-key-v1` (`ed25519_pub-...`). Multiple keys express
  rotation overlap; the client accepts a wire-advertised key if it matches
  *any* currently-valid declared key.
- **`id`** --- the repo's `ids.RepoId` (the short kebab-case name), so a domain
  can disambiguate which logical repo a key belongs to.
- **`ep`** *(optional)* --- a transport endpoint hint (the `TomlUriV0`
  location, e.g. `https://example.com:9999`) so a single domain lookup yields
  both identity and address. Absence means "use the default transport for this
  domain."
- **`exp`** *(optional)* --- a validity window / not-after, mirroring
  DNS-PERSIST-01's `persistUntil`. Lets a domain pre-publish a successor key
  with a future activation and retire an old one without a flag day.

The declaration is **not** a secret and **not** a capability; it is a public
statement of "these keys are mine." Its integrity comes from the channel
(authority of the zone, or authority of the TLS-served origin), optionally
reinforced by a signature (see Option B).

## Option A --- DNS-based Declaration

A TXT record under a dodder-specific underscore label, modeled directly on
DNS-PERSIST-01's persistent-value form:

    _dodder.example.com.  IN  TXT
      "v=dodder1; id=myrepo; k=dodder-repo-public-key-v1@ed25519_pub-q7w…; ep=https://example.com:9999"

Rotation publishes a second record (or a second `k=` token) for the overlap
window:

    _dodder.example.com.  IN  TXT  "v=dodder1; id=myrepo; k=…@ed25519_pub-OLD; exp=1793axxxxx"
    _dodder.example.com.  IN  TXT  "v=dodder1; id=myrepo; k=…@ed25519_pub-NEW"

Resolution: to bootstrap `example.com`, the client queries `_dodder.example.com`
TXT, parses the declared key set, and pins it (or, ideally, re-resolves
periodically rather than pinning forever --- the record *is* the source of
truth).

**Strengths**

- **No web server required.** A domain can declare a repo key with a single DNS
  record even if it serves no HTTP at all --- matches dodder's
  transport-agnostic posture (`-tcp`, `-unix`, `-stdio`; HTTPS is optional).
- **Location-independent by construction.** DNS is already the indirection
  layer; the key declaration lives beside the addressing it complements.
- **Cheap rotation and pre-publication** via multiple records + `exp`, exactly
  as DNS-PERSIST-01 intends.
- **Cacheable and relayable.** TXT records are world-readable and TTL-cached;
  no live origin needs to be reachable at verification time.
- **Strong anchor *with* DNSSEC.** Where the zone is DNSSEC-signed, the
  declaration inherits cryptographic chain-of-trust to the root.

**Weaknesses**

- **Weak anchor *without* DNSSEC.** Plain DNS is forgeable by an on-path or
  resolver-level attacker; the declaration is only as trustworthy as the
  resolution path. (ACME mitigates this with multi-perspective validation from
  the CA side --- a single dodder client has no such vantage.)
- **TXT ergonomics.** 255-byte string chunking, registrar UIs that mangle
  semicolons/quoting, and propagation delay make hand-editing error-prone ---
  the same friction that makes DNS-01 "harder to configure than HTTP-01."
- **No payload signature path.** A bare TXT record can carry a markl key but
  not comfortably carry a *signature over* a richer declaration; it leans
  entirely on channel integrity (DNSSEC) for trust.

## Option B --- Well-Known HTTPS Declaration

A document served at a dodder-specific `.well-known` path, in the lineage of
did:web and JWKS:

    GET https://example.com/.well-known/dodder/repo.json

returning a declaration document. Because dodder already speaks hyphence and
already signs blobs, the natural form is a **signed hyphence document** rather
than bare JSON:

    ---
    ! dodder-domain-declaration-v1
    ---

    [repo]
    id = "myrepo"
    endpoint = "https://example.com:9999"

    [[repo.keys]]
    public-key = "dodder-repo-public-key-v1@ed25519_pub-q7w…"
    not-after = 1793000000

    # self-signature over the document body, by one of the declared keys
    signature = "dodder-object-sig-v2@ed25519_sig-…"

Path resolution follows did:web: the bare domain maps to
`/.well-known/dodder/repo.json`; a sub-path
(`dodder:example.com:team:alice`-style) could map to
`/team/alice/dodder/repo.json` if multi-repo fan-out is ever wanted.

**Strengths**

- **TLS authenticates the origin for free.** A successful HTTPS fetch already
  proves "I am talking to whoever holds a cert for `example.com`" --- the same
  Web PKI did:web relies on. No DNSSEC deployment needed for a baseline.
- **Rich, evolvable payload.** A document can carry many keys, endpoints,
  validity windows, and metadata without TXT chunking gymnastics, and can
  version cleanly via the hyphence type string.
- **Self-signature → channel-independent integrity.** Because the document is
  signed by a declared key, it can be relayed, cached, mirrored, or fetched
  over plain HTTP and still be verifiable --- a strictly stronger position than
  did:web, which trusts the TLS channel alone. The first-time bootstrap still
  needs an authentic key (chicken-and-egg, resolved by trust-on-first-fetch or
  by cross-anchoring with Option A), but every subsequent fetch and every relay
  is self-verifying.
- **Reuses HTTP infra dodder already has** (`sierra/remote_http`), and CNAME
  delegation lets a host serve the document for many domains --- the property
  that makes HTTP-01 popular with hosting providers.

**Weaknesses**

- **Requires a reachable HTTPS origin.** A repo reachable only over `-unix` /
  `-stdio` / a non-web `-tcp` port cannot publish this way without standing up
  a web server purely for declaration.
- **Web PKI is the trust root** (for the bootstrap), inheriting its failure
  modes: mis-issuance, compromised CA, and *silent domain ownership transfer* ---
  did:web's known soft spots. The self-signature limits damage after first
  contact but not at it.
- **Port-80/expired-cert footguns.** The same operational fragility ACME's
  HTTP-01 carries: cert expiry or redirect misconfiguration breaks
  verification.

## Two-Layer Trust

These mechanisms answer **"is this key authentic for this domain?"** They do
**not** replace dodder's existing **"is the peer on this connection holding the
private key?"** challenge-response (nonce in `X-Dodder-Challenge-Nonce`,
signature in `X-Dodder-Challenge-Response`, key advertised in
`X-Dodder-Repo-Public_Key`). The two compose:

1.  **Declaration layer (this FDR).** Resolve `example.com` → the set of
    repo public keys the domain vouches for. Source of *authenticity*.
2.  **Liveness layer (existing).** Challenge the connected peer to prove it
    holds the private key for an *advertised* public key. Source of *liveness /
    possession*.

`MakeRemoteFromBlobAndSetPublicKey` changes from "pin whatever key the wire
advertised" to "accept the advertised key **iff** it is in the domain's
declared set, then proceed with the existing challenge-response." TOFU's
unauthenticated first contact is closed without touching the liveness protocol.

## Recommendation (tentative, this is `exploring`)

Follow the Let's Encrypt arc rather than picking a single winner: HTTP-01 and
DNS-01 coexist because their tradeoffs are genuinely complementary, and
DNS-PERSIST-01 shows the durable-declaration framing is the right one for
stable facts.

- **Lead with Option B (signed well-known document)** as the primary mechanism:
  it reuses dodder's hyphence + signing, carries a self-signature so it is
  channel-independent after first contact, and needs no DNSSEC to be useful on
  day one.
- **Offer Option A (DNS TXT) as a peer mechanism and as a cross-anchor**: a
  `_dodder` TXT record that publishes the *same* key set lets a client that
  trusts DNSSEC verify the well-known document's bootstrap key out-of-band,
  collapsing both mechanisms' weaknesses (DNS without DNSSEC is forgeable; HTTPS
  bootstrap trusts Web PKI at first contact). When both agree, the anchor is as
  strong as the stronger of the two channels.

This mirrors how a cautious operator runs CAA (DNS) *and* `.well-known` (HTTP)
together rather than choosing.

## Sketch of Integration (not yet designed)

- **Remote resolution.** A domain-typed remote blob (`repo_blobs.TomlDomainV0`,
  new) stores the domain name as the durable anchor; the transport endpoint and
  pinned key become *cached, refreshable* fields resolved from the declaration
  rather than authoritative ones.
- **`remote-add example.com`.** Resolve the declaration (B, then A as
  cross-check), present the declared key set to the user for confirmation
  (replacing the TODO TOFU prompt), persist the domain + accepted key.
- **Verification.** `round_tripper_wrapped_signer` consults the declared key
  set when validating `X-Dodder-Repo-Public_Key`.
- **Rotation.** A periodic (TTL- or `not-after`-driven) re-resolution updates
  the cached key set; overlap windows mean no flag day.

## Open Questions

- **Underscore label and path.** `_dodder` vs `_dodder-repo`; `.well-known/dodder/repo.json` vs a hyphence `.well-known/dodder/repo` --- bikeshed, but pin it before promotion.
- **Multi-repo per domain.** Does one domain declare exactly one default repo,
  or a set (did:web sub-path style)? Out of scope here; gates on the FDR-0003
  remote-addressing syntax.
- **Bootstrap of the well-known signature key.** First fetch must trust *some*
  key; is that Web PKI (TLS), the cross-anchored DNS record, or an explicit
  pinned fingerprint passed on the CLI? Likely "any of the three, operator's
  choice."
- **Revocation.** `exp` / `not-after` handles graceful retirement;
  emergency revocation of a compromised key (short TTLs? a published revocation
  list? shrinking the declared set?) needs its own treatment.
- **Relationship to encryption keys.** This FDR scopes *signing/identity* keys
  only. Whether a domain should also declare age/pivy *recipient* keys
  (`pivy_ecdh_p256_pub`) for encrypted transfer is a natural extension, deferred.

## Limitations

This feature is in the `exploring` stage. The problem and the option space are
defined and a tentative direction (signed well-known document, DNS TXT as peer
and cross-anchor) is recorded, but no wire format, persistence format, blob
type, or resolution algorithm has been committed. Until it lands, remote trust
remains TOFU-on-`remote-add` (itself still a TODO) verified by the existing
challenge-response layer.

## More Information

- [FDR-0003: Repo Disambiguation](0003-repo-disambiguation.md) --- deferred
  remote selection pending "user-config-wide repo definitions"; domains are a
  candidate shape for those.
- [FDR-0004: Bindingless Local Repo Transfer](0004-bindingless-local-repo-transfer.md)
  --- `-direct` transfers that skip key exchange entirely; the explicit
  counterpoint to domain-anchored trust.
- [RFC 0002: Markl ID Format](../rfcs/0002-markl-id-format.md) --- the
  self-describing key/signature encoding a declaration carries.
- RFC 0001: Hyphence Format --- the signed-document serialization Option B
  would reuse.
- External: Let's Encrypt "DNS-PERSIST-01" (2026-02-18); CA/Browser Forum
  SC-088v3; IETF `draft-ietf-acme-dns-persist`; W3C did:web; RFC 8555 (ACME).
