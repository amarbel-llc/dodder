package markl

import (
	"crypto"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"

	"github.com/go-piv/piv-go/v2/piv"
)

type PIVTokenInfo struct {
	GUID      [16]byte
	SlotId    byte
	PublicKey crypto.PublicKey
	Serial    uint32
	Card      string
}

func DiscoverPIVTokens() ([]PIVTokenInfo, error) {
	cards, err := piv.Cards()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list smart card readers")
	}

	var tokens []PIVTokenInfo

	for _, card := range cards {
		yk, err := piv.Open(card)
		if err != nil {
			continue
		}

		serial, err := yk.Serial()
		if err != nil {
			yk.Close()
			continue
		}

		// Encode serial as first 4 bytes of GUID, matching OpenPIVSigner.
		var guid [16]byte
		guid[0] = byte(serial >> 24)
		guid[1] = byte(serial >> 16)
		guid[2] = byte(serial >> 8)
		guid[3] = byte(serial)

		signingSlot := piv.SlotSignature // 0x9c

		cert, err := yk.Certificate(signingSlot)
		if err != nil {
			yk.Close()
			continue
		}

		tokens = append(tokens, PIVTokenInfo{
			GUID:      guid,
			SlotId:    0x9c,
			PublicKey: cert.PublicKey,
			Serial:    serial,
			Card:      card,
		})

		yk.Close()
	}

	return tokens, nil
}
