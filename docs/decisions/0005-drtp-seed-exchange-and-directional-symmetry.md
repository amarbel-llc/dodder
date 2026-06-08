---
date: 2026-06-08
status: proposed
---

# drtp Seed Exchange and Its Directional Symmetry

## Context and Problem Statement

drtp (the remote transfer protocol, RFC-0004, PR #253) is a hard fork of
`sierra/remote_http`. RFC-0004 claims the server's `drtp-capabilities-v1`
frame "carries its immutable public config as the bootstrap document,
removing the separate `GET /config-immutable` round trip the HTTP backend
requires." The implementation does **not** do this: the `control` struct
has no immutable-config field and the handshake sends no config frame. The
capabilities frame carries only two of the genesis config's public fields
(`public_key`, `inventory_list_type`).

The HTTP backend serves the repo's public seed at `GET /config-immutable`
(`remote_http/server.go` `handleGetConfigImmutable` → `genesis_configs.
ConfigPublic`, encoded with `genesis_configs.CoderPublic`; no private
keys). Tracing how the HTTP client uses it
(`remote_http/client.go:Initialize`) shows the seed is fetched once at
client init and used for (1) **wire-format negotiation** —
`GetInventoryListTypeId()` selects the object-batch decoder,
`GetObjectSigMarklTypeId()` verifies object signatures — and (2)
**identity attestation** via the remote's `public_key`. Critically, the
clone command genesises a **purely local** repo (`clone.go` →
`OnTheFirstDay` → `genesis_configs.Default()`): it does **not** adopt the
remote's immutable config. The seed is a handshake/validation input, not a
bootstrap.

Because drtp omits the rest of the seed (`store_version`,
`object_sig_type`, `repo_id`), a drtp clone silently assumes the remote is
the same store version and object-signature type as the local default —
working today only because tests clone between current-version repos. This
is a latent clone-correctness bug.

Two questions: must drtp exchange the public seed at all, and if so, should
the exchange be symmetric (both peers) or directional?

## Decision Drivers

- The seed is load-bearing for **decoding**: the receiver of objects needs
  the sender's `inventory_list_type` / `store_version` / `object_sig_type`
  to decode and verify them. The HTTP backend proves this is required, not
  optional.
- Clone genesis is local in both backends — so the seed exchange is a
  validation/negotiation moment, not a config bootstrap. Reusing the HTTP
  serialization keeps the public-no-private-keys guarantee for free
  (`GetImmutableConfigPublic()` returns only `ConfigPublic`).
- drtp is peer-to-peer with a leader: the requester (client) initiates both
  pull and push; the data direction flips, the leader does not.
- **git offers no template.** Per `gitprotocol-pack` / `gitprotocol-v2`,
  git's wire protocol is client-server and asymmetric: the server
  advertises refs and capabilities first and the client selects a subset
  (`gitprotocol-pack` Reference Discovery). git has **no repo-identity
  primitive** at all — `object-format` (sha1/sha256) and `agent` are the
  only seed-like capabilities, and trust is entirely out-of-band (SSH host
  keys, HTTPS certs, signed objects). So git is neither symmetric nor a
  guide for a cryptographically-identified seed.
- drtp already diverges from git toward symmetry: both peers send a
  `capabilities` frame and perform **mutual** nonce/signature attestation —
  more than git does — and dodder has the repo-identity primitive git
  lacks. The seed-symmetry choice is therefore dodder's to define on its
  own terms.

## Considered Options

**Whether/how to carry the seed:**

1. Status quo — rely on the two partial fields already in `capabilities`.
   Leaves the cross-version/cross-sig-type clone bug unfixed.
2. Flatten the missing `ConfigPublic` fields into `capabilities`. Minimal,
   but couples the capabilities frame to the genesis config's shape; every
   future seed field needs a protocol change.
3. A dedicated `drtp-seed-v1` control frame carrying the full
   `TypedConfigPublic` via `genesis_configs.CoderPublic` (the exact
   serialization HTTP already serves).

**Symmetry of the exchange:**

- a. Server-only.
- b. Fully symmetric — both peers always send their seed.
- c. Directional — server always; client only when it is the data sender
  (push).

## Decision Outcome

**Carry the seed as option 3** (dedicated `drtp-seed-v1` frame, reusing
`genesis_configs.CoderPublic`) **with directional symmetry, option c.**

- **The server always sends `drtp-seed-v1`** with its capabilities
  response. It is load-bearing for fetch (the client, as receiver, decodes
  the server's objects with it) and lets the client validate and TOFU-pin
  the server's identity before committing the `want` — preserving the
  "followee proves itself first" ordering drtp already has (server signs
  the client's nonce in its capabilities response; client signs the
  server's nonce in the `want`).
- **The client sends `drtp-seed-v1` only on push**, after the `want` and
  before streaming. It is load-bearing only when the client is the sender
  (the server, as receiver, decodes pushed objects with it). On a read-only
  fetch the client's identity is already attested by the `want`'s
  signature, and a client seed would do nothing on the common path.

The governing principle: the seed's job is *"the receiver decodes the
sender's objects,"* so the **sender's seed must reach the receiver**, and
that direction flips with the transfer. Server-always additionally serves
trust-pinning, which only the followee must satisfy. This keeps git's
leader/follower asymmetry where it earns its keep while adding the mutual
identity git never had.

### Concrete RFC-0004 revision (apply during #253 integration)

This ADR is the spec for the following RFC-0004 edits (the RFC lives on the
PR branch, not master):

1. **Frame-kind table**: replace the RFC's two kinds with the three the
   implementation uses — `0x01` control, `0x02` objects (dedicated, so the
   receiver dispatches on the frame byte instead of parsing the typed doc),
   `0x03` blob.
2. **Add `drtp-seed-v1`** to the control-type list: payload is
   `TypedConfigPublic` encoded by `genesis_configs.CoderPublic`; fields are
   the `ConfigPublic` set (`public_key`, `repo_id`, `store_version`,
   `inventory_list_type`, `object_sig_type`). Specify the directional rule
   above. The receiver MUST configure its importer/verifier from the
   sender's seed and MAY cross-check that the sender's attestation
   `public_key` (capabilities) equals the seed's `public_key`.
3. **Fetch sequence**: insert the server seed right after capabilities;
   correct the object/blob ordering to **blobs-before-objects** (the
   importer needs every referenced blob on disk before object import — the
   `remote_proto` design doc states this; no `RemoteBlobStore` is wired into
   the receiver's importer). The RFC currently has objects-before-blobs and
   is the side that is wrong.
4. **Push sequence**: make the manifest→have exchange explicitly symmetric
   (client sends manifest, server replies have — as implemented), and
   insert the client seed after the `want`.
5. **Backfill** the concrete field lists for `want` / `manifest` / `have` /
   `blob_header` / `ack` / `error` from the `control` struct; drop the
   unused `blob_length`; document the one-session-per-connection,
   no-multiplexing concurrency contract.
6. **hash-format**: note that markl ids are self-describing
   (`blake2b256-…`), so the algorithm travels with every digest; the
   `hash-format` capability field is advisory and a mismatch surfaces
   naturally as a digest parse/verify failure. (No separate enforcement
   needed; the seed's identity fields are the load-bearing negotiation.)

### Consequences

- Good: closes the latent cross-version / cross-sig-type clone bug; a drtp
  clone now decodes and verifies against the remote's actual genesis
  parameters as the HTTP backend does.
- Good: reuses the HTTP serialization wholesale — no new wire format, and
  the public-no-private-keys property comes from `GetImmutableConfigPublic`.
- Good: the capabilities-vs-seed `public_key` cross-check binds "who signed
  this session" to "who this repo claims to be."
- Good: the read-only fetch path gains exactly one frame (the server seed);
  the client over-shares nothing.
- Bad: a small redundancy (`public_key`, `inventory_list_type` appear in
  both capabilities and seed) — accepted for the early-attestation /
  cross-check value.
- Bad: requires the RFC-0004 text edits above plus the `drtp-seed-v1`
  implementation in `remote_proto`; both are part of #253 integration, not
  this ADR.

### Confirmation

When #253 integrates: RFC-0004's frame table and both sequences are revised
per the change-list above; `remote_proto` gains the `drtp-seed-v1` frame
(server-always, client-on-push); and a bats test exercises the seed
exchange — ideally a clone across a store-version or object-sig-type
boundary, or at minimum asserting the seed frame is sent and consumed in
both fetch and push.

## More Information

- PR #253 (drtp websocket protocol); RFC-0004 (on the PR branch).
- HTTP seed path: `remote_http/server.go` `handleGetConfigImmutable`,
  `remote_http/client.go` `Initialize`, `genesis_configs.ConfigPublic` /
  `CoderPublic`, `commands_dodder/clone.go` `OnTheFirstDay`.
- `gitprotocol-pack(7)`, `gitprotocol-v2(7)` — git's asymmetric,
  identity-free wire protocol (the contrast that frames this decision).
- [ADR-0002](0002-move-workspace-filename-resolution-to-command-layer.md),
  [ADR-0004](0004-stateless-operations-layer-shared-by-cli-and-mcp.md).
