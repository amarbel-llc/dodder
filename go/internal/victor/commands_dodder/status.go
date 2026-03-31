package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/box_format"
	pkg_query "code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
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

	query := cmd.MakeQueryResolvingFilenames(
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

	h := localWorkingCopy.GetEnvWorkspace().GetHaustoria()
	if h == nil {
		return
	}

	result, err := h.Status()
	if err != nil {
		localWorkingCopy.GetUI().Printf("haustoria: %s", err)
		return
	}

	ui := localWorkingCopy.GetUI()

	ui.Printf("")
	ui.Printf("[Haustoria: %s]", result.StoreType)

	if len(result.ExternalResources) == 0 {
		ui.Printf("  (no external resources)")
		return
	}

	ui.Printf("  %d external resource(s):", len(result.ExternalResources))

	for _, r := range result.ExternalResources {
		ui.Printf(
			"  %s  %s  %s",
			r.ExternalId,
			r.TypeId,
			fmt.Sprintf("%q", r.Description),
		)
	}
}
