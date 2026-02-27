package commands_dodder

import (
	"net"

	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/lima/command_components"
	"code.linenisgreat.com/dodder/go/internal/xray/remote_http"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/flags"
	"tailscale.com/client/local"
)

func init() {
	utility.AddCmd("serve", &Serve{})
}

type Serve struct {
	command_components.Env
	command_components_dodder.EnvRepo
	command_components_dodder.LocalWorkingCopy

	TailscaleTLS bool
}

var _ interfaces.CommandComponentWriter = (*Serve)(nil)

func (cmd *Serve) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

	flags.BoolVar(
		&cmd.TailscaleTLS,
		"tailscale-tls",
		false,
		"use tailscale for TLS",
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

		if err := server.Serve(listener); err != nil {
			envLocal.Cancel(err)
		}
	}
}
