package markl

import (
	"crypto/ed25519"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func RegisterPIVEd25519Format(signer PIVSigner) {
	_, alreadyExists := formats[FormatIdEd25519PIV]
	if alreadyExists {
		return
	}

	makeFormat(FormatSec{
		Id:          FormatIdEd25519PIV,
		Size:        PIVReferenceSize,
		PubFormatId: FormatIdEd25519Pub,
		GetPublicKey: func(_ domain_interfaces.MarklId) ([]byte, error) {
			pub, ok := signer.Public().(ed25519.PublicKey)
			if !ok {
				return nil, errors.Errorf("PIV signer public key is not Ed25519")
			}
			return []byte(pub), nil
		},
		SigFormatId: FormatIdEd25519Sig,
		Sign: func(
			sec, mes domain_interfaces.MarklId,
			readerRand io.Reader,
		) ([]byte, error) {
			return signer.Sign(readerRand, mes.GetBytes(), &ed25519.Options{})
		},
	})
}
