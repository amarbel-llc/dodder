package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	pkg_query "code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type Query struct {
	sku.ExternalQueryOptions
}

var _ interfaces.CommandComponentWriter = (*Query)(nil)

func (cmd Query) GetArgGroup() command.ArgGroup {
	return command.ArgGroup{
		Name:        "doddish",
		Description: "query terms (AND-combined): genre filters (:z :e :t), tag names, type filters (!type). See doddish(7).",
		Args: []command.Arg{{
			Name:        "doddish",
			Description: "doddish query expression",
			Variadic:    true,
		}},
	}
}

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

func (cmd Query) MakeQueryResolvingFilenames(
	req command.Request,
	options pkg_query.BuilderOption,
	repo *local_working_copy.Repo,
	args []string,
) (query *pkg_query.Query) {
	var resolved []sku.ExternalObjectId
	var remaining []string

	if store, ok := repo.GetWorkspaceStoreForQuery(cmd.RepoId); ok {
		for _, arg := range args {
			if arg == "." {
				// "." means all external objects. Resolve via workspace
				// store for pinned IDs and pass to parser to set
				// dotOperatorActive.
				if externalIds, err := store.GetObjectIdsForString(arg); err == nil {
					resolved = append(resolved, externalIds...)
				}

				remaining = append(remaining, arg)

				continue
			}

			if externalIds, err := store.GetObjectIdsForString(arg); err == nil {
				resolved = append(resolved, externalIds...)
			} else {
				remaining = append(remaining, arg)
			}
		}
	} else {
		remaining = args
	}

	if len(resolved) > 0 {
		options = pkg_query.BuilderOptions(
			options,
			pkg_query.BuilderOptionPinnedExternalObjectIds(resolved),
		)
	}

	options = pkg_query.BuilderOptions(
		options,
		pkg_query.BuilderOptionWorkspace(repo),
	)

	return cmd.MakeQuery(
		req,
		options,
		repo,
		remaining,
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
