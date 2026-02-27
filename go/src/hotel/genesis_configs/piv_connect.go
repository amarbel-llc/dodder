package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func connectPIVSignerIfNecessary(privateKey markl.Id) (err error) {
	format := privateKey.GetMarklFormat()
	if format == nil {
		return err
	}

	if format.GetMarklFormatId() != markl.FormatIdEd25519PIV {
		return err
	}

	guid, slotId, err := markl.DecodePIVReference(privateKey.GetBytes())
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	signer, err := markl.OpenPIVSigner(guid, slotId)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	markl.RegisterPIVEd25519Format(signer)

	return err
}
