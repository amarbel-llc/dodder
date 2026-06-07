package remote_proto

import (
	"bytes"

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
	listType string,
	pinnedPublicKey mad_domain_interfaces.MarklId,
) (serverCaps control, clientNonce string, err error) {
	if clientNonce, err = generateNonce(); err != nil {
		err = errors.Wrap(err)
		return serverCaps, clientNonce, err
	}

	if err = s.writeControl(TypeCapabilities, control{
		ProtocolVersion:   ProtocolVersion,
		Role:              RoleClient,
		InventoryListType: listType,
		ExpandEdges:       true,
		PublicKey:         self.publicString(),
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
	listType string,
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
		InventoryListType: listType,
		ExpandEdges:       true,
		Public:            public,
		PublicKey:         self.publicString(),
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
