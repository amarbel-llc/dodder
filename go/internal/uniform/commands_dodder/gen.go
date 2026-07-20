package commands_dodder

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type Gen struct{}

var _ interfaces.CommandComponentWriter = (*Gen)(nil)

func (cmd Gen) GetDescription() command.Description {
	return command.Description{
		Short: "generate cryptographic keys",
	}
}

func init() {
	utility.AddCmd(
		"gen",
		&Gen{},
	)
}

func (cmd *Gen) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "key-purposes",
			Description: "key purpose identifiers to generate",
			Variadic:    true,
			EnumValues: []string{
				markl.PurposeMadderPrivateKeyV0,
				markl.PurposeMadderPrivateKeyV1,
				markl.PurposeRepoPrivateKeyV1,
			},
		}},
	}}
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
