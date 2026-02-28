package markl

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/pivy"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func DiscoverPivyAgentECDHKeys() ([]Id, error) {
	socket := os.Getenv("PIVY_AUTH_SOCK")
	if socket == "" {
		return nil, errors.Errorf("PIVY_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to pivy-agent")
	}
	defer conn.Close()

	keys, err := agent.NewClient(conn).List()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list pivy-agent keys")
	}

	var ids []Id

	for _, key := range keys {
		if key.Type() != "ecdsa-sha2-nistp256" {
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

		ecdhPub, err := ecdhPubFromCryptoKey(cryptoPub.CryptoPublicKey())
		if err != nil {
			continue
		}

		compressed := pivy.CompressP256Point(ecdhPub)

		var id Id
		if err := id.SetMarklId(FormatIdPivyEcdhP256Pub, compressed); err != nil {
			continue
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func ecdhPubFromCryptoKey(pub interface{}) (*ecdh.PublicKey, error) {
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		return k.ECDH()
	default:
		return nil, errors.Errorf("unsupported key type: %T", pub)
	}
}
