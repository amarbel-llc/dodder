package remote_proto

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/coder/websocket"
)

// The websocket transport is the motivating addition of RFC 0004: it carries
// the drtp session over a connection upgraded from an ordinary HTTP request,
// so the protocol can traverse HTTP proxies and share a port with a web
// frontend. websocket.NetConn adapts the upgraded connection to a net.Conn,
// so the same session code runs unchanged over websockets, stdio, raw TCP,
// and the in-memory net.Pipe used by the tests.

// Serve accepts HTTP connections on listener and upgrades requests to
// PathTransfer into drtp sessions. PathHealthz is served without an upgrade
// for liveness probes, mirroring remote_http.
func (server *Server) Serve(listener net.Listener) (err error) {
	mux := http.NewServeMux()

	mux.HandleFunc(PathHealthz, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	})

	mux.HandleFunc(PathTransfer, server.handleTransfer)

	httpServer := &http.Server{Handler: mux}

	go func() {
		<-server.Repo.GetEnv().Done()
		_ = httpServer.Close()
	}()

	if err = httpServer.Serve(listener); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (server *Server) handleTransfer(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	conn, err := websocket.Accept(
		responseWriter,
		request,
		// CLI clients do not send a browser Origin; skip the cross-origin
		// check. Authentication is the per-session markl challenge, not the
		// Origin header.
		&websocket.AcceptOptions{InsecureSkipVerify: true},
	)
	if err != nil {
		ui.Err().Printf("websocket upgrade failed: %s", err)
		return
	}

	// Disable the default 32 KiB message read limit: a flushed objects or
	// blob frame may be a single large message.
	conn.SetReadLimit(-1)

	netConn := websocket.NetConn(
		request.Context(),
		conn,
		websocket.MessageBinary,
	)

	if serveErr := server.ServeConn(netConn); serveErr != nil {
		ui.Err().Printf("drtp session failed: %s", serveErr)
		_ = conn.Close(websocket.StatusInternalError, "transfer failed")
		return
	}

	_ = conn.Close(websocket.StatusNormalClosure, "")
}

// ServeStdio runs a single drtp session over the process's stdin/stdout,
// for the `dodder serve-proto -` (local and SSH) transports.
func (server *Server) ServeStdio() (err error) {
	if err = server.ServeConn(stdioReadWriteCloser{}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// DialWebSocket upgrades an HTTP connection to urlString into a drtp byte
// stream. ctx bounds the connection's lifetime and MUST outlive the
// transfer.
func DialWebSocket(
	ctx context.Context,
	urlString string,
) (conn io.ReadWriteCloser, err error) {
	wsConn, _, dialErr := websocket.Dial(ctx, urlString, nil)
	if dialErr != nil {
		err = errors.Wrapf(dialErr, "dialing websocket %q", urlString)
		return conn, err
	}

	wsConn.SetReadLimit(-1)

	conn = websocket.NetConn(ctx, wsConn, websocket.MessageBinary)

	return conn, err
}

// WebSocketURL converts an http(s) base URL into the ws(s) PathTransfer URL
// a client dials. A bare host (no scheme) defaults to ws://.
func WebSocketURL(base string) string {
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	case strings.HasPrefix(base, "wss://"), strings.HasPrefix(base, "ws://"):
		// already a websocket scheme
	default:
		base = "ws://" + base
	}

	base = strings.TrimRight(base, "/")

	return base + PathTransfer
}

// stdioReadWriteCloser presents the process's stdin/stdout as one byte
// stream for the stdio transport. Close is a no-op: the parent process owns
// the descriptors.
type stdioReadWriteCloser struct{}

func (stdioReadWriteCloser) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdioReadWriteCloser) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (stdioReadWriteCloser) Close() error                { return nil }
