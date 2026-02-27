package markl

import (
	"crypto/ed25519"
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/bech32"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/age"
)

func AgeX25519Generate(_ io.Reader) (bites []byte, err error) {
	var ageId age.Identity

	// TODO add support for injecting rand reader
	if err = ageId.GenerateIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return bites, err
	}

	bech32String := ageId.String()

	if _, bites, err = bech32.Decode(bech32String); err != nil {
		err = errors.Wrap(err)
		return bites, err
	}

	return bites, err
}

// TODO verify if this is correct
func AgeX25519GetPublicKey(
	private domain_interfaces.MarklId,
) (bites []byte, err error) {
	// the ed25519 package includes a public key suffix, so we need to
	// reconstruct their version of a private key for a public key value
	privateKey := ed25519.PrivateKey(private.GetBytes())
	bites = privateKey.Public().(ed25519.PublicKey)

	return bites, err
}

func AgeX25519GetIOWrapper(
	private domain_interfaces.MarklId,
) (ioWrapper interfaces.IOWrapper, err error) {
	var ageId age.Identity

	var bech32String string

	if bech32String, err = bech32.Encode(
		"AGE-SECRET-KEY-",
		private.GetBytes(),
	); err != nil {
		err = errors.Wrap(err)
		return ioWrapper, err
	}

	if err = ageId.Set(bech32String); err != nil {
		err = errors.Wrap(err)
		return ioWrapper, err
	}

	ioWrapper = &ageId

	return ioWrapper, err
}
