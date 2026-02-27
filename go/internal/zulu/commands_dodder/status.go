package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/box_format"
	pkg_query "code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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
