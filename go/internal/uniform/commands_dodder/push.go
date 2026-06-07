package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd("push", &Push{})
}

func (cmd Push) GetDescription() command.Description {
	return command.Description{
		Short: "push objects to a remote repository",
	}
}

type Push struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.RemoteTransfer
	command_components_dodder.Query
}

var (
	_ interfaces.CommandComponentWriter = (*Push)(nil)
	_ command.CommandWithArgs           = (*Push)(nil)
)

func (cmd *Push) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{Args: []command.Arg{{
			Name:        "repo-id",
			Description: "remote repository object id (not needed with -direct or -home)",
		}}},
		cmd.Query.GetArgGroup(),
	}
}

func (cmd *Push) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.RemoteTransfer.SetFlagDefinitions(flagSet)
	cmd.Query.SetFlagDefinitions(flagSet)
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)
}

func (cmd Push) Run(req command.Request) {
	local := cmd.MakeLocalWorkingCopy(req)

	cmd.ResolveImplicitDirectPath(local)

	var remote repo.Repo
	var remoteObject *sku.Transacted
	useProto := false

	if cmd.IsHomeRepoParent() {
		remote = cmd.MakeHomeRepoRemote(req)
	} else if cmd.IsDirectTransfer() {
		remote = cmd.MakeDirectRemoteFromPath(req, local)
	} else {
		{
			var err error

			if remoteObject, err = local.GetObjectFromObjectId(
				req.PopArg("repo-id"),
			); err != nil {
				local.Cancel(err)
			}
		}

		if cmd.IsWebSocketProtocol() {
			useProto = true
		} else {
			remote = cmd.MakeRemote(req, local, remoteObject)
		}
	}

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

	if useProto {
		conn, client := cmd.MakeProtoConnectionFromObject(
			req,
			local,
			remoteObject,
		)

		if err := client.Push(
			conn,
			queryGroup.String(),
			cmd.WithPrintCopies(true),
		); err != nil {
			local.Cancel(err)
		}
	} else if err := remote.PullQueryGroupFromRemote(
		local,
		queryGroup,
		cmd.WithPrintCopies(true),
	); err != nil {
		local.Cancel(err)
	}

	if err := local.GetEnvWorkspace().UpdateSyncBaseline(
		local.GetInventoryListStore(),
	); err != nil {
		local.Cancel(err)
	}
}
