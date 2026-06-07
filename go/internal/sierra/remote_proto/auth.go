package remote_proto

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// Authentication reuses remote_http's markl challenge/response, hoisted to
// once per session (see the RFC, "Authentication"). Each peer advertises a
// random nonce in its capabilities frame; each peer signs the other's nonce
// with its repo private key and returns the signature plus its public key.
// A peer verifies the signature against the advertised public key before
// trusting the connection.

// generateNonce mints a fresh challenge nonce, matching the
// RoundTripperBufioWrappedSigner nonce in remote_http.
func generateNonce() (nonceString string, err error) {
	var nonce markl.Id

	if err = nonce.GeneratePrivateKey(
		nil,
		markl.FormatIdNonceSec,
		markl.PurposeRequestAuthChallengeV1,
	); err != nil {
		err = errors.Wrap(err)
		return nonceString, err
	}

	return nonce.String(), err
}

// signNonce signs the peer's nonce with priv under the same purpose
// remote_http uses for its challenge response.
func signNonce(
	priv mad_domain_interfaces.MarklId,
	nonceString string,
) (sigString string, err error) {
	var nonce markl.Id

	if err = nonce.Set(nonceString); err != nil {
		err = errors.Wrap(err)
		return sigString, err
	}

	var sig markl.Id

	if err = priv.Sign(
		nonce,
		&sig,
		markl.PurposeRequestAuthResponseV1,
	); err != nil {
		err = errors.Wrap(err)
		return sigString, err
	}

	return sig.String(), err
}

// verifyNonceSignature verifies that sigString is a signature over
// nonceString produced by the private key matching pubKeyString. An empty
// pubKeyString or sigString is treated as "no attestation offered" and
// returns errNoAttestation so the caller can decide whether that is
// acceptable (e.g. a public read server).
func verifyNonceSignature(
	pubKeyString, nonceString, sigString string,
) (err error) {
	if pubKeyString == "" || sigString == "" {
		return errNoAttestation
	}

	var pubKey markl.Id

	if err = pubKey.Set(pubKeyString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var nonce markl.Id

	if err = nonce.Set(nonceString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var sig markl.Id

	if err = sig.Set(sigString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = pubKey.Verify(nonce, sig); err != nil {
		err = errors.Wrapf(err, "challenge signature verification failed")
		return err
	}

	return err
}

var errNoAttestation = errors.Errorf("peer offered no attestation")
