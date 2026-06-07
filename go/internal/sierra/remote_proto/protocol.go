// Package remote_proto implements the dodder remote transfer protocol
// (drtp), specified in docs/rfcs/0004-remote-transfer-protocol.md.
//
// It is a hard fork of sierra/remote_http: rather than a sequence of REST
// calls glued together by a reactive missing-blob retry loop, drtp is a
// session protocol. One byte stream carries a whole transfer from a
// capability handshake through a single, sender-computed object and blob
// stream. The exchange takes its shape from git's fetch/push: capability
// advertisement, want/have negotiation, and a batched object stream.
//
// Every control and payload message on the wire is a typed hyphence
// document (RFC 0001), so the protocol inherits hyphence's type-string
// versioning and markl-id (RFC 0002) content addressing rather than a
// bespoke pack format. The sender computes the transfer's transitive
// closure up front with the type system's expand-edges traversal
// (sku.EdgeExplorer), so a fetch delivers a complete, self-consistent
// object graph in one pass.
//
// The session runs over any reliable, ordered, bidirectional byte stream:
// stdio, a unix socket, TCP, or — the motivating addition — a WebSocket
// upgraded from HTTP (see transport_websocket.go). WebSocket support is
// optional and negotiated.
package remote_proto

// ProtocolVersion is the drtp wire-protocol version advertised in the
// capabilities handshake. Bumped only on an incompatible change to the
// overall exchange shape; additive changes ride hyphence type-string
// versioning instead (see the RFC, "Versioning and Evolution").
const ProtocolVersion = 1

// PathTransfer is the HTTP path a websocket-capable server exposes for the
// upgrade. A client requesting websocket transport issues the upgrade here.
const PathTransfer = "/drtp"

// PathHealthz mirrors remote_http's liveness route so probes need no
// upgrade.
const PathHealthz = "/healthz"

// Control-message type strings. Each is the hyphence type of a control
// frame's typed document. New fields are additive (decoded leniently);
// an incompatible change introduces a new "-vN" string and keeps the old
// one decodable, per horizontal versioning.
const (
	TypeCapabilities = "drtp-capabilities-v1"
	TypeWant         = "drtp-want-v1"
	TypeManifest     = "drtp-manifest-v1"
	TypeHave         = "drtp-have-v1"
	TypeBlobHeader   = "drtp-blob_header-v1"
	TypeAck          = "drtp-ack-v1"
	TypeError        = "drtp-error-v1"
)

// Transfer directions carried in a want frame. Fetch makes the server the
// sender (pull/clone); push makes the client the sender.
const (
	DirectionFetch = "fetch"
	DirectionPush  = "push"
)

// Roles advertised in a capabilities frame.
const (
	RoleClient = "client"
	RoleServer = "server"
)

// Ack statuses.
const (
	StatusComplete = "complete"
)

// frameKind identifies how a frame's payload is interpreted. It is the
// single leading byte of every frame (see frame.go).
type frameKind byte

const (
	// frameKindControl payloads are a single typed hyphence control
	// document (one of the Type* strings above).
	frameKindControl frameKind = 0x01
	// frameKindObjects payloads are an inventory_list hyphence stream,
	// encoded by inventory_list_coders.Closet — the same bytes used for
	// on-disk and HTTP object transfer.
	frameKindObjects frameKind = 0x02
	// frameKindBlob payloads are the raw bytes of one content-addressed
	// blob. A blob frame is always immediately preceded by a control
	// frame of type TypeBlobHeader announcing its markl id and length.
	frameKindBlob frameKind = 0x03
)

// maxControlFrameLen bounds a single control or objects frame so a peer
// cannot exhaust memory with an oversized length prefix. Blob payloads are
// bounded by their announced header length and streamed.
const maxControlFrameLen = 64 << 20 // 64 MiB
