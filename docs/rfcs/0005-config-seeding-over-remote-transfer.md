---
status: experimental
date: 2026-06-14
---

# Config Seeding over Remote Transfer

## Abstract

Repository configuration is a repo-local, signed append-only log (FDR 0020):
it is not a store object and is therefore never carried by the object/blob
sync. A direct (local-path) clone already seeds the new repository's config
from the source; a clone over a network transport does not, so it silently
starts from the genesis default. This document specifies an optional,
negotiated affordance by which a serving repository conveys its current
config-log head — a config blob digest and the blob's type — to a cloning
client, over both the drtp session protocol (RFC 0004) and the legacy HTTP
backend, and specifies how the client seeds its own config log from it. The
affordance is read-only, content-addressed, and applied only at clone time.

## Introduction

After FDR 0020, configuration stopped being a store object. It lives in a
repo-local append-only log (`config_log`, a signed `!inventory_list-v2`
stream) that is excluded from the inventory lists, the stream index, and the
query surface. Consequently the remote sync — which transfers query-matched
objects and their transitive blob closure (RFC 0004) — never moves config.

To preserve the pre-FDR-0020 behavior in which a clone inherited the source's
config (then carried by the `konfig` object), the `clone` command seeds the
new repository's config log from the source. For a **direct** transfer the
source is opened in-process as a local working copy, so the client reads the
source's config blob digest and type directly and appends a new, locally
signed config-log entry referencing the (copied) config blob. For a
**network** transfer (drtp/WebSocket, or the legacy HTTP backend) the client
has no such access: the serving peer exposes only objects, their blobs, and
the public genesis seed (`id`, `store-version`) — not the mutable config log.
A network clone therefore keeps only its genesis-default config, a regression
from direct clone and from pre-FDR-0020 behavior.

This document specifies the missing affordance: how a serving repository
offers its current config-log head over a network transfer, and how a client
seeds from it. The scope is limited to **clone-time** seeding; ordinary
`pull` MUST NOT adopt the remote's config (a pull must not overwrite local
config). The affordance is OPTIONAL and negotiated: a server that does not
offer it, or a client that does not request it, behaves exactly as today (the
clone keeps its genesis default).

This specification builds on, and normatively references, the drtp protocol
(RFC 0004), the hyphence wire format (RFC 0001), and markl IDs (RFC 0002). It
realizes the network half of issue #268; the direct half shipped with
FDR 0020.

## Requirements Language

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be
interpreted as described in RFC 2119.

## Specification

### Config Descriptor

A **config descriptor** identifies a single config state without carrying its
bytes. It has these fields:

- `blob-id` (REQUIRED) — the markl ID (RFC 0002) of the config TOML blob that
  is the source repository's current config-log head.
- `config-type` (REQUIRED) — the config blob's own type string (e.g.
  `!toml-config-v2`). The seeded entry keeps this type so the client's
  `store_config` bootstrap can decode the blob via the config coder.
- `tai` (OPTIONAL, informative) — the source head entry's timestamp. It is
  provenance only; the client MUST NOT use it as the timestamp of the entry it
  seeds (see Client Seeding).

The descriptor names a config blob; the blob itself is transferred over the
existing content-addressed blob mechanism of the transport, never inline in
the descriptor.

### drtp Transport (RFC 0004)

This section extends RFC 0004 with one new control message type and one
transfer obligation. It introduces no new frame kind.

#### `drtp-config-v1` control message

A new CONTROL message type (`kind = 0x01`), carrying a config descriptor.
Body (TOML, in the reference implementation's `control` envelope):

    blob-id     = "blake2b256-..."     # source config-log head blob digest
    config-type = "!toml-config-v2"    # the config blob's own type
    tai         = "..."                # OPTIONAL, informative provenance

Per RFC 0004's versioning discipline, additional optional fields MAY be added
to `drtp-config-v1` without a version bump; an incompatible change introduces
`drtp-config-v2` with the prior type still decodable.

#### Server obligations (fetch)

During a fetch (RFC 0004 §Fetch), a server that supports config seeding and
whose repository has a non-empty config log:

1. MUST send exactly one `drtp-config-v1` CONTROL frame naming its current
   config-log head, after the client's `drtp-want-v1` is accepted and before
   the terminal `drtp-ack-v1`.
2. MUST include the config blob named by that descriptor's `blob-id` in the
   transfer's blob set, so it participates in have-negotiation
   (`drtp-manifest-v1` / `drtp-have-v1`) and is streamed via
   `drtp-blob_header-v1` + BLOB frames exactly like any other blob. The config
   blob is added to the closure's blob set even though no transferred object
   references it.

A server whose config log is empty, or that does not implement this
affordance, MUST NOT send `drtp-config-v1` and MUST behave exactly as RFC 0004
specifies otherwise.

A server MUST NOT send `drtp-config-v1` for a push: config is repo-local and a
push does not convey the sender's config to the receiver.

#### Client obligations (fetch)

A client:

1. MUST tolerate the absence of `drtp-config-v1`; absence means "no config
   offered" and the clone keeps its genesis-default config.
2. MUST ignore a received `drtp-config-v1` unless the current operation is a
   clone (see Client Seeding). A non-clone fetch (pull) that receives the
   frame MUST discard the descriptor.
3. that received a `drtp-config-v1` and intends to seed MUST verify that the
   config blob arrived (or already exists locally) and that its computed
   content digest equals the descriptor's `blob-id` before seeding (content
   addressing; see RFC 0004 §Authentication). A mismatch MUST abort seeding
   with a diagnostic and MUST NOT append a config-log entry.
4. that does not recognize the `drtp-config-v1` control type (an un-upgraded
   client) MUST skip it and continue the session, per RFC 0004's additive
   versioning of control message types. Thus an un-upgraded client tolerates
   an upgraded server's unconditional send of `drtp-config-v1`; the frame is
   ignored and the clone keeps its genesis-default config without aborting.

### HTTP Backend Transport

The legacy HTTP backend (the request/response `remote_http` server) MAY expose
config seeding via a dedicated route. A server that supports it:

- MUST serve `GET /config` returning a config descriptor as a JSON object with
  fields `blob-id`, `config-type`, and OPTIONAL `tai`, with the same meanings
  as above, naming the repository's current config-log head.
- MUST return HTTP `404 Not Found` from `GET /config` when the repository's
  config log is empty.

A client cloning over the HTTP backend:

- MAY issue `GET /config`. A `404` (or a connection to a server that does not
  route `/config`) MUST be treated as "no config offered" — the clone keeps
  its genesis default.
- MUST fetch the config blob named by `blob-id` via the existing blob route
  and MUST verify its digest against `blob-id` before seeding.

The HTTP `/config` route is OPTIONAL. drtp is the forward-looking transport;
the HTTP route exists so an HTTP-backend clone is not permanently denied config
seeding.

### Client Seeding

Seeding is applied by the `clone` operation only. A `pull` MUST NOT seed config
from the remote.

Given a verified config descriptor and the config blob present in the clone's
default blob store, the client:

1. MUST compare the descriptor's `blob-id` with the blob digest of the clone's
   current config-log head. If they are equal, the client MUST skip seeding
   (the genesis-default root already records the identical config).
2. Otherwise MUST append a new entry to the clone's config log that:
   - references the config blob (`blob-id`),
   - carries the descriptor's `config-type` as the entry's type,
   - is timestamped with the clone's own clock (not the descriptor's `tai`),
     and
   - is signed with the **clone's** repository key.

The appended entry becomes the clone's config-log head, so the clone's
`store_config` bootstrap reads the source's config thereafter. This mirrors the
direct-transfer seeding shipped with FDR 0020: the clone adopts the source's
config *content* but owns the entry (its own signature, its own timestamp,
chained onto its own genesis root). The source's signature on its head entry is
not transferred or relied upon.

### Errors

A drtp `drtp-config-v1` exchange failure (e.g. the named config blob never
arrives, or fails digest verification) MUST be reported as a `drtp-error-v1`
(RFC 0004 §Errors) or surfaced as a clone diagnostic, and MUST leave the
clone's config log unchanged beyond its genesis-default root. An HTTP-backend
client that cannot fetch or verify the config blob MUST surface a diagnostic
and MUST NOT append a config-log entry. Failure to seed config MUST NOT, by
itself, fail the clone of objects and blobs — a clone with default config is a
valid, usable repository.

## Security Considerations

- **Trust boundary.** Cloning from a repository already entails trusting that
  peer for the objects and blobs it serves; the config it serves is within the
  same trust boundary and is applied only by the explicit `clone` action. A
  malicious or compromised server can serve arbitrary config (e.g. a hostile
  default blob-store or type), exactly as it can serve arbitrary objects. A
  client that pins the server's public key (RFC 0004 §Authentication) binds the
  config to that identity.
- **Integrity.** The config blob is content-addressed; the client MUST verify
  its computed digest against the descriptor's `blob-id` before seeding, so a
  man-in-the-middle cannot substitute config content without detection. The
  descriptor itself rides the authenticated drtp session (or the HTTP backend's
  existing request signing); it conveys only a digest and a type string, not
  executable content.
- **Provenance, not authority.** The clone re-signs the seeded entry with its
  own key and does not retain the source's signature. The seeded config is the
  clone's own config from that point; there is no cross-repository signature
  trust to forge.
- **No new authorization surface.** Config seeding is read-only from the
  server's perspective and is offered over the same session/route auth as the
  rest of the transfer. A public (read-only) server MAY offer it; a server MUST
  NOT accept config *from* a client (there is no push-config path).
- **Scope containment.** Seeding never occurs on `pull`, so a routine sync
  cannot silently overwrite a user's local config with a remote's.

## Conformance Testing

Conformance tests for this specification live in
`zz-tests_bats/current_version/` (network-lane clone tests, e.g.
`clone_port.bats` for the drtp/WebSocket path; the direct-transfer baseline is
`clone.bats::clone_direct_seeds_config_from_source`).

Tests use binary injection via `bats-emo`:

    require_bin DODDER dodder

### Covered Requirements

| Requirement | Test File | Description |
|-------------|-----------|-------------|
| drtp §Server obligations, MUST send `drtp-config-v1` + include the config blob | `clone_port.bats` | Edit source config to a distinctive marker, clone over the network transport, assert the clone's `show-config` streams the marker (not the genesis default) |
| Client Seeding, MUST skip when equal | `clone_port.bats` | Clone a source whose config equals the default; assert the clone's `show-config -history` has no extra entry beyond the genesis root |
| Client obligations, MUST tolerate absence | `clone_port.bats` | Clone from a server that does not offer config seeding; assert the clone succeeds with default config |
| Client Seeding, MUST NOT seed on pull | `pull.bats` | Pull from a source with different config; assert the local config is unchanged |
| Integrity, MUST verify blob digest | (go unit / integration) | A config blob whose bytes mismatch the descriptor `blob-id` aborts seeding |

## Compatibility

- **Optional and negotiated.** A server that does not implement this affordance
  sends no `drtp-config-v1` and routes no `GET /config`; a client tolerates
  both. A new client against an old server, or an old client against a new
  server, behaves exactly as today: the network clone keeps its genesis-default
  config. There is no wire break — drtp adds an optional control type per
  RFC 0004's versioning rules, and the HTTP route is additive.
- **Direct transfer is the reference.** The behavior specified here matches the
  already-shipped direct-transfer seeding (`clone seedConfigFromDirectSource`,
  FDR 0020); this document brings the network transports to parity, not a new
  behavior.
- **Push unaffected.** Config is never pushed; this document adds nothing to the
  push path.
- **No store-version bump.** Config seeding is a transfer-time affordance; it
  does not change any persisted format. (FDR 0020 itself required no bump.)

## References

### Normative

- [RFC 0001] Hyphence Format — the wire representation for drtp control
  messages.
- [RFC 0002] Markl ID Format — config blob digests and signatures.
- [RFC 0004] Remote Transfer Protocol — the drtp session protocol this
  affordance extends (control framing, blob streaming, have-negotiation,
  authentication).

### Informative

- [FDR 0020] Config as a non-object — the repo-local config log and the
  direct-transfer seeding this document mirrors.
- [dodder #268] clone config-seeding over websocket/http — the issue this
  specification resolves.
