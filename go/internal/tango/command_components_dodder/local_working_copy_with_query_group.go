package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type LocalWorkingCopyWithQueryGroup struct {
	LocalWorkingCopy
	Query
}

var _ interfaces.CommandComponentWriter = (*LocalWorkingCopyWithQueryGroup)(nil)

func (cmd *LocalWorkingCopyWithQueryGroup) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(f)
	cmd.Query.SetFlagDefinitions(f)
}

func (cmd LocalWorkingCopyWithQueryGroup) MakeLocalWorkingCopyAndQueryGroup(
	req command.Request,
	builderOptions queries.BuilderOption,
) (*local_working_copy.Repo, *queries.Query) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		builderOptions,
		localWorkingCopy,
		req.PopArgs(),
	)

	return localWorkingCopy, queryGroup
}

func (cmd LocalWorkingCopyWithQueryGroup) MakeLocalWorkingCopyAndQueryGroupResolvingFilenames(
	req command.Request,
	builderOptions queries.BuilderOption,
) (*local_working_copy.Repo, *queries.Query) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	queryGroup := cmd.MakeQueryResolvingFilenames(
		req,
		builderOptions,
		localWorkingCopy,
		req.PopArgs(),
	)

	return localWorkingCopy, queryGroup
}
