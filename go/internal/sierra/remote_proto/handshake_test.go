package remote_proto

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
	"github.com/coder/websocket"
)

const testListType = "inventory_list-v2"

func makeTestKeys(t *testing.T) keys {
	t.Helper()

	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	edPub := edPriv.Public().(ed25519.PublicKey)

	priv := &markl.Id{}
	if err := priv.SetPurposeId(markl.PurposeRepoPrivateKeyV1); err != nil {
		t.Fatalf("priv.SetPurposeId: %v", err)
	}
	if err := priv.SetMarklId(markl.FormatIdEd25519Sec, []byte(edPriv)); err != nil {
		t.Fatalf("priv.SetMarklId: %v", err)
	}

	pub := &markl.Id{}
	if err := pub.SetPurposeId(markl.PurposeRepoPubKeyV1); err != nil {
		t.Fatalf("pub.SetPurposeId: %v", err)
	}
	if err := pub.SetMarklId(markl.FormatIdEd25519Pub, []byte(edPub)); err != nil {
		t.Fatalf("pub.SetMarklId: %v", err)
	}

	return keys{private: priv, public: pub}
}

// TestHandshakeMutualAttestation runs a full client/server handshake over an
// in-memory pipe with generated keys and asserts mutual attestation succeeds
// and the want is delivered intact.
func TestHandshakeMutualAttestation(t1 *testing.T) {
	t := ui.MakeT(t1)

	clientConn, serverConn := net.Pipe()

	serverKeys := makeTestKeys(t.T)
	clientKeys := makeTestKeys(t.T)

	type serverResult struct {
		want control
		err  error
	}

	results := make(chan serverResult, 1)

	go func() {
		s := makeSession(serverConn)
		_, want, err := serverHandshake(s, serverKeys, selfCaps{listType: testListType}, false)
		results <- serverResult{want: want, err: err}
	}()

	s := makeSession(clientConn)

	serverCaps, _, err := clientHandshake(s, clientKeys, selfCaps{listType: testListType}, serverKeys.public)
	t.AssertNoError(err)
	t.AssertEqual(testListType, serverCaps.InventoryListType)
	t.AssertEqual(RoleServer, serverCaps.Role)

	want, err := signWant(clientKeys, serverCaps.Nonce, control{
		Direction: DirectionFetch,
		Query:     ":z",
	})
	t.AssertNoError(err)
	t.AssertNoError(s.writeControl(TypeWant, want))

	got := <-results
	t.AssertNoError(got.err)
	t.AssertEqual(DirectionFetch, got.want.Direction)
	t.AssertEqual(":z", got.want.Query)
}

// TestHandshakePinnedKeyMismatch asserts a client that pins a different
// server key rejects the connection.
func TestHandshakePinnedKeyMismatch(t1 *testing.T) {
	t := ui.MakeT(t1)

	clientConn, serverConn := net.Pipe()

	serverKeys := makeTestKeys(t.T)
	clientKeys := makeTestKeys(t.T)
	otherKeys := makeTestKeys(t.T)

	go func() {
		s := makeSession(serverConn)
		_, _, _ = serverHandshake(s, serverKeys, selfCaps{listType: testListType}, false)
		_ = serverConn.Close()
	}()

	s := makeSession(clientConn)
	defer func() { _ = clientConn.Close() }()

	_, _, err := clientHandshake(s, clientKeys, selfCaps{listType: testListType}, otherKeys.public)
	t.AssertError(err)
	if !strings.Contains(err.Error(), "public key mismatch") {
		t.Fatalf("expected public key mismatch, got: %v", err)
	}
}

// TestHandshakeIdentityIsPubkeyNotRepoId asserts peer identity is the public
// key, not the legacy repo id: two peers advertising the SAME repoId
// ("default") but distinct keys still complete mutual attestation. This guards
// the FDR-0021 deprecation — the config-seed id stops being written, so the
// handshake's repoId field rides empty or colliding and must never gate
// identity (two hosts both named "default" are distinguished by pubkey).
func TestHandshakeIdentityIsPubkeyNotRepoId(t1 *testing.T) {
	t := ui.MakeT(t1)

	clientConn, serverConn := net.Pipe()

	serverKeys := makeTestKeys(t.T)
	clientKeys := makeTestKeys(t.T)

	results := make(chan error, 1)

	go func() {
		s := makeSession(serverConn)
		_, _, err := serverHandshake(
			s,
			serverKeys,
			selfCaps{listType: testListType, repoId: "default"},
			false,
		)
		results <- err
	}()

	s := makeSession(clientConn)

	serverCaps, _, err := clientHandshake(
		s,
		clientKeys,
		selfCaps{listType: testListType, repoId: "default"},
		serverKeys.public,
	)
	t.AssertNoError(err)
	// The server advertised the colliding repoId, yet attestation succeeded:
	// identity came from the pubkey, and repoId is carried-but-inert metadata.
	t.AssertEqual("default", serverCaps.RepoId)

	want, err := signWant(clientKeys, serverCaps.Nonce, control{
		Direction: DirectionFetch,
		Query:     ":z",
	})
	t.AssertNoError(err)
	t.AssertNoError(s.writeControl(TypeWant, want))

	t.AssertNoError(<-results)
}

// TestHandshakeOverWebSocket exercises the websocket transport end to end:
// the handshake runs over a real upgraded connection adapted via
// websocket.NetConn, verifying framing survives message boundaries.
func TestHandshakeOverWebSocket(t1 *testing.T) {
	t := ui.MakeT(t1)

	serverKeys := makeTestKeys(t.T)
	clientKeys := makeTestKeys(t.T)

	wantReceived := make(chan control, 1)
	serverErr := make(chan error, 1)

	handler := func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			serverErr <- err
			return
		}
		conn.SetReadLimit(-1)
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		netConn := websocket.NetConn(r.Context(), conn, websocket.MessageBinary)
		s := makeSession(netConn)

		_, want, hErr := serverHandshake(s, serverKeys, selfCaps{listType: testListType}, true)
		if hErr != nil {
			serverErr <- hErr
			return
		}

		wantReceived <- want
		serverErr <- nil
	}

	httpServer := httptest.NewServer(http.HandlerFunc(handler))
	defer httpServer.Close()

	wsURL := WebSocketURL(httpServer.URL)
	if !strings.HasPrefix(wsURL, "ws://") {
		t.Fatalf("expected ws:// URL, got %q", wsURL)
	}

	ctx := context.Background()

	conn, err := DialWebSocket(ctx, wsURL)
	t.AssertNoError(err)

	s := makeSession(conn)

	serverCaps, _, err := clientHandshake(s, clientKeys, selfCaps{listType: testListType}, serverKeys.public)
	t.AssertNoError(err)
	t.AssertEqual(testListType, serverCaps.InventoryListType)

	want, err := signWant(clientKeys, serverCaps.Nonce, control{
		Direction: DirectionFetch,
		Query:     ":t",
	})
	t.AssertNoError(err)
	t.AssertNoError(s.writeControl(TypeWant, want))

	t.AssertNoError(<-serverErr)

	got := <-wantReceived
	t.AssertEqual(DirectionFetch, got.Direction)
	t.AssertEqual(":t", got.Query)

	_ = conn.Close()
}
