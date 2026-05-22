//go:build debug

package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func (cmd DebugPrintProbeIndex) GetDescription() command.Description {
	return command.Description{
		Short: "print stream index probes",
	}
}

type DebugPrintProbeIndex struct {
	command_components_dodder.LocalWorkingCopy
}

var _ interfaces.CommandComponentWriter = (*DebugPrintProbeIndex)(nil)

func init() {
	utility.AddCmd("debug-print-probe-index", &DebugPrintProbeIndex{})
}

func (*DebugPrintProbeIndex) SetFlagDefinitions(
	interfaces.CLIFlagDefinitions,
) {
}

func (cmd DebugPrintProbeIndex) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	if err := repo.GetStore().GetStreamIndex().PrintAllProbes(); err != nil {
		repo.Cancel(err)
	}
}
