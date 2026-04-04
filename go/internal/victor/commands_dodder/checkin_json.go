package commands_dodder

import (
	"encoding/json"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("checkin-json", &CheckinJson{})
}

func (cmd CheckinJson) GetDescription() command.Description {
	return command.Description{
		Short: "commit objects from JSON on stdin",
	}
}

type CheckinJson struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*CheckinJson)(nil)

func (cmd *CheckinJson) GetArgs() []command.ArgGroup { return nil }

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
