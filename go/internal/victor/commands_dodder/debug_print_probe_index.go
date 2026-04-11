//go:build debug

package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/0/interfaces"
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
