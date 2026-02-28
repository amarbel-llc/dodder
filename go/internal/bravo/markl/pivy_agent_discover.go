package markl

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/pivy"

	"golang.org/x/crypto/ssh/agent"
)

func DiscoverPivyAgentECDHKeys() ([]Id, error) {
	discovered, err := DiscoverPivyAgentECDHKeysVerbose()
	if err != nil {
		return nil, err
	}

	ids := make([]Id, len(discovered))
	for i, dk := range discovered {
		ids[i] = dk.Id
	}

	return ids, nil
}

func DiscoverPivyAgentECDHKeysVerbose() ([]DiscoveredKey, error) {
	var keys []*agent.Key
	var err error

	if os.Getenv("PIVY_AUTH_SOCK") != "" {
		keys, err = listAgentKeys("PIVY_AUTH_SOCK")
	} else {
		keys, err = listAgentKeys("SSH_AUTH_SOCK")
	}

	if err != nil {
		return nil, err
	}

	return discoverECDHKeysFromAgentKeys(keys)
}

func discoverECDHKeysFromAgentKeys(keys []*agent.Key) ([]DiscoveredKey, error) {
	var discovered []DiscoveredKey

	for _, key := range keys {
		if key.Type() != "ecdsa-sha2-nistp256" {
			continue
		}

		parsed, err := parseSSHPublicKey(key)
		if err != nil {
			continue
		}

		ecdhPub, err := ecdhPubFromCryptoKey(parsed.CryptoPublicKey())
		if err != nil {
			continue
		}

		compressed := pivy.CompressP256Point(ecdhPub)

		var id Id
		if err := id.SetMarklId(FormatIdPivyEcdhP256Pub, compressed); err != nil {
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

func ecdhPubFromCryptoKey(pub interface{}) (*ecdh.PublicKey, error) {
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		return k.ECDH()
	default:
		return nil, errors.Errorf("unsupported key type: %T", pub)
	}
}
