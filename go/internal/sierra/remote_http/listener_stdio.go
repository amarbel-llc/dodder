package remote_http

import (
	"io"
	"net"
	"os"
	"time"
)

type StdioListener struct {
	accept, done chan struct{}
}

func MakeStdioListener() net.Listener {
	listener := &StdioListener{
		accept: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}

	// prepopulate the accept channel with the first request
	listener.accept <- struct{}{}

	return listener
}

func (listener *StdioListener) Accept() (conn net.Conn, err error) {
	// if we're done, io.EOF, otherwise wait for the previous request to finish
	// before we permit this new request
	//
	// TODO, somehow ensure that concurrent requests are never made by the client
	select {
	case <-listener.done:
		err = io.EOF

	case <-listener.accept:
		conn = &StdioConn{acceptNext: listener.accept}
	}

	return conn, err
}

func (listener *StdioListener) Close() error {
	close(listener.done)
	return nil
}

func (listener *StdioListener) Addr() net.Addr { return &net.TCPAddr{} }

type StdioConn struct {
	// writes a single value when Close is called to enable StdioListener to
	// "accept" a new connection
	acceptNext chan<- struct{}
}

func (conn *StdioConn) Read(b []byte) (n int, err error) {
	return os.Stdin.Read(b)
}

func (conn *StdioConn) Write(b []byte) (n int, err error) {
	return os.Stdout.Write(b)
}

func (conn *StdioConn) Close() error {
	conn.acceptNext <- struct{}{}
	return nil
}
func (conn *StdioConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (conn *StdioConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (conn *StdioConn) SetDeadline(t time.Time) error      { return nil }
func (conn *StdioConn) SetReadDeadline(t time.Time) error  { return nil }
func (conn *StdioConn) SetWriteDeadline(t time.Time) error { return nil }
