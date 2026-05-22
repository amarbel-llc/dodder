package commands_dodder

import (
	"encoding/json"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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

// GetArgs returns nil: reads from stdin, not positional arguments.
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
