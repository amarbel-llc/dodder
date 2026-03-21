package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components"
	"code.linenisgreat.com/dodder/go/internal/tango/mcp_dodder"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/flags"
)

func init() {
	utility.AddCmd("serve-web", &ServeWeb{})
}

type ServeWeb struct {
	command_components.Env
	command_components_dodder.EnvRepo
	command_components_dodder.LocalWorkingCopy

	CorsOrigin string
}

var _ interfaces.CommandComponentWriter = (*ServeWeb)(nil)

func (cmd *ServeWeb) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

	flags.StringVar(
		&cmd.CorsOrigin,
		"cors-origin",
		"*",
		"Access-Control-Allow-Origin header value",
	)
}

func (cmd ServeWeb) Run(req command.Request) {
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

	reader := mcp_dodder.NewResourceReader(req.Utility, repo)

	server := &mcp_dodder.WebServer{
		Reader:     reader,
		CorsOrigin: cmd.CorsOrigin,
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

	if err := server.ListenAndServe(network, address); err != nil {
		envLocal.Cancel(err)
	}
}
