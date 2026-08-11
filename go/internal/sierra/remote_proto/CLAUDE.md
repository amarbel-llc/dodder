# remote_proto

The drtp remote transfer protocol: a session protocol that streams a
sender-computed expand-edges closure, optionally over a websocket. It is a
**hard fork** of `sierra/remote_http` (which is retained and unchanged); the
two coexist. Specified in `docs/rfcs/0004-remote-transfer-protocol.md`.

## Why a fork

`remote_http` is reactive (REST calls + a missing-blob retry loop), has no
version negotiation, and re-handshakes per request. drtp handshakes once per
session, negotiates capabilities, computes the whole transfer up front with
the type system's `expand-edges`, and is framed independently of any one
transport so it can ride a websocket.

## Layers (bottom to top)

- `protocol.go` — version, frame kinds, control type strings, constants.
- `frame.go` — length-prefixed frames: `[kind:1][len:uint32 BE][payload]`.
- `control.go` — control messages: one `control` envelope encoded as a typed
  hyphence document (`! drtp-*-v1` metadata + JSON body). Dispatch is by the
  hyphence type string.
- `auth.go` / `handshake.go` — per-session markl challenge/response, reusing
  the same purposes as remote_http. `handshake.go` is decoupled from
  `*local_working_copy.Repo` (takes a `keys` struct) so it is unit-tested
  with generated keys.
- `session.go` — buffered framing over an `io.ReadWriteCloser`.
- `edges.go` — `expandEdges`, a fork of `local_working_copy.expandEdges`,
  driving `store.MakeEdgeExplorer` to compute the transitive closure.
- `transfer.go` — `sendClosure` (sender) / `receiveClosure` (receiver). The
  sender streams blobs first, then the object batch, then a completion ack.
  `receiveClosure` has an opt-in staging mode (`bufferedObjectsOut`): when set
  it decodes the object batch into a buffer for `clone -script` to transform
  and re-sign instead of importing it inline, skipping the merge negotiator
  (a fresh clone has no history). Blobs still stream to the local store, so
  every non-scripted receiver passes nil and is unaffected (dodder#396).
- `server.go` / `client.go` — repo-backed wrappers. `Server.ServeConn` runs
  one session; `Client.Fetch` / `Client.Push` initiate one.
- `transport_websocket.go` — `Server.Serve` (HTTP listener, `/drtp` upgrade,
  `/healthz`), `Server.ServeStdio`, and `DialWebSocket`. Uses
  `coder/websocket` + `websocket.NetConn` to present the upgraded connection
  as a `net.Conn`, so the same session code runs over websocket, stdio, TCP,
  and the in-memory `net.Pipe` used by the tests.

## Gotchas learned the hard way

- **Object batches MUST use the *typed* writer.** `sendObjects` calls
  `Closet.WriteTypedBlobToWriter` (not `WriteBlobToWriter`) so the stream
  carries its `! inventory_list-vN` line; the receiver's
  `AllDecodedObjectsFromStream` reads that type to pick the decoder. Using
  the untyped writer yields `no coders available for type: ""` on import.
- **Never `errors.Wrap` a bare `io.EOF`** (the dewey errors package panics).
  Frame readers return raw `io.EOF`; `readControlExpecting` converts it to a
  descriptive error before any wrap.
- **Disable the websocket read limit** (`conn.SetReadLimit(-1)`) on both ends
  — a flushed objects/blob frame can exceed the 32 KiB default message size.
- **Blobs stream before objects** so the importer finds every blob already on
  disk; no `RemoteBlobStore` is wired into the receiver's importer.
- **Blobs are streamed in chunks, never buffered whole.** `session.writeBlob`
  emits a `blob_header` then `blobChunkSize` BLOB frames ended by a zero-length
  terminator; `recvBlob` copies chunks straight into the blob writer and
  verifies the digest after the terminator. The header carries no length.
- **Have-negotiation precedes the stream.** The sender sends `drtp-manifest-v1`
  (every closure blob it holds); the receiver replies `drtp-have-v1` (the
  subset it already has); the sender streams only the rest. Both
  `sendClosure`/`receiveClosure` perform this exchange before the blob/object
  frames, so the reads stay in lockstep regardless of direction.
- **Blob compression (zstd) is negotiated in capabilities.** Each peer
  advertises `supportedCompression`; the *sender* calls `negotiateCompression`
  on the peer's advertised value and, when both agree, compresses each blob's
  chunk stream through `DataDog/zstd` (the dodder/madder zstd lib, already in
  the build closure — cgo). The digest is verified over the **decompressed**
  bytes, so content addressing is unaffected. `blobFrameWriter` /
  `blobFrameReader` (session.go) adapt the frame stream to the
  encoder/decoder.
- **Every transfer ships full object history for merge negotiation, both
  directions (#299).** `sendClosure` always expands the closure to every
  version of each object (`expandListToObjectHistory` → `ReadObjectHistory`)
  before edges/objects, and `receiveClosure` always builds a
  `local_working_copy.ParentNegotiatorInBand` from the single objects frame
  (`addObjectsToNegotiator`), setting it on `importerOptions` before importing
  so the merge finds the common ancestor by TAI. This is symmetric: on a fetch
  the server sends and the client receives; on a push the client sends and the
  server receives — `dst` is the receiving repo either way, so the same code
  resolves the base for both. It is the lock-step protocol's substitute for the
  out-of-band history query the HTTP transport uses (`/object-history`): there
  is no in-session way to ask the peer for history, so the sender always
  volunteers it. Do not gate this by direction or "optimize" it back to
  latest-only: blobs are already deduped by have-negotiation, so the only extra
  bytes are historical object metadata, and without them the receiver cannot
  distinguish a fast-forward from a divergence. (HTTP/stdio *push* merge
  resolution is separately still open on #299, blocked on #166.)

## CLI surface

- `serve-proto [network] [address]` (`commands_dodder/serve_proto.go`) —
  `-public`, `-handshake`, `-` for stdio.
- `remote_connection_types.TypeUrlWebsocket` (`url-websocket`) — selects this
  backend in `pull`/`push` via `-remote-connection-type`. `remote-add`
  registers a websocket url remote Trust-On-First-Use (no connect at add
  time, since serve-proto does not serve the remote_http REST surface).

## Tests

`*_test.go` here run with plain `go test` (no repo needed): control/frame
round-trips, the markl auth handshake over `net.Pipe`, and a full handshake
over a real websocket via `httptest`. End-to-end pull/push over websocket is
covered by `zz-tests_bats/current_version/serve_proto.bats`.
