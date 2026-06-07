---
date: 2026-06-07
status: proposed
---

# Remote Transfer Protocol

## Abstract

Dodder synchronizes objects and blobs between repositories over a remote
transfer protocol. Today that protocol is *implicit*: it is whatever the
`sierra/remote_http` client and server happen to send each other --- a sequence
of REST calls (`GET /query`, `POST /inventory_lists`, `POST /blobs`) glued
together by a reactive missing-blob retry loop. There is no document that
specifies the exchange, no version negotiation, and no single connection over
which a whole transfer happens; every request re-establishes framing and
re-runs the signing handshake.

This document specifies an explicit, versioned **remote transfer protocol**
(`drtp`). It is a session protocol: one connection carries a complete transfer
from capability handshake through object and blob streaming. It takes its
overall shape from git's fetch/push (`send-pack` / `receive-pack` /
`upload-pack`): capability advertisement, a `want`/`have` reference
negotiation, and a single batched object stream. Unlike git, every control and
payload message on the wire is a **typed hyphence document** (RFC 0001), so the
protocol inherits hyphence's free versioning (type-string dispatch) and
content-addressed, signable framing (RFC 0002 markl IDs) rather than inventing a
bespoke pack format. The sender computes the transfer's transitive closure up
front using the type system's *expand-edges* traversal, so a fetch delivers a
complete, self-consistent object graph in one pass instead of discovering
missing blobs by trial and error.

The protocol MAY run over any reliable, ordered, bidirectional byte stream:
stdio pipes (`dodder serve -`), a Unix domain socket, a raw TCP connection, or
--- the motivating addition of this document --- a **WebSocket** upgraded from an
ordinary HTTP request. WebSocket support is optional and negotiated: a client
attempts the upgrade and falls back to the request/response transport when the
peer does not offer it.

The implementation is a hard fork of `sierra/remote_http` into a new package,
`sierra/remote_proto`; the existing HTTP backend is retained unchanged for
backward compatibility.

## Introduction

The HTTP backend grew organically and works, but its protocol is hard to reason
about and hard to evolve:

- **It is reactive about blobs.** `POST /inventory_lists` imports the objects a
  client offers, replies with the subset of blobs it is *missing*, the client
  uploads those, and the loop repeats until the server reports completion (the
  `201`/`417` dance in `client.pullQueryGroupFromWorkingCopy`). Each round trip
  is a full HTTP request. The local-to-local path
  (`local_working_copy.pullQueryGroupFromWorkingCopy`) already does better: it
  computes the *entire* set of reachable objects and blobs once, with
  `expandEdges`, and copies them. The remote path does not use expand-edges at
  all.

- **It has no version negotiation.** There is a literal `// TODO local / remote
  version negotiation` in the client. Two peers cannot discover each other's
  protocol version, supported compression, or feature set; they can only hope
  the REST surface matches.

- **It re-handshakes per request.** The signing middleware
  (`RoundTripperBufioWrappedSigner`) mints a nonce, verifies a challenge
  response, and verifies a body-digest trailer on *every* request. Over a
  high-latency link a transfer is dozens of serial handshakes.

- **The framing is HTTP-specific.** The wire format is "whatever
  `http.Request.Write` / `http.ReadResponse` produce," which couples the
  protocol to Go's `net/http` and rules out transports (like WebSocket) that do
  not present a fresh request per exchange.

The goal is a protocol that is specified, versioned, computes its transfer up
front via the type system, handshakes once per session, and is framed
independently of any one transport so it can ride a WebSocket.

### Relationship to existing work

This protocol reuses, rather than replaces, three existing facilities:

- **Hyphence (RFC 0001)** is the wire representation for every message. Control
  messages are small typed blobs; object batches are exactly the
  `inventory_list` hyphence stream the store already encodes
  (`inventory_list_coders.Closet`).
- **Markl IDs (RFC 0002)** address blobs, lock types, and carry signatures, so
  the protocol's integrity and authentication reduce to existing markl
  operations.
- **Expand-edges** (`sku.EdgeExplorer`, `store.MakeEdgeExplorer`) computes the
  transitive closure of an object set: its type object, every tag object,
  referenced objects, blob references, and --- recursively, driven by each
  type's `[references]` script --- nested blob and object references.

## Requirements Language

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be
interpreted as described in RFC 2119.

## Specification

### Transport and Session

A *session* is a single transfer conducted over one reliable, ordered,
bidirectional byte stream (an `io.ReadWriteCloser`). The protocol is defined
entirely in terms of that stream and is therefore transport-agnostic. A
conforming implementation MUST support running a session over:

1.  **stdio** --- the child process spawned by `dodder serve -`, whose stdin and
    stdout are the stream (local and SSH transports);
2.  **Unix domain socket** --- a `net.Conn` to a bound socket;
3.  **TCP** --- a `net.Conn` to a `host:port`;
4.  **WebSocket** --- a connection upgraded from HTTP, adapted to a `net.Conn`.

The client opens the session; the server accepts it. Exactly one transfer
(fetch or push) runs per session. After the transfer terminates, either peer
MAY close the stream.

#### WebSocket upgrade

A server that supports WebSocket transport MUST expose an HTTP endpoint at the
path `/drtp` that accepts the standard WebSocket upgrade (`GET` with
`Upgrade: websocket`). On a successful upgrade the connection is adapted to a
binary message stream: every protocol frame (below) is sent as one binary
WebSocket message. The upgraded connection is then treated as the session's
byte stream.

A client requesting WebSocket transport MUST issue the upgrade to `/drtp`. If
the upgrade fails --- because the peer is an older server that does not offer
`/drtp`, or a proxy strips the upgrade --- the client MUST either fall back to a
non-upgraded transport (the request/response HTTP backend, `remote_http`) or
fail with a diagnostic that names the missing capability. This is the "optional
upgrade": WebSocket is used when both peers support it and is never required.

The `/healthz` liveness route and, for bootstrap, `/config-immutable` MAY be
served alongside `/drtp` so that probes and capability discovery do not require
an upgrade.

### Framing

The session byte stream carries a sequence of **frames**. A frame is:

    +--------+-----------------+==================+
    | kind   | length (uint32) | payload (length) |
    +--------+-----------------+==================+

- `kind` is a single byte identifying the payload's interpretation.
- `length` is a big-endian unsigned 32-bit byte count of the payload.
- `payload` is exactly `length` bytes.

Two frame kinds are defined:

- `kind = 0x01` **CONTROL**: the payload is a single typed hyphence document
  (RFC 0001) whose type string is one of the control types below.
- `kind = 0x02` **BLOB**: the payload is the raw bytes of one content-addressed
  blob. A BLOB frame MUST be immediately preceded by a CONTROL frame of type
  `drtp-blob_header-v1` naming the blob's markl ID and byte length.

A receiver MUST reject a frame whose `length` exceeds a configured maximum
(RECOMMENDED default 64 MiB for CONTROL frames; BLOB frames are bounded by the
`drtp-blob_header-v1` length and MAY stream in chunks --- see Blob Streaming).
Framing over WebSocket maps one frame to one binary message; framing over a raw
byte stream writes the header and payload contiguously.

The choice of length-prefixed frames (rather than hyphence's own `---`
boundaries as the outer container) keeps the reader from having to scan for
boundaries on an unbounded stream and lets BLOB payloads --- which are arbitrary
bytes and need not be valid hyphence --- share one channel with control
messages.

### Control Message Types

Every control message is a typed hyphence document. New fields are added without
a version bump when they are optional; an incompatible change introduces a new
type string (e.g. `-v2`), and both versions remain decodable, exactly as
horizontal versioning prescribes. The defined types are:

- `drtp-capabilities-v1` --- capability advertisement (both directions).
- `drtp-want-v1` --- the requested query plus transfer options (client to
  server for fetch; server to client for push).
- `drtp-have-v1` --- object identities (with version digests) the receiver
  already holds.
- `drtp-blob_header-v1` --- announces the BLOB frame that follows.
- `drtp-ack-v1` --- a phase acknowledgement (e.g. "objects received, here are
  the blobs I still need" or "transfer complete").
- `drtp-error-v1` --- a terminal error; carries a human-readable message and an
  optional status code.

Object batches are not a distinct control type: they are CONTROL frames of the
repository's configured `inventory_list` type (e.g. `inventory_list-v2`),
encoded by `inventory_list_coders.Closet` exactly as in on-disk and HTTP
transfer. A receiver dispatches a CONTROL frame by its hyphence type string,
so inventory-list frames and `drtp-*` frames coexist on the same channel.

#### `drtp-capabilities-v1`

Sent by each peer as its first frame. Body (TOML):

    protocol-version   = 1
    role               = "server"            # "server" | "client"
    websocket          = true                # transport actually in use is ws
    inventory-list-type = "inventory_list-v2"
    hash-format        = "blake2b256"
    expand-edges       = true                # sender computes closure
    public-key         = "..."               # markl id, for attestation
    nonce              = "..."               # challenge nonce (see Authentication)
    signature          = "..."               # signature over peer's nonce

`protocol-version` MUST be present. A peer that receives a `protocol-version` it
does not support MUST reply with `drtp-error-v1` and close. The remaining fields
are advertisements; a peer MUST ignore fields it does not understand (forward
compatibility) and MUST NOT assume a feature is available unless the peer
advertised it.

The server's capabilities frame also carries its immutable public config as the
bootstrap document, removing the separate `GET /config-immutable` round trip
the HTTP backend requires: the field `immutable-config` holds the genesis config
as a nested typed hyphence document (or the client MAY request it explicitly,
see Fetch). [Implementation note: the reference implementation transmits the
immutable config as its own CONTROL frame immediately after capabilities, which
is simpler than nesting and keeps each frame a single typed blob.]

### Authentication

Authentication reuses the markl challenge/response of the HTTP backend, hoisted
to once per session. In its `drtp-capabilities-v1` frame each peer includes a
random `nonce`. Each peer's *next* frame (or the same frame, for the responder)
includes a `signature` over the other peer's nonce, produced with the repo
private key under purpose `PurposeRequestAuthResponseV1`, and its `public-key`.
A peer verifies the signature against the advertised public key before
proceeding. A client MAY pin the server's public key (rejecting a mismatch) or
accept it Trust-On-First-Use.

Object batches and blob payloads are content-addressed: every object carries its
own markl object digest and every blob is validated against the markl ID in its
`drtp-blob_header-v1`. A receiver MUST verify each blob's computed digest
against the announced ID and MUST reject a mismatch. This gives the protocol the
same end-to-end integrity the HTTP backend obtains from its body-digest trailer,
without a per-frame signature.

### Fetch (pull / clone)

Fetch corresponds to git `upload-pack`: the *server* is the sender. The
exchange is:

1.  **Handshake.** Client and server exchange `drtp-capabilities-v1` and verify
    authentication. The server sends its immutable config.

2.  **Want.** The client sends `drtp-want-v1` carrying the query string (the
    same doddish query the HTTP backend puts in `GET /query/...`) and the
    transfer options (e.g. `allow-merge-conflicts`, `exclude-blobs`).

3.  **Closure.** The server resolves the query to an inventory list of matching
    object tips (`MakeInventoryList`), then runs expand-edges over that list to
    obtain the full transitive closure: every dependency object (types, tags,
    referenced objects, type-script-discovered objects) and every reachable blob
    digest. This is the protocol's central use of the type system --- the sender
    computes a self-consistent graph so the receiver never imports an object
    whose type, tags, or referenced blobs are absent.

4.  **Have (optional).** If the client advertised `have`, it sends one or more
    `drtp-have-v1` frames enumerating the object identities and version digests
    it already holds. The server omits those objects from the object stream and
    omits blobs the client already has. Absent any `have`, the server sends the
    whole closure (the clone case).

5.  **Objects.** The server sends the closure's objects as one or more CONTROL
    frames of the `inventory_list` type. The client imports them through the
    ordinary importer (`ImportSeq` + `MakeImporter`), with the same merge /
    conflict handling as a local pull.

6.  **Blobs.** For each blob digest in the closure that the client lacks, the
    server sends a `drtp-blob_header-v1` CONTROL frame followed by the BLOB
    frame. The client validates the digest and writes the blob to its store.

7.  **Done.** The server sends `drtp-ack-v1` with `status = "complete"`. Either
    peer closes.

Because the closure is computed once and streamed in a single direction, a
fetch is one request/one response at the protocol level regardless of how many
objects and blobs move --- the reactive `201`/`417` retry loop disappears.

### Push (send)

Push corresponds to git `receive-pack`: the *client* is the sender. It is the
mirror of fetch:

1.  Handshake (as above).
2.  The client sends `drtp-want-v1` describing what it intends to push (its
    query), so the server can apply receive-side policy.
3.  The client computes the closure with expand-edges over its own store.
4.  The server MAY send `drtp-have-v1` to tell the client which objects/blobs it
    already has, so the client can skip them.
5.  The client streams objects, then the blobs the server lacks.
6.  The server imports and replies `drtp-ack-v1` (`status = "complete"`) or
    `drtp-error-v1` (e.g. a rejected merge conflict).

The existing CLI keeps its inverted-direction convention (push is a fetch with
local and remote swapped); the protocol makes the direction explicit via the
`role` capability and which peer sends `want`.

### Blob Streaming

A blob MAY be larger than the CONTROL frame maximum. The `drtp-blob_header-v1`
frame carries the blob's total `length` and markl `id`. The blob bytes then
follow as one BLOB frame whose `length` matches, OR --- when chunking is
negotiated --- as a sequence of BLOB frames whose lengths sum to the header
length, terminated by a zero-length BLOB frame. A receiver computes the blob
digest incrementally and verifies it against the header `id` once the announced
length is reached. Compression, if advertised in capabilities, is applied to the
blob payload and MUST be reflected in a `compression` field on the header.

### Versioning and Evolution

The protocol's version surface is deliberately small:

- `protocol-version` in `drtp-capabilities-v1` gates the overall exchange shape.
- Every other message is a typed hyphence document, so a new field is a
  non-breaking addition and a breaking change is a new type string with the old
  one still decodable. This is the same horizontal-versioning discipline the
  store uses for persisted blobs; the wire format gets it for free by reusing
  hyphence coders.
- The `inventory_list` type carried in object frames is the repository's
  configured list type, so object encoding tracks the store version without the
  protocol needing its own object schema.

A peer MUST NOT remove support for decoding an older control type string when it
bumps the protocol; old senders must remain understandable, mirroring the
store's "never drop old codec support" rule.

### Errors

A terminal error is a `drtp-error-v1` CONTROL frame:

    message     = "human readable description"
    status-code = 409                          # optional, HTTP-status-like

A peer that receives `drtp-error-v1` MUST surface the message and terminate the
session. Transient transport errors (connection reset, timeout) are handled by
the transport, not the protocol; a half-completed transfer leaves the receiver's
store unchanged because imports are atomic per the store's existing semantics.

### Security Considerations

- **Authentication** is per-session markl challenge/response (see
  Authentication); a public server (read-only) MAY waive the client's challenge
  exactly as `remote_http`'s `Public` mode does, but MUST NOT waive it for a
  push.
- **Integrity** is content-addressing: objects carry their own digests and blobs
  are verified against announced markl IDs. A man-in-the-middle cannot
  substitute object or blob content without detection.
- **Confidentiality** is the transport's responsibility. WebSocket transport
  SHOULD run over TLS (`wss://`); the protocol does not encrypt payloads itself,
  though blobs MAY already be encrypted at rest by the blob store.
- **Resource exhaustion** is bounded by the frame maximum and by the receiver's
  freedom to abort a session whose closure exceeds policy.

### Compatibility and Migration

`sierra/remote_http` is retained unchanged. The new protocol lives in
`sierra/remote_proto`. A new remote connection type, `url-websocket`, selects
the WebSocket transport; `serve` gains the `/drtp` endpoint when the new
protocol is enabled. Existing remotes and the stdio/SSH/unix-socket/URL
transports continue to use `remote_http` until repointed. Because WebSocket
upgrade is optional and negotiated, a new client talking to an old server
degrades to the HTTP backend rather than failing.

## Resolved Decisions

1.  **Typed hyphence docs as the wire format, not a bespoke pack.** Reusing
    hyphence gives versioning, type dispatch, and signable/content-addressed
    framing for free, and lets object batches be the exact inventory-list stream
    the store already produces. A git-style binary packfile would be denser but
    would duplicate machinery and forfeit free versioning.
2.  **Sender-computed closure via expand-edges.** The sender, which has the
    whole store, computes the transitive closure once. This removes the reactive
    missing-blob loop and guarantees referential completeness, matching what the
    local-to-local pull already does.
3.  **Length-prefixed frames over hyphence boundaries as the outer container.**
    Avoids unbounded boundary scanning and lets opaque blob bytes share the
    channel with control messages.
4.  **Optional, negotiated WebSocket upgrade.** WebSocket is one transport among
    several; the protocol is defined over a byte stream so the same session code
    runs over stdio, sockets, TCP, and WebSocket. A failed upgrade degrades to
    the HTTP backend.
5.  **Hard fork, coexistence.** `remote_http` stays; `remote_proto` is new. No
    behavior change for existing deployments until they opt in.

## Open Questions

- **Capability-driven auto-upgrade from a stored remote.** Whether a remote
  stored as a plain `url` should probe `/drtp` and silently upgrade, or whether
  the WebSocket transport must be selected explicitly via `url-websocket`. v1
  selects it explicitly; auto-probing is deferred.
- **Multiplexing multiple transfers per session.** v1 is one transfer per
  session. A future version MAY add stream IDs to frames to allow concurrent
  fetch and push, at the cost of framing complexity.
- **Delta encoding of objects/blobs.** git sends deltas; v1 sends whole objects
  and whole blobs (deduplicated by content address). Delta transfer is deferred.

## More Information

- RFC 0001: Hyphence Format --- the wire representation for every message.
- RFC 0002: Markl ID Format --- addressing, locks, and signatures.
- `docs/plans/2026-03-21-edge-explorer-design.md` --- the expand-edges design
  the closure computation builds on.
