package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// TODO-P2 consolidate info-ssh_agent and info-pivy_agent into
// info-available_keys, exposing all keys on ssh/pivy agents in formats
// accepted by `dodder init -private_key` and `madder init -private_key`.
func init() {
	utility.AddCmd("info-pivy_agent", &InfoPivyAgent{})
}

type InfoPivyAgent struct{}

var _ command.CommandWithArgs = (*InfoPivyAgent)(nil)

// GetArgs returns nil: no positional arguments.
func (cmd *InfoPivyAgent) GetArgs() []command.ArgGroup { return nil }

func (cmd InfoPivyAgent) GetDescription() command.Description {
	return command.Description{
		Short: "list ECDSA keys in pivy-agent",
	}
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

		if req.Utility.GetConfig().GetVerbose() {
			fmt.Printf("%s\t%s\t%s\n", dk.KeyType, dk.Comment, string(text))
		} else {
			fmt.Printf("%s\n", string(text))
		}
	}
}
