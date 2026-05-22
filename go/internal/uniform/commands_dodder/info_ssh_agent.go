package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// TODO-P2 consolidate info-ssh_agent and info-pivy_agent into
// info-available_keys, exposing all keys on ssh/pivy agents in formats
// accepted by `dodder init -private_key` and `madder init -private_key`.
func init() {
	utility.AddCmd("info-ssh_agent", &InfoSSHAgent{})
}

type InfoSSHAgent struct{}

var _ command.CommandWithArgs = (*InfoSSHAgent)(nil)

// GetArgs returns nil: no positional arguments.
func (cmd *InfoSSHAgent) GetArgs() []command.ArgGroup { return nil }

func (cmd InfoSSHAgent) GetDescription() command.Description {
	return command.Description{
		Short: "list keys in the SSH agent",
	}
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

		if req.Utility.GetConfig().GetVerbose() {
			fmt.Printf("%s\t%s\t%s\n", dk.KeyType, dk.Comment, string(text))
		} else {
			fmt.Printf("%s\n", string(text))
		}
	}
}
