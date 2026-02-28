package commands_dodder

import (
	"sort"
	"strconv"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("peek-zettel-ids", &PeekZettelIds{})
}

type PeekZettelIds struct {
	command_components_dodder.LocalWorkingCopy
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
