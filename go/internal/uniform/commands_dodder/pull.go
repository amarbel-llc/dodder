package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd("pull", &Pull{})
}

func (cmd Pull) GetDescription() command.Description {
	return command.Description{
		Short: "pull objects from a remote repository",
	}
}

type Pull struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.RemoteTransfer
	command_components_dodder.Query
}

var (
	_ interfaces.CommandComponentWriter = (*Pull)(nil)
	_ command.CommandWithArgs           = (*Pull)(nil)
)

func (cmd *Pull) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{Args: []command.Arg{{
			Name:        "repo-id",
			Description: "remote repository object id (not needed with -direct or -home)",
		}}},
		cmd.Query.GetArgGroup(),
	}
}

func (cmd *Pull) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.RemoteTransfer.SetFlagDefinitions(f)
	cmd.Query.SetFlagDefinitions(f)
	cmd.LocalWorkingCopy.SetFlagDefinitions(f)
}

func (cmd Pull) Run(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	cmd.ResolveImplicitDirectPath(localWorkingCopy)

	var remote repo.Repo
	var object *sku.Transacted
	useProto := false

	if cmd.IsHomeRepoParent() {
		remote = cmd.MakeHomeRepoRemote(req)
	} else if cmd.IsDirectTransfer() {
		remote = cmd.MakeDirectRemoteFromPath(req, localWorkingCopy)
	} else {
		{
			var err error

			if object, err = localWorkingCopy.GetObjectFromObjectId(
				req.PopArg("repo-id"),
			); err != nil {
				localWorkingCopy.Cancel(err)
			}
		}

		if cmd.IsWebSocketProtocol() {
			useProto = true
		} else {
			remote = cmd.MakeRemote(req, localWorkingCopy, object)
		}
	}

	// #287b: when the remote was resolved from the workspace parent, verify
	// its pinned identity (or confirm-pin a legacy workspace) before any
	// transfer. No-op for explicit -direct / stored-remote pulls.
	if !useProto {
		cmd.VerifyOrPinParent(req, localWorkingCopy, remote)
	}

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

	if useProto {
		conn, client := cmd.MakeProtoConnectionFromObject(
			req,
			localWorkingCopy,
			object,
		)

		// A pull discards any RFC-0005 config descriptor the server
		// offers: config is seeded only at clone time, never on pull.
		if _, err := client.Fetch(
			conn,
			qg.String(),
			cmd.WithPrintCopies(true),
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	} else if err := localWorkingCopy.PullQueryGroupFromRemote(
		remote,
		qg,
		cmd.WithPrintCopies(true),
	); err != nil {
		localWorkingCopy.Cancel(err)
	}

	if err := localWorkingCopy.GetEnvWorkspace().UpdateSyncBaseline(
		localWorkingCopy.GetInventoryListStore(),
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
