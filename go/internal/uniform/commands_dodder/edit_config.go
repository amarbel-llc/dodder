package commands_dodder

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	utility.AddCmd("edit-config", &EditConfig{})
}

type EditConfig struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*EditConfig)(nil)

// GetArgs returns nil: edit-config pops args but ignores them with a warning.
func (cmd *EditConfig) GetArgs() []command.ArgGroup { return nil }

func (cmd EditConfig) GetDescription() command.Description {
	return command.Description{
		Short: "edit the repository configuration",
	}
}

func (cmd EditConfig) Run(req command.Request) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if len(args) > 0 {
		ui.Err().Print("Command edit-config ignores passed in arguments.")
	}

	var digest mad_domain_interfaces.MarklId

	{
		var err error

		if digest, err = editKonfigInVim(localWorkingCopy); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}
	}

	if err := localWorkingCopy.Reset(); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	if err := localWorkingCopy.Lock(); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	defer localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock),
	)

	if _, err := localWorkingCopy.GetStore().UpdateKonfig(digest); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}
}
