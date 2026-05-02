package markl

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

var ErrEd25519SSHAgentNotConnected, IsErrEd25519SSHAgentNotConnected = errors.MakeTypedSentinel[pkgErrDisamb](
	"ed25519 SSH agent signer not connected",
)
