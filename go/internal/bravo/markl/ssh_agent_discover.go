package markl

import (
	"crypto/ed25519"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func listAgentKeys(socketEnvVar string) ([]*agent.Key, error) {
	socket := os.Getenv(socketEnvVar)
	if socket == "" {
		return nil, errors.Errorf("%s not set", socketEnvVar)
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to agent at %s", socketEnvVar)
	}
	defer conn.Close()

	keys, err := agent.NewClient(conn).List()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list keys from %s", socketEnvVar)
	}

	return keys, nil
}

func parseSSHPublicKey(key *agent.Key) (ssh.CryptoPublicKey, error) {
	parsed, err := ssh.ParsePublicKey(key.Marshal())
	if err != nil {
		return nil, errors.Wrap(err)
	}

	cryptoPub, ok := parsed.(ssh.CryptoPublicKey)
	if !ok {
		return nil, errors.Errorf("key does not implement CryptoPublicKey")
	}

	return cryptoPub, nil
}

func DiscoverSSHAgentEd25519Keys() ([]Id, error) {
	discovered, err := DiscoverSSHAgentEd25519KeysVerbose()
	if err != nil {
		return nil, err
	}

	ids := make([]Id, len(discovered))
	for i, dk := range discovered {
		ids[i] = dk.Id
	}

	return ids, nil
}

func DiscoverSSHAgentEd25519KeysVerbose() ([]DiscoveredKey, error) {
	keys, err := listAgentKeys("SSH_AUTH_SOCK")
	if err != nil {
		return nil, err
	}

	var discovered []DiscoveredKey

	for _, key := range keys {
		if key.Type() != ssh.KeyAlgoED25519 {
			continue
		}

		parsed, err := parseSSHPublicKey(key)
		if err != nil {
			continue
		}

		pubKey, ok := parsed.CryptoPublicKey().(ed25519.PublicKey)
		if !ok {
			continue
		}

		var id Id
		if err := id.SetMarklId(FormatIdEd25519SSH, []byte(pubKey)); err != nil {
			continue
		}

		discovered = append(discovered, DiscoveredKey{
			Id:      id,
			KeyType: key.Type(),
			Comment: key.Comment,
		})
	}

	return discovered, nil
}

func DiscoverSSHAgentECDHKeys() ([]Id, error) {
	discovered, err := DiscoverSSHAgentECDHKeysVerbose()
	if err != nil {
		return nil, err
	}

	ids := make([]Id, len(discovered))
	for i, dk := range discovered {
		ids[i] = dk.Id
	}

	return ids, nil
}

func DiscoverSSHAgentECDHKeysVerbose() ([]DiscoveredKey, error) {
	keys, err := listAgentKeys("SSH_AUTH_SOCK")
	if err != nil {
		return nil, err
	}

	return discoverEcdsaP256KeysFromAgentKeys(keys)
}
