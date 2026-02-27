package markl

import (
	"crypto/ed25519"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func DiscoverSSHAgentEd25519Keys() ([]Id, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, errors.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to SSH agent")
	}
	defer conn.Close()

	keys, err := agent.NewClient(conn).List()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list SSH agent keys")
	}

	var ids []Id

	for _, key := range keys {
		if key.Type() != ssh.KeyAlgoED25519 {
			continue
		}

		parsed, err := ssh.ParsePublicKey(key.Marshal())
		if err != nil {
			continue
		}

		cryptoPub, ok := parsed.(ssh.CryptoPublicKey)
		if !ok {
			continue
		}

		pubKey, ok := cryptoPub.CryptoPublicKey().(ed25519.PublicKey)
		if !ok {
			continue
		}

		var id Id
		if err := id.SetMarklId(FormatIdEd25519SSH, []byte(pubKey)); err != nil {
			continue
		}

		ids = append(ids, id)
	}

	return ids, nil
}
