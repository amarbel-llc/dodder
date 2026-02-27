package markl

import (
	"crypto/ed25519"
	"crypto/rand"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

func Ed25519GeneratePrivateKey(rand io.Reader) (bites []byte, err error) {
	if _, bites, err = ed25519.GenerateKey(rand); err != nil {
		err = errors.Wrap(err)
		return bites, err
	}

	return bites, err
}

func Ed25519GetPublicKey(private domain_interfaces.MarklId) (bites []byte, err error) {
	privateBytes := private.GetBytes()
	var privateKey ed25519.PrivateKey

	switch len(privateBytes) {
	case ed25519.SeedSize:
		// TODO emit error
		// err = errors.Errorf(
		// 	"private key is just seed, not full go ed25519 private key",
		// )
		// return
		privateKey = ed25519.NewKeyFromSeed(privateBytes)

	case ed25519.PrivateKeySize:
		privateKey = ed25519.PrivateKey(privateBytes)

	default:
		err = errors.Errorf(
			"unsupported key size: %d",
			len(privateBytes),
		)
		return bites, err
	}

	pubKey := privateKey.Public()
	bites = pubKey.(ed25519.PublicKey)

	return bites, err
}

func Ed25519Verify(pub, message, sig domain_interfaces.MarklId) (err error) {
	pubBites := ed25519.PublicKey(pub.GetBytes())

	if err = ed25519.VerifyWithOptions(
		pubBites,
		message.GetBytes(),
		sig.GetBytes(),
		&ed25519.Options{},
	); err != nil {
		err = errors.Err422UnprocessableEntity.Errorf(
			"invalid signature: %w. Signature: %q",
			err,
			sig.StringWithFormat(),
		)
		return err
	}

	return err
}

func Ed25519Sign(
	sec domain_interfaces.MarklId,
	mes domain_interfaces.MarklId,
	readerRand io.Reader,
) (sigBites []byte, err error) {
	if readerRand == nil {
		readerRand = rand.Reader
	}

	privateKey := ed25519.PrivateKey(sec.GetBytes())

	if sigBites, err = privateKey.Sign(
		readerRand,
		mes.GetBytes(),
		&ed25519.Options{},
	); err != nil {
		err = errors.Wrap(err)
		return sigBites, err
	}

	return sigBites, err
}
