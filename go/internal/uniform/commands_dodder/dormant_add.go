package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/catgut"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func init() {
	utility.AddCmd("dormant-add", &DormantAdd{})
}

func (cmd DormantAdd) GetDescription() command.Description {
	return command.Description{
		Short: "add tags to the dormant index",
	}
}

type DormantAdd struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*DormantAdd)(nil)

func (cmd *DormantAdd) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "tags",
			Description: "tag names to add to the dormant index",
			Required:    true,
			Variadic:    true,
		}},
	}}
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
