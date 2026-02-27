package pivy

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"filippo.io/age"
	"golang.org/x/crypto/chacha20poly1305"
)

const StanzaTypePivyEcdhP256 = "pivy-ecdh-p256"

type Recipient struct {
	Pubkey *ecdh.PublicKey
}

func (r *Recipient) Wrap(fileKey []byte) ([]*age.Stanza, error) {
	ephemeralPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	sharedSecret, err := ephemeralPriv.ECDH(r.Pubkey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
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

	wrappedKey := aead.Seal(nil, make([]byte, chacha20poly1305.NonceSize), fileKey, nil)

	ephPubBytes := compressP256Point(ephemeralPriv.PublicKey())

	stanza := &age.Stanza{
		Type: StanzaTypePivyEcdhP256,
		Args: []string{
			base64.RawStdEncoding.EncodeToString(ephPubBytes),
			base64.RawStdEncoding.EncodeToString(nonce),
		},
		Body: wrappedKey,
	}

	return []*age.Stanza{stanza}, nil
}

func deriveWrappingKey(sharedSecret, nonce []byte) []byte {
	h := sha512.New()
	h.Write(sharedSecret)
	h.Write(nonce)
	sum := h.Sum(nil)
	return sum[:32]
}

func compressP256Point(pub *ecdh.PublicKey) []byte {
	// ecdh.PublicKey.Bytes() returns uncompressed point (0x04 || x || y)
	raw := pub.Bytes()
	// Compressed: 0x02 or 0x03 prefix (based on y parity) || x
	x := raw[1:33]
	y := raw[33:65]
	compressed := make([]byte, 33)
	if y[31]&1 == 0 {
		compressed[0] = 0x02
	} else {
		compressed[0] = 0x03
	}
	copy(compressed[1:], x)
	return compressed
}
