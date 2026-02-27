package genesis_configs

import (
	"crypto/ed25519"
	"io"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

var (
	sshConnectOnce sync.Once
	sshConnectErr  error
	sshConn        io.Closer
)

func connectSSHSignerIfNecessary(privateKey markl.Id) error {
	format := privateKey.GetMarklFormat()
	if format == nil {
		return nil
	}

	if format.GetMarklFormatId() != markl.FormatIdEd25519SSH {
		return nil
	}

	sshConnectOnce.Do(func() {
		pubKey := ed25519.PublicKey(privateKey.GetBytes())
		signer, closer, err := markl.ConnectSSHAgentSigner(pubKey)
		if err != nil {
			sshConnectErr = errors.Wrap(err)
			return
		}

		sshConn = closer
		markl.RegisterSSHEd25519Format(signer)
	})

	return sshConnectErr
}
