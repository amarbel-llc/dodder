package commands_dodder

import (
	"encoding/json"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("checkin-json", &CheckinJson{})
}

type CheckinJson struct {
	command_components_dodder.LocalWorkingCopy
}

type TomlBookmark struct {
	ObjectId string
	Tags     []string
	Url      string
}

func (cmd CheckinJson) Run(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	dec := json.NewDecoder(localWorkingCopy.GetInFile())

	for {
		var entry TomlBookmark

		if err := dec.Decode(&entry); err != nil {
			if errors.IsEOF(err) {
				err = nil
				break
			} else {
				localWorkingCopy.Cancel(err)
			}
		}
	}
}
