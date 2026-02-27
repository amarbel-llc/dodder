package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
)

func init() {
	utility.AddCmd("info-ssh_agent", &InfoSSHAgent{})
}

type InfoSSHAgent struct{}

func (cmd InfoSSHAgent) Run(req command.Request) {
	keys, err := markl.DiscoverSSHAgentEd25519Keys()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	if len(keys) == 0 {
		fmt.Println("no Ed25519 keys found in SSH agent")
		return
	}

	for _, key := range keys {
		text, err := key.MarshalText()
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}

		fmt.Printf("%s\n", string(text))
	}
}
