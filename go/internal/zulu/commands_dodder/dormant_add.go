package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

func init() {
	utility.AddCmd("dormant-add", &DormantAdd{})
}

type DormantAdd struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd DormantAdd) Run(dep command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(dep)

	localWorkingCopy.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Lock))

	for _, v := range dep.PopArgs() {
		cs, csRepool := catgut.MakeFromString(v)

		if err := localWorkingCopy.GetDormantIndex().AddDormantTag(cs); err != nil {
			csRepool()
			localWorkingCopy.Cancel(err)
		}

		csRepool()
	}

	localWorkingCopy.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock))
}
