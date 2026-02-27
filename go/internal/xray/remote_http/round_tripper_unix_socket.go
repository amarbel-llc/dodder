package remote_http

import (
	"bufio"
	"net"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/uniform/repo"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type RoundTripperUnixSocket struct {
	repo.UnixSocket
	net.Conn
	RoundTripperBufioWrappedSigner
}

// TODO add public key
func (roundTripper *RoundTripperUnixSocket) Initialize(
	remote *Server,
	pubkey markl.Id,
) (err error) {
	roundTripper.PublicKey = pubkey

	if roundTripper.UnixSocket, err = remote.InitializeUnixSocket(
		net.ListenConfig{},
		"",
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if roundTripper.Conn, err = net.Dial("unix", roundTripper.Path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	roundTripper.Writer = bufio.NewWriter(roundTripper.Conn)
	roundTripper.Reader = bufio.NewReader(roundTripper.Conn)

	return err
}
