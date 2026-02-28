package markl

import (
	"crypto"
	"crypto/ed25519"
	"io"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

var sshFormatOnce sync.Once

func RegisterSSHEd25519Format(signer crypto.Signer) {
	sshFormatOnce.Do(func() {
		formats[FormatIdEd25519SSH] = FormatSec{
			Id:          FormatIdEd25519SSH,
			Size:        ed25519.PublicKeySize,
			PubFormatId: FormatIdEd25519Pub,
			GetPublicKey: func(_ domain_interfaces.MarklId) ([]byte, error) {
				pub, ok := signer.Public().(ed25519.PublicKey)
				if !ok {
					return nil, errors.Errorf("SSH agent signer public key is not Ed25519")
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

func resetSSHFormatForTesting() {
	makeStubSSHFormat()
	sshFormatOnce = sync.Once{}
}

func makeStubSSHFormat() {
	formats[FormatIdEd25519SSH] = FormatSec{
		Id:          FormatIdEd25519SSH,
		Size:        ed25519.PublicKeySize,
		PubFormatId: FormatIdEd25519Pub,
		GetPublicKey: func(_ domain_interfaces.MarklId) ([]byte, error) {
			return nil, errors.Wrap(ErrEd25519SSHAgentNotConnected)
		},
		SigFormatId: FormatIdEd25519Sig,
		Sign: func(
			_, _ domain_interfaces.MarklId,
			_ io.Reader,
		) ([]byte, error) {
			return nil, errors.Wrap(ErrEd25519SSHAgentNotConnected)
		},
	}
}
