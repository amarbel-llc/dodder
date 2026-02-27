package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("info-pivy_agent", &InfoPivyAgent{})
}

type InfoPivyAgent struct{}

func (cmd InfoPivyAgent) Run(req command.Request) {
	keys, err := markl.DiscoverPivyAgentECDHKeys()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	if len(keys) == 0 {
		fmt.Println("no ECDSA P-256 keys found in pivy-agent")
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
