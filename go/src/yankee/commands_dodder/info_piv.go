package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
)

func init() {
	utility.AddCmd("info-piv", &InfoPIV{})
}

type InfoPIV struct{}

func (cmd InfoPIV) Run(req command.Request) {
	tokens, err := markl.DiscoverPIVTokens()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	if len(tokens) == 0 {
		fmt.Println("no PIV tokens found")
		return
	}

	for _, token := range tokens {
		ref := markl.EncodePIVReference(token.GUID, token.SlotId)

		var pivId markl.Id
		if err := pivId.SetMarklId(
			markl.FormatIdEd25519PIV,
			ref[:],
		); err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}

		text, err := pivId.MarshalText()
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}

		fmt.Printf(
			"%s  # serial=%d, slot=%02x\n",
			string(text),
			token.Serial,
			token.SlotId,
		)
	}
}
