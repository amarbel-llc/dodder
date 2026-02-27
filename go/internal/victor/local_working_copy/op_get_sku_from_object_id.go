package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	pkg_query "code.linenisgreat.com/dodder/go/internal/november/queries"
)

// TODO add to repo.Repo interface
func (local *Repo) GetZettelFromObjectId(
	objectIdString string,
) (object *sku.Transacted, err error) {
	builder := local.MakeQueryBuilder(ids.MakeGenre(genres.Zettel), nil)

	var query *pkg_query.Query

	if query, err = builder.BuildQueryGroupWithRepoId(
		sku.ExternalQueryOptions{},
		objectIdString,
	); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	if object, err = local.GetStore().QueryExactlyOneExternal(query); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	return object, err
}

// TODO add to repo.Repo interface
func (local *Repo) GetObjectFromObjectId(
	objectIdString string,
) (object *sku.Transacted, err error) {
	builder := local.MakeQueryBuilder(ids.MakeGenre(genres.All()...), nil)

	var queryGroup *pkg_query.Query

	if queryGroup, err = builder.BuildQueryGroupWithRepoId(
		sku.ExternalQueryOptions{},
		objectIdString,
	); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	if object, err = local.GetStore().QueryExactlyOneExternal(
		queryGroup,
	); err != nil {
		if errors.IsErrNotFound(err) {
			err = errors.BadRequestf(
				"object with id %q not found",
				objectIdString,
			)
		} else {
			err = errors.Wrapf(err, "ObjectIdString: %q", objectIdString)
		}

		return object, err
	}

	return object, err
}
