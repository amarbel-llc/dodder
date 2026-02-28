package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("info-ssh_agent", &InfoSSHAgent{})
}

type InfoSSHAgent struct {
	Verbose bool
}

var _ interfaces.CommandComponentWriter = (*InfoSSHAgent)(nil)

func (cmd *InfoSSHAgent) SetFlagDefinitions(
	f interfaces.CLIFlagDefinitions,
) {
	f.BoolVar(&cmd.Verbose, "verbose", false, "show key type and comment")
}

func (cmd InfoSSHAgent) Run(req command.Request) {
	ed25519Keys, err := markl.DiscoverSSHAgentEd25519KeysVerbose()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	ecdhKeys, err := markl.DiscoverSSHAgentECDHKeysVerbose()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	allKeys := append(ed25519Keys, ecdhKeys...)

	if len(allKeys) == 0 {
		fmt.Println("no keys found in SSH agent")
		return
	}

	for _, dk := range allKeys {
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
