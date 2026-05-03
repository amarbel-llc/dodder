package commands_dodder

import (
	"sort"
	"strconv"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func init() {
	utility.AddCmd("peek-zettel-ids", &PeekZettelIds{})
}

func (cmd PeekZettelIds) GetDescription() command.Description {
	return command.Description{
		Short: "preview available zettel ids",
	}
}

type PeekZettelIds struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*PeekZettelIds)(nil)

func (cmd *PeekZettelIds) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "count",
			Description: "number of ids to show (0 for all)",
		}},
	}}
}

func (cmd PeekZettelIds) Run(req command.Request) {
	args := req.PopArgs()

	n := 0

	if len(args) > 0 {
		{
			var err error

			if n, err = strconv.Atoi(args[0]); err != nil {
				errors.ContextCancelWithErrorf(
					req,
					"expected int but got %s",
					args[0],
				)
			}
		}
	}

	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	var hs []*ids.ZettelId

	{
		var err error
		if hs, err = localWorkingCopy.GetStore().GetZettelIdIndex().PeekZettelIds(
			n,
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	sort.Slice(
		hs,
		func(i, j int) bool {
			return hs[i].String() < hs[j].String()
		},
	)

	for i, h := range hs {
		localWorkingCopy.GetUI().Printf("%d: %s", i, h)
	}

	return
}
