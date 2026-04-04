package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

func init() {
	utility.AddCmd("dormant-remove", &DormantRemove{})
}

func (cmd DormantRemove) GetDescription() command.Description {
	return command.Description{
		Short: "remove tags from the dormant index",
	}
}

type DormantRemove struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*DormantRemove)(nil)

func (cmd *DormantRemove) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "tags",
			Description: "tag names to remove from the dormant index",
			Required:    true,
			Variadic:    true,
		}},
	}}
}

func (cmd DormantRemove) Run(dep command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(dep)
	localWorkingCopy.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Lock))

	for _, v := range dep.PopArgs() {
		cs, csRepool := catgut.MakeFromString(v)

		if err := localWorkingCopy.GetDormantIndex().RemoveDormantTag(
			cs,
		); err != nil {
			csRepool()
			localWorkingCopy.Cancel(err)
		}

		csRepool()
	}

	localWorkingCopy.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock))
}
