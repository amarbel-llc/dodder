package remote_http

import (
	"bufio"
	"net"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/values"
)

// RoundTripperBufioHost dials a remote host:port over TCP and runs
// requests through the embedded signer. Holds a single net.Conn for
// its lifetime and runs requests sequentially — no connection pooling.
type RoundTripperBufioHost struct {
	net.Conn
	RoundTripperBufioWrappedSigner
}

func (roundTripper *RoundTripperBufioHost) Initialize(
	uri values.Uri,
	hashFormat mad_domain_interfaces.FormatHash,
) (err error) {
	roundTripper.HashFormat = hashFormat
	// Zero-value markl.Id (vs nil interface) so the signer's IsNull()
	// TOFU check doesn't deref a nil interface.
	roundTripper.PublicKey = &markl.Id{}

	host := uri.GetUrl().Host

	if roundTripper.Conn, err = net.Dial("tcp", host); err != nil {
		err = errors.Wrap(err)
		return err
	}

	roundTripper.Writer = bufio.NewWriter(roundTripper.Conn)
	roundTripper.Reader = bufio.NewReader(roundTripper.Conn)

	return err
}
