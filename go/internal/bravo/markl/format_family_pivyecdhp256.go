package markl

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/pivy"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func PivyEcdhP256GetIOWrapper(
	id domain_interfaces.MarklId,
) (ioWrapper interfaces.IOWrapper, err error) {
	compressed := id.GetBytes()

	pubkey, err := pivy.DecompressP256Point(compressed)
	if err != nil {
		err = errors.Wrapf(err, "parsing P-256 public key")
		return ioWrapper, err
	}

	socketPath, err := pivy.ResolveAgentSocketPath()
	if err != nil {
		err = errors.Wrap(err)
		return ioWrapper, err
	}

	ioWrapper = &pivy.IOWrapper{
		RecipientPubkey: pubkey,
		DecryptECDH:     pivy.AgentECDHFunc(socketPath, pubkey),
	}

	return ioWrapper, err
}
