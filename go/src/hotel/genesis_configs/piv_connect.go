package genesis_configs

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

var (
	pivConnectOnce sync.Once
	pivConnectErr  error
)

func connectPIVSignerIfNecessary(privateKey markl.Id) error {
	format := privateKey.GetMarklFormat()
	if format == nil {
		return nil
	}

	if format.GetMarklFormatId() != markl.FormatIdEd25519PIV {
		return nil
	}

	pivConnectOnce.Do(func() {
		guid, slotId, err := markl.DecodePIVReference(privateKey.GetBytes())
		if err != nil {
			pivConnectErr = errors.Wrap(err)
			return
		}

		signer, err := markl.OpenPIVSigner(guid, slotId)
		if err != nil {
			pivConnectErr = errors.Wrap(err)
			return
		}

		markl.RegisterPIVEd25519Format(signer)
	})

	return pivConnectErr
}
