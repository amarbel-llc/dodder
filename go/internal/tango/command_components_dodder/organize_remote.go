package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	pkg_query "code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// RunOrganizeAgainstRemote opens qg's matching objects, as resolved against
// remote (not the local store, since clone/init-workspace act on objects
// that don't exist locally yet), in an organize outline for accept/reject
// filtering, and returns a query narrowed to the surviving objects. Edits to
// tags/descriptions are not meaningful pre-pull, so only entry deletion
// (narrowing) is honored, mirroring checkout's -organize.
func (cmd Query) RunOrganizeAgainstRemote(
	req command.Request,
	local *local_working_copy.Repo,
	remote repo.Repo,
	qg *pkg_query.Query,
	instructions string,
) (narrowed *pkg_query.Query) {
	opOrganize := repo_actions.MakeOrganize(
		local,
		orgie.Metadata{
			RepoId: qg.RepoId,
			OptionCommentSet: orgie.MakeOptionCommentSet(
				nil,
				&orgie.OptionCommentUnknown{
					Value: instructions,
				},
			),
		},
	)
	opOrganize.DontUseQueryGroupForOrganizeMetadata = true

	originalRepoId := qg.RepoId
	qg.RepoId.Reset()

	organizeResults, err := opOrganize.RunWithRemoteQueryGroup(remote, qg)
	if err != nil {
		req.Cancel(errors.Wrap(err))
		return narrowed
	}

	changes, err := orgie.ChangesFromResults(
		local.GetConfig().GetPrintOptions(),
		organizeResults,
	)
	if err != nil {
		req.Cancel(errors.Wrap(err))
		return narrowed
	}

	b := local.MakeQueryBuilder(
		ids.MakeGenre(genres.All()...),
		nil,
	).WithTransacted(
		changes.After.AsTransactedSet(),
		ids.SigilExternal,
	).WithOptions(pkg_query.BuilderOptions(
		pkg_query.BuilderOptionDoNotMatchEmpty(),
		pkg_query.BuilderOptionRequireNonEmptyQuery(),
	))

	if narrowed, err = b.BuildQueryGroup(); err != nil {
		req.Cancel(errors.Wrap(err))
		return narrowed
	}

	narrowed.RepoId = originalRepoId

	return narrowed
}
