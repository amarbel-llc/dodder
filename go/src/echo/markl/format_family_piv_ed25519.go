package markl

import (
	"crypto/ed25519"
	"io"
	"sync"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

var pivFormatOnce sync.Once

func RegisterPIVEd25519Format(signer PIVSigner) {
	pivFormatOnce.Do(func() {
		// Replace the stub registered in init() with a live signer-backed format.
		formats[FormatIdEd25519PIV] = FormatSec{
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
		}
	})
}

func resetPIVFormatForTesting() {
	// Restore the stub format so the next RegisterPIVEd25519Format call
	// can replace it with a new signer.
	makeStubPIVFormat()
	pivFormatOnce = sync.Once{}
}

func makeStubPIVFormat() {
	formats[FormatIdEd25519PIV] = FormatSec{
		Id:          FormatIdEd25519PIV,
		Size:        PIVReferenceSize,
		PubFormatId: FormatIdEd25519Pub,
		GetPublicKey: func(_ domain_interfaces.MarklId) ([]byte, error) {
			return nil, errors.Wrap(ErrEd25519PIVSignerConnectionNotEstablished)
		},
		SigFormatId: FormatIdEd25519Sig,
		Sign: func(
			_, _ domain_interfaces.MarklId,
			_ io.Reader,
		) ([]byte, error) {
			return nil, errors.Wrap(ErrEd25519PIVSignerConnectionNotEstablished)
		},
	}
}
