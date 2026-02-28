package repo

import "net"

type UnixSocket struct {
	net.Listener
	Path string
}
