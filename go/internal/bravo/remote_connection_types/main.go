package remote_connection_types

import (
	"strings"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

//go:generate stringer -type=Type
type Type int

const (
	TypeUnspecified = Type(iota)
	TypeNative
	TypeNativeLocalOverridePath
	TypeSocketUnix
	TypeUrl
	TypeStdioLocal
	TypeStdioSSH
	_TypeMax
)

func GetAllTypeTypes() []Type {
	types := make([]Type, 0)

	for i := TypeUnspecified + 1; i < _TypeMax; i++ {
		types = append(types, Type(i))
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

func (tipe *Type) GetCLICompletion() map[string]string {
	return map[string]string{
		"none":   "",
		"native": "",
		// TODO rename
		"native-dotenv-xdg": "",
		"socket-unix":       "",
		"stdio-local":       "",
		"stdio-ssh":         "",
		"unspecified":       "",
		"url":               "",
	}
}

func (tipe *Type) Set(value string) (err error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "none", "unspecified":
		*tipe = TypeUnspecified

	case "native":
		*tipe = TypeNative

		// TODO rename
	case "native-dotenv-xdg":
		*tipe = TypeNativeLocalOverridePath

	case "socket-unix":
		*tipe = TypeSocketUnix

	case "url":
		*tipe = TypeUrl

	case "stdio-local":
		*tipe = TypeStdioLocal

	case "stdio-ssh":
		*tipe = TypeStdioSSH

	default:
		err = errors.ErrorWithStackf("unsupported remote type: %q", value)
		return err
	}

	return err
}
