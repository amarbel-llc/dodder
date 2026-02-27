package commands_dodder

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
)

type Gen struct{}

var _ interfaces.CommandComponentWriter = (*Gen)(nil)

func init() {
	utility.AddCmd(
		"gen",
		&Gen{},
	)
}

func (cmd Gen) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {}

func (cmd Gen) Run(req command.Request) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		env_ui.Options{},
	)

	args := req.PopArgs()

	for _, arg := range args {
		arg = strings.ToLower(arg)

		switch arg {
		case markl.PurposeMadderPrivateKeyV0:
			var id markl.Id

			if err := id.GeneratePrivateKey(
				nil,
				markl.FormatIdAgeX25519Sec,
				arg,
			); err != nil {
				ui.Err().Print(err)
				continue
			}

			envUI.GetUI().Print(id.StringWithFormat())

		case markl.PurposeMadderPrivateKeyV1:
			var id markl.Id

			if err := id.GeneratePrivateKey(
				nil,
				markl.FormatIdAgeX25519Sec,
				arg,
			); err != nil {
				ui.Err().Print(err)
				continue
			}

			envUI.GetUI().Print(id.StringWithFormat())

		case markl.PurposeRepoPrivateKeyV1:
			var id markl.Id

			if err := id.GeneratePrivateKey(
				nil,
				markl.FormatIdEd25519Sec,
				arg,
			); err != nil {
				ui.Err().Print(err)
				continue
			}

			envUI.GetUI().Print(id.StringWithFormat())
		}
	}
}
