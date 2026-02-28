package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("info-pivy_agent", &InfoPivyAgent{})
}

type InfoPivyAgent struct {
	Verbose bool
}

var _ interfaces.CommandComponentWriter = (*InfoPivyAgent)(nil)

func (cmd *InfoPivyAgent) SetFlagDefinitions(
	f interfaces.CLIFlagDefinitions,
) {
	f.BoolVar(&cmd.Verbose, "verbose", false, "show key type and comment")
}

func (cmd InfoPivyAgent) Run(req command.Request) {
	keys, err := markl.DiscoverPivyAgentECDHKeysVerbose()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	if len(keys) == 0 {
		fmt.Println("no ECDSA P-256 keys found in pivy-agent")
		return
	}

	for _, dk := range keys {
		text, err := dk.Id.MarshalText()
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}

		if cmd.Verbose {
			fmt.Printf("%s\t%s\t%s\n", dk.KeyType, dk.Comment, string(text))
		} else {
			fmt.Printf("%s\n", string(text))
		}
	}
}
