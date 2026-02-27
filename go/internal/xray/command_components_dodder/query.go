package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	pkg_query "code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/tango/repo"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
)

type Query struct {
	sku.ExternalQueryOptions
}

var _ interfaces.CommandComponentWriter = (*Query)(nil)

func (cmd *Query) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	// TODO switch to repo
	flagSet.Var(&cmd.RepoId, "kasten", "none or Browser")
	flagSet.BoolVar(&cmd.ExcludeUntracked, "exclude-untracked", false, "")
	flagSet.BoolVar(&cmd.ExcludeRecognized, "exclude-recognized", false, "")
}

func (cmd Query) MakeQueryIncludingWorkspace(
	req command.Request,
	options pkg_query.BuilderOption,
	repo *local_working_copy.Repo,
	args []string,
) (query *pkg_query.Query) {
	options = pkg_query.BuilderOptions(
		options,
		pkg_query.BuilderOptionWorkspace(repo),
	)

	return cmd.MakeQuery(
		req,
		options,
		repo,
		args,
	)
}

func (cmd Query) MakeQuery(
	req command.Request,
	options pkg_query.BuilderOption,
	workingCopy repo.Repo,
	args []string,
) (query *pkg_query.Query) {
	var err error

	if query, err = workingCopy.MakeExternalQueryGroup(
		options,
		cmd.ExternalQueryOptions,
		args...,
	); err != nil {
		req.Cancel(err)
	}

	return query
}
