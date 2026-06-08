package remote_proto

import (
	"bytes"

	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// keys is the keypair a peer uses for the per-session challenge/response.
// It is decoupled from *local_working_copy.Repo (the production source) so
// the handshake can be unit-tested with generated keys over net.Pipe.
type keys struct {
	private mad_domain_interfaces.MarklId
	public  mad_domain_interfaces.MarklId
}

// selfCaps carries the local repo's config-derived capability fields a peer
// advertises (beyond protocol-version / nonce / signature): the object
// inventory-list type and the public seed (RepoId provenance, StoreVersion
// guard). Bundled so the handshake stays decoupled from *Repo (tests pass
// literals) and a future seed field is a one-field change.
type selfCaps struct {
	listType     string
	repoId       string
	storeVersion string
}

// assertStoreVersionCompatible rejects a peer advertising a store version
// this build cannot decode. store_version.Version.Set returns
// ErrFutureStoreVersion when the parsed version exceeds VCurrent, so a
// future peer fails fast here rather than deep in import. An empty value
// (an older peer that does not advertise the seed) is accepted.
func assertStoreVersionCompatible(peerStoreVersion string) (err error) {
	if peerStoreVersion == "" {
		return err
	}

	var version store_version.Version

	if err = version.Set(peerStoreVersion); err != nil {
		err = errors.Wrapf(err, "remote store version %q is not supported", peerStoreVersion)
		return err
	}

	return err
}

func (k keys) publicString() string {
	if k.public == nil || k.public.IsNull() {
		return ""
	}

	return k.public.String()
}

// clientHandshake sends the client capabilities, then reads and verifies the
// server capabilities. It returns the server's capabilities (whose Nonce the
// caller signs into its want frame) so mutual attestation completes within
// the want exchange. The server MUST attest (sign the client's nonce);
// pinnedPublicKey, when non-nil, additionally pins the server identity.
func clientHandshake(
	s *session,
	self keys,
	caps selfCaps,
	pinnedPublicKey mad_domain_interfaces.MarklId,
) (serverCaps control, clientNonce string, err error) {
	if clientNonce, err = generateNonce(); err != nil {
		err = errors.Wrap(err)
		return serverCaps, clientNonce, err
	}

	if err = s.writeControl(TypeCapabilities, control{
		ProtocolVersion:   ProtocolVersion,
		Role:              RoleClient,
		InventoryListType: caps.listType,
		ExpandEdges:       true,
		Compression:       supportedCompression,
		PublicKey:         self.publicString(),
		RepoId:            caps.repoId,
		StoreVersion:      caps.storeVersion,
		Nonce:             clientNonce,
	}); err != nil {
		err = errors.Wrap(err)
		return serverCaps, clientNonce, err
	}

	if serverCaps, err = s.readControlExpecting(TypeCapabilities); err != nil {
		err = errors.Wrap(err)
		return serverCaps, clientNonce, err
	}

	if serverCaps.ProtocolVersion != ProtocolVersion {
		err = errors.Errorf(
			"unsupported remote protocol version %d (need %d)",
			serverCaps.ProtocolVersion,
			ProtocolVersion,
		)
		return serverCaps, clientNonce, err
	}

	// Fail fast if the server's store version is newer than this build can
	// decode (the client is the receiver on a fetch).
	if err = assertStoreVersionCompatible(serverCaps.StoreVersion); err != nil {
		err = errors.Wrap(err)
		return serverCaps, clientNonce, err
	}

	if pinnedPublicKey != nil && !pinnedPublicKey.IsNull() {
		if err = assertPublicKeyMatches(
			pinnedPublicKey,
			serverCaps.PublicKey,
		); err != nil {
			err = errors.Wrap(err)
			return serverCaps, clientNonce, err
		}
	}

	// The server must always attest by signing the client's nonce, exactly
	// as remote_http requires a signed challenge response.
	if err = verifyNonceSignature(
		serverCaps.PublicKey,
		clientNonce,
		serverCaps.Signature,
	); err != nil {
		err = errors.Wrapf(err, "verifying server attestation")
		return serverCaps, clientNonce, err
	}

	return serverCaps, clientNonce, err
}

// signWant produces the client's want frame, signing the server's nonce so
// the server can attest the client (required for a push, optional for a
// public fetch).
func signWant(
	self keys,
	serverNonce string,
	want control,
) (signed control, err error) {
	signed = want
	signed.PublicKey = self.publicString()

	if self.private != nil && !self.private.IsNull() && serverNonce != "" {
		if signed.Signature, err = signNonce(self.private, serverNonce); err != nil {
			err = errors.Wrap(err)
			return signed, err
		}
	}

	return signed, err
}

// serverHandshake reads the client capabilities, sends the server
// capabilities (signing the client's nonce), then reads the want and — unless
// the connection is a public read — verifies the client's attestation over
// the server's nonce.
func serverHandshake(
	s *session,
	self keys,
	caps selfCaps,
	public bool,
) (clientCaps, want control, err error) {
	if clientCaps, err = s.readControlExpecting(TypeCapabilities); err != nil {
		err = errors.Wrap(err)
		return clientCaps, want, err
	}

	if clientCaps.ProtocolVersion != ProtocolVersion {
		err = errors.Errorf(
			"unsupported client protocol version %d (need %d)",
			clientCaps.ProtocolVersion,
			ProtocolVersion,
		)
		s.writeError(err.Error(), 400)
		return clientCaps, want, err
	}

	// Fail fast if the client's store version is newer than this build can
	// decode (the server is the receiver on a push).
	if err = assertStoreVersionCompatible(clientCaps.StoreVersion); err != nil {
		err = errors.Wrap(err)
		s.writeError(err.Error(), 400)
		return clientCaps, want, err
	}

	var serverNonce string

	if serverNonce, err = generateNonce(); err != nil {
		err = errors.Wrap(err)
		return clientCaps, want, err
	}

	var serverSignature string

	if clientCaps.Nonce != "" && self.private != nil && !self.private.IsNull() {
		if serverSignature, err = signNonce(self.private, clientCaps.Nonce); err != nil {
			err = errors.Wrap(err)
			return clientCaps, want, err
		}
	}

	if err = s.writeControl(TypeCapabilities, control{
		ProtocolVersion:   ProtocolVersion,
		Role:              RoleServer,
		InventoryListType: caps.listType,
		ExpandEdges:       true,
		Compression:       supportedCompression,
		Public:            public,
		PublicKey:         self.publicString(),
		RepoId:            caps.repoId,
		StoreVersion:      caps.storeVersion,
		Nonce:             serverNonce,
		Signature:         serverSignature,
	}); err != nil {
		err = errors.Wrap(err)
		return clientCaps, want, err
	}

	if want, err = s.readControlExpecting(TypeWant); err != nil {
		err = errors.Wrap(err)
		return clientCaps, want, err
	}

	// A push (the client writes to us) always requires client attestation. A
	// public fetch may waive it, mirroring remote_http's Public read mode.
	requireClientAuth := want.Direction == DirectionPush || !public

	if requireClientAuth {
		if err = verifyNonceSignature(
			clientCaps.PublicKey,
			serverNonce,
			want.Signature,
		); err != nil {
			err = errors.Wrapf(err, "verifying client attestation")
			s.writeError(err.Error(), 401)
			return clientCaps, want, err
		}
	}

	return clientCaps, want, err
}

// negotiateCompression picks the blob compression a sender uses: the
// advertised algorithm when the peer also supports it, otherwise none.
func negotiateCompression(peerCompression string) string {
	if peerCompression == supportedCompression {
		return supportedCompression
	}

	return CompressionNone
}

func assertPublicKeyMatches(
	expected mad_domain_interfaces.MarklId,
	actualString string,
) (err error) {
	var actual markl.Id

	if err = actual.Set(actualString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if !bytes.Equal(expected.GetBytes(), actual.GetBytes()) {
		err = errors.Errorf(
			"remote public key mismatch: expected %q, got %q",
			expected,
			actualString,
		)
		return err
	}

	return err
}
