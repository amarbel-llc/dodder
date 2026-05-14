//go:build test

package remote_http

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl_io"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"

	// Side-effect: register format-purpose pairs (matches production binaries
	// which pull this in via commands_dodder/main.go's blank import).
	_ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"
)

// These tests exercise the full sign-on-server / verify-on-client round
// trip through RoundTripperBufioWrappedSigner + sigMiddleware. The
// verifyingBodyReader cases in round_tripper_wrapped_signer_test.go cover
// the body-reader path in isolation; this file covers the integrated
// transport.

// sigRoundTripFixture stands up a real Repo (via local_working_copy.MakeTesting)
// with a real keypair and a httptest.NewServer wrapping the supplied handler in
// server.sigMiddleware. Tests dial the httptest server over plain TCP and
// drive the request through RoundTripperBufioWrappedSigner — same wire
// framing as production.
type sigRoundTripFixture struct {
	repo       *local_working_copy.Repo
	server     *Server
	hashFormat mad_domain_interfaces.FormatHash
	htServer   *httptest.Server
	host       string
}

// makeTrailerSigningHandler mimics the production response-handling pattern
// (server.go:482-524): announce the trailer, write the body through a
// digesting writer, then call signBodyTrailer with the final digest.
func makeTrailerSigningHandler(
	t *ui.TestContext,
	server *Server,
	hashFormat mad_domain_interfaces.FormatHash,
	body []byte,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Trailer", headerRepoSig)
			w.WriteHeader(http.StatusOK)

			hash, repoolHash := hashFormat.GetHash()
			defer repoolHash()

			digestWriter := markl_io.MakeWriter(hash, w)
			_, err := digestWriter.Write(body)
			t.AssertNoError(err)

			server.signBodyTrailer(digestWriter.GetMarklId(), w)
		},
	)
}

// makeNoTrailerHandler writes the body without ever signing the trailer.
// Used to drive the client's missing-trailer path.
func makeNoTrailerHandler(
	t *ui.TestContext,
	body []byte,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Trailer", headerRepoSig)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write(body)
			t.AssertNoError(err)
		},
	)
}

func makeFixture(
	t *ui.TestContext,
	makeHandler func(server *Server, hashFormat mad_domain_interfaces.FormatHash) http.Handler,
) *sigRoundTripFixture {
	repo := local_working_copy.MakeTesting(t, nil)

	server := &Server{
		EnvLocal: repo,
		Repo:     repo,
	}

	hashFormat := repo.GetBlobStore().GetDefaultHashType()

	handler := server.sigMiddleware(makeHandler(server, hashFormat))
	htServer := httptest.NewServer(handler)
	t.Cleanup(htServer.Close)

	parsed, err := url.Parse(htServer.URL)
	t.AssertNoError(err)

	return &sigRoundTripFixture{
		repo:       repo,
		server:     server,
		hashFormat: hashFormat,
		htServer:   htServer,
		host:       parsed.Host,
	}
}

// dialRoundTripper opens a fresh TCP connection to the fixture's
// server and returns a RoundTripperBufioWrappedSigner configured with
// the given pubkey. Connection is closed at test cleanup.
func dialRoundTripper(
	t *ui.TestContext,
	fixture *sigRoundTripFixture,
	pubkey mad_domain_interfaces.MarklId,
) *RoundTripperBufioWrappedSigner {
	conn, err := net.Dial("tcp", fixture.host)
	t.AssertNoError(err)
	t.Cleanup(func() { _ = conn.Close() })

	rt := &RoundTripperBufioWrappedSigner{
		PublicKey:  pubkey,
		HashFormat: fixture.hashFormat,
	}
	rt.roundTripperBufio.Writer = bufio.NewWriter(conn)
	rt.roundTripperBufio.Reader = bufio.NewReader(conn)

	return rt
}

func TestSigRoundTripPositive(t1 *testing.T) {
	ui.RunTestContext(t1, testSigRoundTripPositive)
}

func testSigRoundTripPositive(t *ui.TestContext) {
	body := []byte("signed response body")

	fixture := makeFixture(
		t,
		func(server *Server, hashFormat mad_domain_interfaces.FormatHash) http.Handler {
			return makeTrailerSigningHandler(t, server, hashFormat, body)
		},
	)

	pubkey := fixture.repo.GetImmutableConfigPublic().GetPublicKey()

	rt := dialRoundTripper(t, fixture, pubkey)

	req, err := http.NewRequest(http.MethodGet, fixture.htServer.URL+"/", nil)
	t.AssertNoError(err)

	resp, err := rt.RoundTrip(req)
	t.AssertNoError(err)

	got, err := io.ReadAll(resp.Body)
	t.AssertNoError(err)
	_ = resp.Body.Close()

	t.AssertEqual(body, got)
}

func TestSigRoundTripPubkeyMismatch(t1 *testing.T) {
	ui.RunTestContext(t1, testSigRoundTripPubkeyMismatch)
}

func testSigRoundTripPubkeyMismatch(t *ui.TestContext) {
	body := []byte("body the server will sign")

	fixture := makeFixture(
		t,
		func(server *Server, hashFormat mad_domain_interfaces.FormatHash) http.Handler {
			return makeTrailerSigningHandler(t, server, hashFormat, body)
		},
	)

	// Generate a different keypair; tell the round tripper to expect THAT
	// pubkey instead of the server's actual pubkey. Server still returns
	// its real pubkey in the response header — client must reject the
	// mismatch before verifying the challenge sig.
	otherKeys := makeSignerTestKeys(t.T)

	rt := dialRoundTripper(t, fixture, otherKeys.pub)

	req, err := http.NewRequest(http.MethodGet, fixture.htServer.URL+"/", nil)
	t.AssertNoError(err)

	_, err = rt.RoundTrip(req)
	t.AssertErrorContains("expected pubkey", err)
}

func TestSigRoundTripMissingNonceRejected(t1 *testing.T) {
	ui.RunTestContext(t1, testSigRoundTripMissingNonceRejected)
}

func testSigRoundTripMissingNonceRejected(t *ui.TestContext) {
	body := []byte("body the server will sign")

	fixture := makeFixture(
		t,
		func(server *Server, hashFormat mad_domain_interfaces.FormatHash) http.Handler {
			return makeTrailerSigningHandler(t, server, hashFormat, body)
		},
	)

	// Send a bare request — no nonce header. Use stdlib http.Client to
	// bypass the wrapped signer's nonce injection entirely.
	resp, err := http.Get(fixture.htServer.URL + "/")
	t.AssertNoError(err)
	defer resp.Body.Close() //defer:err-checked

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSigRoundTripMissingTrailerFails(t1 *testing.T) {
	ui.RunTestContext(t1, testSigRoundTripMissingTrailerFails)
}

func testSigRoundTripMissingTrailerFails(t *ui.TestContext) {
	body := []byte("body without trailer")

	fixture := makeFixture(
		t,
		func(server *Server, hashFormat mad_domain_interfaces.FormatHash) http.Handler {
			return makeNoTrailerHandler(t, body)
		},
	)

	pubkey := fixture.repo.GetImmutableConfigPublic().GetPublicKey()

	rt := dialRoundTripper(t, fixture, pubkey)

	req, err := http.NewRequest(http.MethodGet, fixture.htServer.URL+"/", nil)
	t.AssertNoError(err)

	resp, err := rt.RoundTrip(req)
	t.AssertNoError(err)

	_, err = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	t.AssertErrorContains("trailer", err)
}
