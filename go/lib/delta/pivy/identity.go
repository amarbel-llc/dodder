package pivy

import (
	"crypto/ecdh"
	"crypto/elliptic"
	"encoding/base64"
	"math/big"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"filippo.io/age"
	"golang.org/x/crypto/chacha20poly1305"
)

// ECDHFunc performs ECDH given an ephemeral public key and returns the shared
// secret. For software keys this is a local operation. For agent-backed keys
// this calls the pivy-agent extension.
type ECDHFunc func(ephemeralPubkey []byte) (sharedSecret []byte, err error)

type Identity struct {
	ecdhFunc ECDHFunc
}

func (id *Identity) Unwrap(stanzas []*age.Stanza) ([]byte, error) {
	for _, s := range stanzas {
		if s.Type != StanzaTypePivyEcdhP256 {
			continue
		}

		fileKey, err := id.tryUnwrap(s)
		if err != nil {
			continue // trial decryption: AEAD failure means wrong recipient
		}

		return fileKey, nil
	}

	return nil, age.ErrIncorrectIdentity
}

func (id *Identity) tryUnwrap(s *age.Stanza) ([]byte, error) {
	if len(s.Args) != 2 {
		return nil, errors.Errorf("expected 2 args, got %d", len(s.Args))
	}

	ephPubBytes, err := base64.RawStdEncoding.DecodeString(s.Args[0])
	if err != nil {
		return nil, errors.Wrap(err)
	}

	nonce, err := base64.RawStdEncoding.DecodeString(s.Args[1])
	if err != nil {
		return nil, errors.Wrap(err)
	}

	sharedSecret, err := id.ecdhFunc(ephPubBytes)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	wrappingKey := deriveWrappingKey(sharedSecret, nonce)

	// Zeroize shared secret
	for i := range sharedSecret {
		sharedSecret[i] = 0
	}

	aead, err := chacha20poly1305.New(wrappingKey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	// Zeroize wrapping key
	for i := range wrappingKey {
		wrappingKey[i] = 0
	}

	fileKey, err := aead.Open(nil, make([]byte, chacha20poly1305.NonceSize), s.Body, nil)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return fileKey, nil
}

func softwareECDH(privKey *ecdh.PrivateKey) ECDHFunc {
	return func(ephPubBytes []byte) ([]byte, error) {
		ephPub, err := decompressP256Point(ephPubBytes)
		if err != nil {
			return nil, err
		}

		return privKey.ECDH(ephPub)
	}
}

// SoftwareECDHForTesting returns an ECDHFunc using a software private key.
// Intended for tests that need an IOWrapper without a running pivy-agent.
func SoftwareECDHForTesting(privKey *ecdh.PrivateKey) ECDHFunc {
	return softwareECDH(privKey)
}

func decompressP256Point(compressed []byte) (*ecdh.PublicKey, error) {
	if len(compressed) != 33 {
		return nil, errors.Errorf(
			"invalid compressed point length: %d",
			len(compressed),
		)
	}

	x := new(big.Int).SetBytes(compressed[1:33])
	curve := elliptic.P256()
	y := decompressY(curve, x, compressed[0] == 0x03)

	if y == nil {
		return nil, errors.Errorf("invalid point: not on curve")
	}

	// Build uncompressed point: 0x04 || x || y
	uncompressed := make([]byte, 65)
	uncompressed[0] = 0x04
	xBytes := x.Bytes()
	yBytes := y.Bytes()
	copy(uncompressed[1+32-len(xBytes):33], xBytes)
	copy(uncompressed[33+32-len(yBytes):65], yBytes)

	return ecdh.P256().NewPublicKey(uncompressed)
}

func decompressY(curve elliptic.Curve, x *big.Int, odd bool) *big.Int {
	params := curve.Params()
	p := params.P

	// y^2 = x^3 - 3x + b (mod p)
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	x3.Mod(x3, p)

	threeX := new(big.Int).Mul(big.NewInt(3), x)
	threeX.Mod(threeX, p)

	y2 := new(big.Int).Sub(x3, threeX)
	y2.Add(y2, params.B)
	y2.Mod(y2, p)

	// y = sqrt(y^2) mod p
	y := new(big.Int).ModSqrt(y2, p)
	if y == nil {
		return nil
	}

	// Choose the correct root based on parity
	if odd != (y.Bit(0) == 1) {
		y.Sub(p, y)
	}

	return y
}
