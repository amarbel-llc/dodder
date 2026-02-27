package markl

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

var ErrEd25519SSHAgentNotConnected, IsErrEd25519SSHAgentNotConnected = errors.MakeTypedSentinel[pkgErrDisamb](
	"ed25519 SSH agent signer not connected",
)
