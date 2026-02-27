package markl

import (
	"crypto"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"

	"github.com/go-piv/piv-go/v2/piv"
)

type gopivSigner struct {
	yubikey *piv.YubiKey
	slot    piv.Slot
	signer  crypto.Signer
	guid    [16]byte
	slotId  byte
}

func OpenPIVSigner(guid [16]byte, slotId byte) (PIVSigner, error) {
	cards, err := piv.Cards()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list smart card readers")
	}

	slot, err := pivSlotFromByte(slotId)
	if err != nil {
		return nil, err
	}

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

		// go-piv exposes serial number, not GUID directly.
		// Encode serial as first 4 bytes of GUID for matching.
		var cardGUID [16]byte
		cardGUID[0] = byte(serial >> 24)
		cardGUID[1] = byte(serial >> 16)
		cardGUID[2] = byte(serial >> 8)
		cardGUID[3] = byte(serial)

		if cardGUID != guid {
			yk.Close()
			continue
		}

		cert, err := yk.Certificate(slot)
		if err != nil {
			yk.Close()
			return nil, errors.Wrapf(err, "failed to read certificate from slot %x", slotId)
		}

		auth := piv.KeyAuth{
			PINPrompt: pivPINPrompt,
		}

		priv, err := yk.PrivateKey(slot, cert.PublicKey, auth)
		if err != nil {
			yk.Close()
			return nil, errors.Wrapf(err, "failed to get private key from slot %x", slotId)
		}

		signer, ok := priv.(crypto.Signer)
		if !ok {
			yk.Close()
			return nil, errors.Errorf("private key in slot %x does not implement crypto.Signer", slotId)
		}

		return &gopivSigner{
			yubikey: yk,
			slot:    slot,
			signer:  signer,
			guid:    guid,
			slotId:  slotId,
		}, nil
	}

	return nil, errors.Errorf(
		"PIV token with GUID %x not found",
		guid,
	)
}

func (s *gopivSigner) Public() crypto.PublicKey {
	return s.signer.Public()
}

func (s *gopivSigner) Sign(
	rand io.Reader,
	digest []byte,
	opts crypto.SignerOpts,
) ([]byte, error) {
	return s.signer.Sign(rand, digest, opts)
}

func (s *gopivSigner) GUID() [16]byte { return s.guid }
func (s *gopivSigner) SlotId() byte   { return s.slotId }

func (s *gopivSigner) Close() error {
	if s.yubikey != nil {
		return s.yubikey.Close()
	}
	return nil
}

func pivSlotFromByte(b byte) (piv.Slot, error) {
	switch b {
	case 0x9a:
		return piv.SlotAuthentication, nil
	case 0x9c:
		return piv.SlotSignature, nil
	case 0x9d:
		return piv.SlotKeyManagement, nil
	case 0x9e:
		return piv.SlotCardAuthentication, nil
	default:
		return piv.Slot{}, errors.Errorf("unsupported PIV slot: %x", b)
	}
}

func pivPINPrompt() (string, error) {
	pin := os.Getenv("DODDER_PIV_PIN")
	if pin != "" {
		return pin, nil
	}

	return "", errors.Errorf(
		"PIV PIN required: set DODDER_PIV_PIN or provide interactive PIN prompt",
	)
}
