package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
)

func (local *Repo) MakeExternalQueryGroup(
	metaBuilder queries.BuilderOption,
	externalQueryOptions sku.ExternalQueryOptions,
	args ...string,
) (queryGroup *queries.Query, err error) {
	builder := local.MakeQueryBuilderExcludingHidden(
		ids.MakeGenre(),
		metaBuilder,
	)

	if queryGroup, err = builder.BuildQueryGroupWithRepoId(
		externalQueryOptions,
		args...,
	); err != nil {
		err = errors.Wrap(err)
		return queryGroup, err
	}

	queryGroup.ExternalQueryOptions = externalQueryOptions

	return queryGroup, err
}

func (local *Repo) makeQueryBuilder() *queries.Builder {
	return queries.MakeBuilder(
		local.GetEnvRepo(),
		local.GetStore().GetTypedBlobStore(),
		local.GetStore().GetStreamIndex(),
		local.envLua.MakeLuaVMPoolBuilder(),
		local,
	)
}

func (local *Repo) MakeQueryBuilderExcludingHidden(
	genre ids.Genre,
	options queries.BuilderOption,
) *queries.Builder {
	if genre.IsEmpty() {
		genre = ids.MakeGenre(genres.Zettel)
	}

	options = queries.BuilderOptions(
		options,
		queries.BuilderOptionWorkspace(local),
	)

	return local.makeQueryBuilder().WithOptions(
		queries.BuilderOptions(
			queries.BuilderOptionDefaultGenres(genre.Slice()...),
			queries.BuilderOptionRepoId(ids.RepoId{}),
			queries.BuilderOptionFileExtensions(local.GetConfig().GetFileExtensions()),
			queries.BuilderOptionExpanders(local.GetStore().GetAbbrStore().GetAbbr()),
			queries.BuilderOptionHidden(local.GetMatcherDormant()),
			options,
		),
	)
}

func (local *Repo) MakeQueryBuilder(
	genress ids.Genre,
	options queries.BuilderOption,
) *queries.Builder {
	if genress.IsEmpty() {
		genress = ids.MakeGenre(genres.Zettel)
	}

	options = queries.BuilderOptions(
		options,
		queries.BuilderOptionWorkspace(local),
	)

	return local.makeQueryBuilder().WithOptions(
		queries.BuilderOptions(
			queries.BuilderOptionDefaultGenres(genress.Slice()...),
			queries.BuilderOptionRepoId(ids.RepoId{}),
			queries.BuilderOptionFileExtensions(local.GetConfig().GetFileExtensions()),
			queries.BuilderOptionExpanders(local.GetStore().GetAbbrStore().GetAbbr()),
			options,
		),
	)
}
