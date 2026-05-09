package remote_http

import (
	"bufio"
	"net"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/values"
)

// RoundTripperBufioHost dials a remote host:port over TCP and runs
// requests through RoundTripperBufioWrappedSigner. Mirrors the
// RoundTripperUnixSocket / RoundTripperStdio pattern (both of which
// already embed RoundTripperBufioWrappedSigner) for the URL/HTTP
// path that previously went through RoundTripperHost +
// DefaultRoundTripper — bypassing sig-auth entirely (#170).
//
// Like the other bufio-flavored transports, this holds a single
// open net.Conn for the lifetime of the client and runs every
// request sequentially on the same wire. No connection pooling.
type RoundTripperBufioHost struct {
	net.Conn
	RoundTripperBufioWrappedSigner
}

// Initialize dials the URI's host:port and wires up the bufio
// reader/writer pair the embedded RoundTripperBufioWrappedSigner
// needs. PublicKey is left null — the wrapped signer's first-contact
// path TOFU-accepts the server's advertised public key on first
// connect; pinning is a follow-up.
func (roundTripper *RoundTripperBufioHost) Initialize(
	uri values.Uri,
	hashFormat mad_domain_interfaces.FormatHash,
) (err error) {
	roundTripper.HashFormat = hashFormat
	// Default PublicKey to a zero-value markl.Id so the embedded
	// signer's IsNull() check (TOFU branch) doesn't dereference a
	// nil interface. Pinning is a follow-up.
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
