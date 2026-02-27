package markl

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

var ErrEd25519SSHAgentNotConnected, IsErrEd25519SSHAgentNotConnected = errors.MakeTypedSentinel[pkgErrDisamb](
	"ed25519 SSH agent signer not connected",
)
