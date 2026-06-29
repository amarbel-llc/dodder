# Repo Identity: Pubkey-Anchored, Resolver-Addressed --- Core Slice Design

- Date: 2026-06-29
- Issue: dodder #294
- FDR: FDR-0021 (`docs/features/0021-pubkey-anchored-repo-identity.md`)
- Builds on: FDR-0019 (scoped repo resolution)

## Scope

Core-first slice of FDR-0021. **In scope**: make the repo public key the
absolute identity that drives the remote-sync handshake, new-object
provenance, and `info-repo id`; deprecate the config-seed id as an identity
source. **Deferred** (tracked): the location-vs-graph namespace reconciliation
(FDR-0021 open Q2), provenance-history migration for mixed name/pubkey stamps,
and foreign pubkey->handle resolution.

## Decisions

1. **Absolute identity = repo ed25519 public key** (`config.GetPublicKey()`,
   `ed25519_pub-...`). Immutable, host-independent, collision-free.
2. **Human-facing identity = markl id `<handle>@<pubkey>`** --- the FDR-0019
   location handle (`name` / `.name`) in the markl `purpose` slot, the pubkey
   as the `ed25519_pub-...` value (e.g. `default@ed25519_pub-9ft3...`). One
   self-describing token. *Nuance*: registered markl purposes follow
   `system-domain-role-version`; a bare handle in the purpose slot is a
   deliberate display repurposing, not a registered purpose.
3. **config-seed id: keep readable, stop writing.** New repos do not populate
   it; existing values remain decodable and are ignored for identity. Drift
   cannot recur because nothing reads it for identity.
4. **Provenance**: new objects stamped by the originating pubkey; legacy
   name stamps stay readable. Display `<handle>@<pubkey>` only when the pubkey
   is *this* repo; foreign repos show the bare abbreviated pubkey (foreign
   resolution deferred).
5. **Handshake: additive pubkey field.** Exchange + compare the pubkey as peer
   identity; older peers that send no pubkey fall back to the name field. No
   wire break.

## Components / touchpoints

- **Shared identity renderer (new)**: produces `<handle>@<pubkey>`; reused by
  `info-repo id` and self-provenance display.
- **`info-repo id`** (`uniform/commands_dodder/info_repo.go`): emit
  `<handle>@<pubkey>` instead of the config-seed name.
- **Provenance** (`golf/box_format/checked_out.go`,
  `foxtrot/sku/{transacted,checked_out}.go`, `golf/store_abbr`): stamp new
  objects by pubkey; display per Decision 4.
- **Handshake** (`sierra/remote_proto/{client,server}.go`): additive pubkey
  field; compare pubkeys.
- **config-seed source** (`charlie/repo_config_cli`,
  `bravo/genesis_config_blobs`): stop writing the id at `init`; keep the
  decoder.

## Data flow

- **Identity**: location handle -> on-disk repo -> `config.GetPublicKey()` ->
  render `<handle>@<pubkey>`.
- **Handshake**: each peer sends its pubkey (plus the legacy name); identity
  comparison keys off the pubkey, falling back to the name only for peers that
  send no pubkey.
- **Provenance**: on commit, stamp the originating pubkey; on display, render
  self as `<handle>@<pubkey>` and foreign as the bare abbreviated pubkey;
  legacy name stamps render as-is.

## Back-compat / rollback

- **Dual-architecture**: the config-seed field is retained + decodable; the
  handshake is additive (name + pubkey coexist on the wire). New and old peers
  interoperate.
- **Rollback**: the change switches the authoritative identity source
  (config-seed name -> pubkey) and *adds* a wire field; nothing is deleted.
  Revert = flip the source back to the config-seed name (single-commit revert).
- **Promotion criteria** (to later retire the legacy handshake name field and
  fully drop config-seed reads): all supported peers exchange a pubkey;
  `info-repo id` shows the pubkey form; the FDR's BATS proof passes (a legacy
  config-seed name differing from the location id no longer yields
  contradictory identities).

## Testing

- A repo resolves to one pubkey via its location handle; `info-repo id`
  renders `<handle>@<pubkey>`.
- Headline proof: a repo whose config-seed name differs from its location id
  shows no contradictory identities (`info-repo id`, discovery, and `-repo_id`
  all agree on the location handle; identity anchors on the pubkey).
- Handshake interop: pubkey exchanged + compared; two repos both locally named
  `default` are distinguished by pubkey; an older peer sending only a name
  still connects.
- Provenance: a new object is stamped by pubkey; self renders as
  `<handle>@<pubkey>`, foreign as the bare pubkey; a legacy name stamp still
  renders.

## Tuning levers

- **Pubkey abbreviation length** in display (`ed25519_pub-9ft3...`). Current:
  the existing markl-abbreviation default. Change signal: collisions or human
  confusion in practice.

## Deferred (tracked follow-ups)

- FDR-0021 open Q2: location id vs in-graph repo-object id --- one namespace or
  two resolvers.
- Provenance-history migration: mixed name/pubkey stamps (rewrite-on-commit?
  unified re-display?).
- Foreign pubkey->handle resolution for provenance and remote display.
