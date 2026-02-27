package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/box_format"
	pkg_query "code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("status", &Status{})
}

type Status struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup
}

func (cmd Status) Run(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)
	localWorkingCopy.GetEnvWorkspace().AssertNotTemporary(req)

	query := cmd.MakeQueryIncludingWorkspace(
		req,
		pkg_query.BuilderOptions(
			pkg_query.BuilderOptionDefaultGenres(genres.All()...),
			pkg_query.BuilderOptionDefaultSigil(ids.SigilExternal),
			pkg_query.BuilderOptionHidden(nil),
		),
		localWorkingCopy,
		req.PopArgs(),
	)

	printer := localWorkingCopy.PrinterCheckedOut(
		box_format.CheckedOutHeaderState{},
	)

	if err := localWorkingCopy.GetStore().QuerySkuType(
		query,
		func(co sku.SkuType) (err error) {
			if err = printer(co); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
