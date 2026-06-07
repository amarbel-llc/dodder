package commands_dodder

import (
	"fmt"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/command_components"
	"code.linenisgreat.com/dodder/go/internal/sierra/remote_proto"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// ProtoHandshakeProtocolName is advertised in field 5 of the handshake
// line so harnesses can distinguish a drtp server from the legacy
// remote_http one. The line shape is identical to `serve -handshake`, so
// the same port-discovery harness reads either.
const ProtoHandshakeProtocolName = "dodder-drtp-v1"

// ProtoHandshakeProtocolVersion is field 1 of the handshake line.
const ProtoHandshakeProtocolVersion = 1

func init() {
	utility.AddCmd("serve-proto", &ServeProto{})
}

// ServeProto serves the drtp remote transfer protocol
// (sierra/remote_proto, RFC 0004): a session protocol that streams a
// sender-computed expand-edges closure, optionally over a websocket
// upgraded from HTTP. It coexists with the legacy `serve` (remote_http)
// command.
type ServeProto struct {
	command_components.Env
	command_components_dodder.EnvRepo
	command_components_dodder.LocalWorkingCopy

	// Public serves fetches without requiring client attestation, mirroring
	// `serve -public`. A push always requires attestation.
	Public bool

	// Handshake forces tcp + ephemeral port and prints a single
	// pipe-delimited handshake line on stdout after binding, mirroring
	// `serve -handshake`, so harnesses can discover the OS-assigned port.
	Handshake bool
}

var _ interfaces.CommandComponentWriter = (*ServeProto)(nil)

func (cmd ServeProto) GetDescription() command.Description {
	return command.Description{
		Short: "serve the drtp remote transfer protocol (websocket-capable)",
	}
}

func (cmd *ServeProto) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "network",
				Description: "network type (tcp, unix, or - for stdio)",
			},
			{
				Name:        "address",
				Description: "listen address (e.g. :8080)",
			},
		},
	}}
}

func (cmd *ServeProto) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

	flagSet.BoolVar(
		&cmd.Public,
		"public",
		false,
		"serve fetches without requiring client attestation",
	)

	flagSet.BoolVar(
		&cmd.Handshake,
		"handshake",
		false,
		"emit a handshake line on stdout after binding (forces tcp + ephemeral port)",
	)
}

func (cmd ServeProto) Run(req command.Request) {
	args := req.PopArgs()
	errors.ContextSetCancelOnSIGHUP(req)

	envLocal := cmd.MakeEnvWithOptions(
		req,
		env_ui.Options{
			UIFileIsStderr: true,
			IgnoreTtyState: true,
		},
	)

	repo := cmd.MakeLocalWorkingCopyFromEnvLocal(envLocal)

	server := remote_proto.Server{
		EnvLocal: envLocal,
		Repo:     repo,
		Public:   cmd.Public,
	}

	var network, address string

	switch len(args) {
	case 0:
		network = "tcp"
		address = ":0"

	case 1:
		network = args[0]

	default:
		network = args[0]
		address = args[1]
	}

	if cmd.Handshake {
		network = "tcp"
		address = "127.0.0.1:0"
	}

	if network == "-" {
		if err := server.ServeStdio(); err != nil {
			envLocal.Cancel(err)
		}

		return
	}

	var listener net.Listener

	{
		var err error

		if listener, err = net.Listen(network, address); err != nil {
			envLocal.Cancel(err)
		}

		defer errors.ContextMustClose(envLocal, listener)
	}

	if cmd.Handshake {
		tcpAddr, ok := listener.Addr().(*net.TCPAddr)
		if !ok {
			envLocal.Cancel(
				errors.Errorf(
					"-handshake requires tcp listener, got: %T",
					listener.Addr(),
				),
			)
		}

		// Write directly to os.Stdout so the handshake is a deterministic
		// process-level protocol output a harness can read from a pipe.
		fmt.Fprintf(
			os.Stdout,
			"%d|1|tcp|127.0.0.1:%d|%s\n",
			ProtoHandshakeProtocolVersion,
			tcpAddr.Port,
			ProtoHandshakeProtocolName,
		)
		_ = os.Stdout.Sync()
	} else if addr, ok := listener.Addr().(*net.TCPAddr); ok {
		envLocal.GetUI().Printf("serving drtp on port: %d", addr.Port)
	}

	if err := server.Serve(listener); err != nil {
		envLocal.Cancel(err)
	}
}
