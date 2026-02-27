package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
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
