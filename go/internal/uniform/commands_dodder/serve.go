package commands_dodder

import (
	"fmt"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/command_components"
	"code.linenisgreat.com/dodder/go/internal/sierra/remote_http"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"tailscale.com/client/local"
)

// HandshakeProtocolVersion is the dodder serve handshake protocol
// version. Modeled on hashicorp/go-plugin and the clown plugin
// protocol; bumped on incompatible changes to the handshake line.
const HandshakeProtocolVersion = 1

// HandshakeProtocolName is the protocol name advertised in field 5
// of the handshake line. Test harnesses can match on this to confirm
// they're talking to a dodder serve and not some other handshake
// protocol on the same line shape.
const HandshakeProtocolName = "dodder-http-v1"

func init() {
	utility.AddCmd("serve", &Serve{})
}

func (cmd Serve) GetDescription() command.Description {
	return command.Description{
		Short: "start the HTTP server",
	}
}

type Serve struct {
	command_components.Env
	command_components_dodder.EnvRepo
	command_components_dodder.LocalWorkingCopy

	TailscaleTLS bool

	// Handshake enables the hashicorp/go-plugin-style handshake. When
	// set, the server forces tcp + ephemeral port, prints a single
	// pipe-delimited handshake line on stdout
	// ("1|1|tcp|127.0.0.1:PORT|dodder-http-v1") after binding, and
	// then begins serving. Diagnostic output is routed to stderr so
	// stdout stays a single-line protocol channel after the
	// handshake. Designed for harnesses (e.g. zz-tests_bats/
	// serve.bats per #150) that need to discover the OS-assigned port
	// without scraping log lines.
	Handshake bool
}

var _ interfaces.CommandComponentWriter = (*Serve)(nil)

func (cmd *Serve) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "network",
				Description: "network type (e.g. tcp, unix)",
			},
			{
				Name:        "address",
				Description: "listen address (e.g. :8080)",
			},
		},
	}}
}

func (cmd *Serve) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

	flagSet.BoolVar(
		&cmd.TailscaleTLS,
		"tailscale-tls",
		false,
		"use tailscale for TLS",
	)

	flagSet.BoolVar(
		&cmd.Handshake,
		"handshake",
		false,
		"emit a hashicorp/go-plugin-style handshake line on stdout after binding (forces tcp + ephemeral port; diagnostic output routed to stderr)",
	)
}

func (cmd Serve) Run(req command.Request) {
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

	server := remote_http.Server{
		EnvLocal: envLocal,
		Repo:     repo,
	}

	if cmd.TailscaleTLS {
		var localClient local.Client
		server.GetCertificate = localClient.GetCertificate
	}

	// TODO switch network to be RemoteServeType
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
		server.ServeStdio()
	} else {
		var listener net.Listener

		{
			var err error

			if listener, err = server.InitializeListener(
				network,
				address,
			); err != nil {
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
			// Write directly to os.Stdout (fd 1) rather than through
			// envLocal.GetOut(), so the handshake is a deterministic
			// process-level protocol output that harnesses can read
			// from a pipe regardless of dodder's UI abstraction.
			fmt.Fprintf(
				os.Stdout,
				"%d|1|tcp|127.0.0.1:%d|%s\n",
				HandshakeProtocolVersion,
				tcpAddr.Port,
				HandshakeProtocolName,
			)
			_ = os.Stdout.Sync()
		}

		if err := server.Serve(listener); err != nil {
			envLocal.Cancel(err)
		}
	}
}
