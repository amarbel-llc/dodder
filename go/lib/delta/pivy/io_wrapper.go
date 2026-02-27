package pivy

import (
	"crypto/ecdh"
	"io"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
	"filippo.io/age"
)

type IOWrapper struct {
	RecipientPubkey *ecdh.PublicKey
	DecryptECDH     ECDHFunc
}

func (iow *IOWrapper) WrapWriter(w io.Writer) (io.WriteCloser, error) {
	out, err := age.Encrypt(w, &Recipient{Pubkey: iow.RecipientPubkey})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return out, nil
}

func (iow *IOWrapper) WrapReader(r io.Reader) (io.ReadCloser, error) {
	identity := &Identity{ecdhFunc: iow.DecryptECDH}

	out, err := age.Decrypt(r, identity)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return ohio.NopCloser(out), nil
}
