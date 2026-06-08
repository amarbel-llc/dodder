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

## Correction: the wire is self-describing (the seed is not load-bearing for decode)

A closer read of the receive path (`remote_proto/transfer.go`
`receiveClosure` → `importObjects`) overturned the premise an earlier draft
of this ADR rested on. The receiver decodes and imports with **the local
repo's machinery against fully self-describing wire data**, consuming
*nothing* from a remote seed:

- the OBJECTS frame is a *typed* `inventory_list` stream (carries its own
  `! inventory_list-vN` line), so `AllDecodedObjectsFromStream` picks the
  decoder from the embedded type, not a remote `inventory_list-type`;
- blobs are digest-verified against self-describing markl ids (the hash
  algorithm rides every digest);
- objects carry their own signature type, and horizontal versioning keeps
  prior store versions decodable.

So the HTTP seed's load-bearing job (let the client build
`/query/{listType}/…` and pick decoders) **does not exist in drtp** — the
server computes the closure and sends a typed stream — and the identity
half is **already** done by the bidirectional capabilities frame (mutual
`public_key` + nonce/signature attestation). The "cross-version clone bug"
an earlier draft posited is not real.

What a seed genuinely adds is therefore narrower than decode-config:
`repo_id` **provenance** (capabilities carries `public_key` but not the
repo id), an explicit **fail-fast store-version compatibility check**, and
the **identity handshake moment** (which capabilities' attestation already
substantially provides).

## Decision Outcome

**Option B (light): carry `repo_id` and `store_version` in the existing
`drtp-capabilities-v1` frame.** No separate frame, no `CoderPublic`
machinery whose decode-justification does not hold up. Because both peers
already send capabilities, the receiver always has the sender's `repo_id`
and `store_version` regardless of direction — so the directional-symmetry
question (server-always vs client-on-push vs fully-symmetric) **dissolves**:
the fields ride a frame that is already mutual. A receiver MAY reject a peer
whose `store_version` it cannot decode (fail-fast on a future version)
rather than failing deep in import.

**Option C (dedicated `drtp-seed-v1` frame carrying the full
`TypedConfigPublic` via `genesis_configs.CoderPublic`) is retained as a
documented future option**, not chosen now. If a later need arises to
exchange the *whole* public genesis config (richer than `repo_id` +
`store_version` — e.g. forward-compatible transport of new `ConfigPublic`
fields, or a stricter genesis-identity cross-check), C is the path: it
reuses the exact HTTP serialization (public-keys-only guarantee for free)
and would be **directional** (server always; client only on push), because
*that* payload's value follows the data flow even though the light fields
do not. The git contrast above (asymmetric, identity-free wire) is why we
do not overbuild C now: drtp's capabilities already out-do git's identity
story, and B closes the real gap.

### Concrete RFC-0004 revision (apply during #253 integration)

This ADR is the spec for the following RFC-0004 edits (the RFC lives on the
PR branch, not master):

1. **Frame-kind table**: replace the RFC's two kinds with the three the
   implementation uses — `0x01` control, `0x02` objects (dedicated, so the
   receiver dispatches on the frame byte instead of parsing the typed doc),
   `0x03` blob.
2. **Capabilities (option B)**: add `repo_id` and `store_version` to the
   `drtp-capabilities-v1` body. Document them as provenance + a fail-fast
   compatibility check (a receiver MAY reject a peer whose `store_version`
   it cannot decode), NOT as decode-config. Note explicitly that the wire
   is self-describing, so the receiver imports with its own machinery; and
   record option C (the dedicated `drtp-seed-v1` frame via
   `genesis_configs.CoderPublic`, directional server-always/client-on-push)
   as a future option if whole-`ConfigPublic` exchange is later needed.
3. **Fetch/push sequences**: correct the object/blob ordering to
   **blobs-before-objects** (the importer needs every referenced blob on
   disk before object import — the `remote_proto` design doc states this; no
   `RemoteBlobStore` is wired into the receiver's importer; the RFC had it
   backwards), and make the push manifest→have exchange explicitly symmetric
   (client sends manifest, server replies have — as implemented). No
   separate seed frame is inserted under option B.
4. **Backfill** the concrete field lists for `want` / `manifest` / `have` /
   `blob_header` / `ack` / `error` from the `control` struct; drop the
   unused `blob_length`; document the one-session-per-connection,
   no-multiplexing concurrency contract.
5. **hash-format**: note that markl ids are self-describing
   (`blake2b256-…`), so the algorithm travels with every digest; the
   `hash-format` capability field is advisory and a mismatch surfaces
   naturally as a digest parse/verify failure.

### Consequences

- Good: adds `repo_id` provenance and a fail-fast `store_version` check
  with no new frame — the fields ride the capabilities frame both peers
  already send, so directionality is moot and nothing over-shares.
- Good: no `CoderPublic` machinery on the wire, no redundancy, and the
  honest decode story is preserved (self-describing streams + local
  importer); the ADR no longer claims a decode-necessity the code refutes.
- Good: option C stays available and fully reasoned if whole-`ConfigPublic`
  exchange is later justified, without committing to it speculatively now.
- Bad: `store_version` provenance/compat is advisory in v1 (the check is a
  guard, not a negotiation); a genuinely incompatible peer is rejected, but
  drtp does not *adapt* to a peer's version.
- Bad: requires the RFC-0004 reframe above (undoing the earlier
  decode-necessity wording) plus the small capabilities-field addition in
  `remote_proto`; both are part of #253 integration, not this ADR.

### Confirmation

When #253 integrates: RFC-0004's capabilities body, frame table, and both
sequences are revised per the change-list above; `remote_proto`'s
capabilities frame carries `repo_id` + `store_version` with a fail-fast
version guard; and `serve_proto.bats` continues to pass (the existing
fetch/push e2e already exercises the capabilities exchange these fields
ride).

## More Information

- PR #253 (drtp websocket protocol); RFC-0004 (on the PR branch).
- HTTP seed path: `remote_http/server.go` `handleGetConfigImmutable`,
  `remote_http/client.go` `Initialize`, `genesis_configs.ConfigPublic` /
  `CoderPublic`, `commands_dodder/clone.go` `OnTheFirstDay`.
- `gitprotocol-pack(7)`, `gitprotocol-v2(7)` — git's asymmetric,
  identity-free wire protocol (the contrast that frames this decision).
- [ADR-0002](0002-move-workspace-filename-resolution-to-command-layer.md),
  [ADR-0004](0004-stateless-operations-layer-shared-by-cli-and-mcp.md).
