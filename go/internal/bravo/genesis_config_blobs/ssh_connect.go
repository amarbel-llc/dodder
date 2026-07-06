package genesis_config_blobs

import (
	"crypto/ed25519"
	"io"
	"sync"

	"github.com/amarbel-llc/piggy/go/pkgs/agent"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

var (
	sshConnectOnce sync.Once
	sshConnectErr  error
	sshConn        io.Closer
)

func connectSSHSignerIfNecessary(privateKey *markl.Id) error {
	format := privateKey.GetMarklFormat()
	if format == nil {
		return nil
	}

	if format.GetMarklFormatId() != markl.FormatIdEd25519SSH {
		return nil
	}

	sshConnectOnce.Do(func() {
		pubKey := ed25519.PublicKey(privateKey.GetBytes())
		signer, closer, err := agent.ConnectSSHAgentSigner(pubKey)
		if err != nil {
			sshConnectErr = errors.Wrap(err)
			return
		}

		sshConn = closer
		agent.RegisterSSHEd25519Format(signer)
		sshConnectErr = privateKey.ReloadFormat()
	})

	return sshConnectErr
}

var (
	ecdsaP256ConnectOnce sync.Once
	ecdsaP256ConnectErr  error
	ecdsaP256Conn        io.Closer
)

func connectEcdsaP256SignerIfNecessary(privateKey *markl.Id) error {
	format := privateKey.GetMarklFormat()
	if format == nil {
		return nil
	}

	if format.GetMarklFormatId() != markl.FormatIdEcdsaP256SSH {
		return nil
	}

	ecdsaP256ConnectOnce.Do(func() {
		compressed := privateKey.GetBytes()
		signer, closer, err := agent.ConnectEcdsaP256AgentSigner(compressed)
		if err != nil {
			ecdsaP256ConnectErr = errors.Wrap(err)
			return
		}

		ecdsaP256Conn = closer
		agent.RegisterEcdsaP256SSHFormat(signer)
		ecdsaP256ConnectErr = privateKey.ReloadFormat()
	})

	return ecdsaP256ConnectErr
}
