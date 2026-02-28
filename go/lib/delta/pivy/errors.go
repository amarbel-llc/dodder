package pivy

import "code.linenisgreat.com/dodder/go/lib/bravo/errors"

type errAgentDisamb struct{}

// ErrAgent is returned when the pivy-agent communication fails (socket error,
// extension failure, card not found, PIN needed). Callers can use IsErrAgent
// to distinguish agent failures from AEAD trial-decryption failures.
var ErrAgent, IsErrAgent = errors.MakeTypedSentinel[errAgentDisamb]("pivy agent error")
