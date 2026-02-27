package repo

import (
	"strings"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

//go:generate stringer -type=RemoteConnection
type RemoteConnection int

const (
	RemoteConnectionUnspecified = RemoteConnection(iota)
	RemoteConnectionNative
	RemoteConnectionNativeLocalOverridePath
	RemoteConnectionSocketUnix
	RemoteConnectionUrl
	RemoteConnectionStdioLocal
	RemoteConnectionStdioSSH
	_RemoteConnectionMax
)

func GetAllRemoteConnectionTypes() []RemoteConnection {
	types := make([]RemoteConnection, 0)

	for i := RemoteConnectionUnspecified + 1; i < _RemoteConnectionMax; i++ {
		types = append(types, RemoteConnection(i))
	}

	return types
}

/*
problem: there are two components of connecting to a remote:

- protocol used (native, socket, stdio, etc)
- how the remote is available and defined in the repo (the remote config type)

the second determines the first

in the CLI, the remote type shoudl really refer to the remote config blob type, and then the connection type will be determined by that

*/

func (remoteConnection *RemoteConnection) GetCLICompletion() map[string]string {
	return map[string]string{
		"native": "",
		// TODO rename
		"native-dotenv-xdg": "",
		"none":              "",
		"socket-unix":       "",
		"stdio-local":       "",
		"stdio-ssh":         "",
		"unspecified":       "",
		"url":               "",
	}
}

func (remoteConnection *RemoteConnection) Set(value string) (err error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "none", "unspecified":
		*remoteConnection = RemoteConnectionUnspecified

	case "native":
		*remoteConnection = RemoteConnectionNative

		// TODO rename
	case "native-dotenv-xdg":
		*remoteConnection = RemoteConnectionNativeLocalOverridePath

	case "socket-unix":
		*remoteConnection = RemoteConnectionSocketUnix

	case "url":
		*remoteConnection = RemoteConnectionUrl

	case "stdio-local":
		*remoteConnection = RemoteConnectionStdioLocal

	case "stdio-ssh":
		*remoteConnection = RemoteConnectionStdioSSH

	default:
		err = errors.ErrorWithStackf("unsupported remote type: %q", value)
		return err
	}

	return err
}
