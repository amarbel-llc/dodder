package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func init() {
	utility.AddCmd("pull", &Pull{})
}

type Pull struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.RemoteTransfer
	command_components_dodder.Query
}

var _ interfaces.CommandComponentWriter = (*Pull)(nil)

func (cmd *Pull) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.RemoteTransfer.SetFlagDefinitions(f)
	cmd.Query.SetFlagDefinitions(f)
	cmd.LocalWorkingCopy.SetFlagDefinitions(f)
}

func (cmd Pull) Run(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	var object *sku.Transacted

	{
		var err error

		if object, err = localWorkingCopy.GetObjectFromObjectId(
			req.PopArg("repo-id"),
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	remote := cmd.MakeRemote(req, localWorkingCopy, object)

	qg := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		localWorkingCopy,
		req.PopArgs(),
	)

	if err := localWorkingCopy.PullQueryGroupFromRemote(
		remote,
		qg,
		cmd.WithPrintCopies(true),
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
