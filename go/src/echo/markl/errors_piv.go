package markl

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

var ErrEd25519PIVSignerConnectionNotEstablished, IsErrEd25519PIVSignerConnectionNotEstablished = errors.MakeTypedSentinel[pkgErrDisamb](
	"ed25519 PIV signer connection not established",
)
