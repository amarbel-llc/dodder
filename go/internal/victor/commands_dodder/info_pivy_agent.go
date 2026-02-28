package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("info-pivy_agent", &InfoPivyAgent{})
}

type InfoPivyAgent struct{}

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
