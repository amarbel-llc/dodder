package remote_proto

import (
	"bytes"
	"encoding/json"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	mad_ids "github.com/amarbel-llc/madder/go/pkgs/ids"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// control is the single envelope carrying every drtp control message. The
// hyphence type of the frame (one of the Type* strings) selects the
// message's meaning; only the fields relevant to that type are populated.
// A single envelope keeps decode-time dispatch trivial: read the hyphence
// type, then switch — no per-type Go struct or speculative decode.
//
// The body is JSON (stdlib, dependency-free, and well-precedented here —
// server_mcp.go speaks JSON-RPC) wrapped in hyphence typed-document
// framing, so the message is a typed hyphence doc on the wire while the
// implementation avoids pulling a TOML codec into the nix build closure.
type control struct {
	// capabilities
	ProtocolVersion   int    `json:"protocol_version,omitempty"`
	Role              string `json:"role,omitempty"`
	WebSocket         bool   `json:"websocket,omitempty"`
	InventoryListType string `json:"inventory_list_type,omitempty"`
	HashFormat        string `json:"hash_format,omitempty"`
	ExpandEdges       bool   `json:"expand_edges,omitempty"`
	Public            bool   `json:"public,omitempty"`
	PublicKey         string `json:"public_key,omitempty"`
	Nonce             string `json:"nonce,omitempty"`
	Signature         string `json:"signature,omitempty"`

	// public seed (RFC-0004 "The public seed"): the genesis-config fields
	// drtp actually needs. RepoId is provenance for the attested PublicKey;
	// StoreVersion is a fail-fast compatibility guard (a receiver rejects a
	// peer whose store version it cannot decode). Not decode-config — the
	// object/blob streams are self-describing (see ADR-0005).
	RepoId       string `json:"repo_id,omitempty"`
	StoreVersion string `json:"store_version,omitempty"`

	// want
	Direction           string `json:"direction,omitempty"`
	Query               string `json:"query,omitempty"`
	AllowMergeConflicts bool   `json:"allow_merge_conflicts,omitempty"`
	ExcludeBlobs        bool   `json:"exclude_blobs,omitempty"`

	// manifest: every blob digest the sender holds for this transfer
	Blobs []string `json:"blobs,omitempty"`

	// have: the subset of the manifest the receiver already holds (and so
	// the sender SHOULD NOT stream)
	Objects   []string `json:"objects,omitempty"`
	HaveBlobs []string `json:"have_blobs,omitempty"`

	// blob_header
	BlobId      string `json:"blob_id,omitempty"`
	BlobLength  int64  `json:"blob_length,omitempty"`
	Compression string `json:"compression,omitempty"`

	// ack
	Status string `json:"status,omitempty"`

	// error
	Message    string `json:"message,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

// controlCoder encodes/decodes a control envelope as a typed hyphence
// document. Every known control type maps to the same JSON body coder; the
// type lives in the hyphence metadata line and is the single source of
// truth for dispatch.
var controlCoder = hyphence.CoderToTypedBlob[control]{
	Metadata: hyphence.TypedMetadataCoder[control]{},
	Blob:     hyphence.CoderTypeMapWithoutType[control](makeControlBlobCoders()),
}

func makeControlBlobCoders() map[string]interfaces.CoderBufferedReadWriter[*control] {
	jsonCoder := hyphence.CoderTommy[control, *control]{
		Decode: func(b []byte) (msg control, err error) {
			// An empty body is a valid message whose meaning is carried
			// entirely by its type (e.g. a bare ack).
			if len(bytes.TrimSpace(b)) == 0 {
				return msg, err
			}

			if err = json.Unmarshal(b, &msg); err != nil {
				err = errors.Wrap(err)
				return msg, err
			}

			return msg, err
		},
		Encode: func(msg control) (b []byte, err error) {
			if b, err = json.Marshal(msg); err != nil {
				err = errors.Wrap(err)
				return b, err
			}

			return b, err
		},
	}

	coders := make(map[string]interfaces.CoderBufferedReadWriter[*control])

	for _, typeString := range []string{
		TypeCapabilities,
		TypeWant,
		TypeManifest,
		TypeHave,
		TypeBlobHeader,
		TypeAck,
		TypeError,
	} {
		// Key on exactly what TypedBlob.Type.String() will produce so the
		// lookup in CoderTypeMapWithoutType matches regardless of the
		// "!"-prefix convention.
		coders[madType(typeString).String()] = jsonCoder
	}

	return coders
}

// madType builds the madder hyphence TypeStruct for a drtp control type
// string.
func madType(typeString string) mad_ids.TypeStruct {
	return ids.MustTypeStruct(typeString).ToMadder()
}

// encodeControl writes a control message of the given type to a byte slice
// as a typed hyphence document.
func encodeControl(typeString string, msg control) (b []byte, err error) {
	typedBlob := &hyphence.TypedBlob[control]{
		Type: madType(typeString),
		Blob: msg,
	}

	var buffer bytes.Buffer

	if _, err = controlCoder.EncodeTo(typedBlob, &buffer); err != nil {
		err = errors.Wrap(err)
		return b, err
	}

	return buffer.Bytes(), err
}

// decodeControl parses a typed hyphence control document, returning its
// type string (with the leading "!") and the decoded envelope.
func decodeControl(b []byte) (typeString string, msg control, err error) {
	typedBlob := &hyphence.TypedBlob[control]{}

	if _, err = controlCoder.DecodeFrom(typedBlob, bytes.NewReader(b)); err != nil {
		err = errors.Wrap(err)
		return typeString, msg, err
	}

	return typedBlob.Type.String(), typedBlob.Blob, err
}
