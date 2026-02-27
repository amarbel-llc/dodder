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
	utility.AddCmd("push", &Push{})
}

type Push struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.RemoteTransfer
	command_components_dodder.Query
}

var _ interfaces.CommandComponentWriter = (*Push)(nil)

func (cmd *Push) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.RemoteTransfer.SetFlagDefinitions(flagSet)
	cmd.Query.SetFlagDefinitions(flagSet)
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)
}

func (cmd Push) Run(req command.Request) {
	local := cmd.MakeLocalWorkingCopy(req)

	var remoteObject *sku.Transacted

	{
		var err error

		if remoteObject, err = local.GetObjectFromObjectId(
			req.PopArg("repo-id"),
		); err != nil {
			local.Cancel(err)
		}
	}

	remote := cmd.MakeRemote(req, local, remoteObject)

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		local,
		req.PopArgs(),
	)

	if err := remote.PullQueryGroupFromRemote(
		local,
		queryGroup,
		cmd.WithPrintCopies(true),
	); err != nil {
		local.Cancel(err)
	}
}
