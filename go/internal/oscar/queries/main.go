package queries

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Query struct {
	sku.ExternalQueryOptions

	hidden           sku.Query
	optimizedQueries map[genres.Genre]*expSigilAndGenre
	userQueries      map[ids.Genre]*expSigilAndGenre
	types            interfaces.SetMutable[ids.TypeStruct]

	dotOperatorActive bool
	matchOnEmpty      bool

	defaultQuery *Query
}

func (query *Query) GetDefaultQuery() *Query {
	return query.defaultQuery
}

func (query *Query) isDotOperatorActive() bool {
	if query.dotOperatorActive {
		return true
	}

	for _, oq := range query.optimizedQueries {
		if oq.Sigil.ContainsOneOf(ids.SigilExternal) {
			return true
		}
	}

	return false
}

type reducer interface {
	reduce(*buildState) error
}

func (query *Query) reduce(b *buildState) (err error) {
	for _, q := range query.userQueries {
		if err = q.reduce(b); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = query.addOptimized(b, q); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for _, q := range query.optimizedQueries {
		if err = q.reduce(b); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (query *Query) addExactExternalObjectId(
	buildState *buildState,
	externalObjectId sku.ExternalObjectId,
) (err error) {
	if externalObjectId == nil {
		err = errors.ErrorWithStackf("nil object id")
		return err
	}

	exp := buildState.makeQuery()

	exp.Sigil.Add(ids.SigilExternal)
	exp.Sigil.Add(ids.SigilLatest)
	exp.Genre.Add(genres.Must(externalObjectId))
	exp.expObjectIds.external[externalObjectId.String()] = externalObjectId

	if err = query.add(exp); err != nil {
		err = errors.Wrap(err)
		return err
	}

	query.dotOperatorActive = true

	return err
}

func (query *Query) add(q *expSigilAndGenre) (err error) {
	existing, ok := query.userQueries[q.Genre]

	if !ok {
		existing = &expSigilAndGenre{
			Hidden: query.hidden,
			Genre:  q.Genre,
			exp: exp{
				expObjectIds: expObjectIds{
					internal: make(map[string]ObjectId),
				},
			},
		}
	}

	if err = existing.Add(q); err != nil {
		err = errors.Wrap(err)
		return err
	}

	query.userQueries[q.Genre] = existing

	return err
}

func (query *Query) addOptimized(
	buildState *buildState,
	exp *expSigilAndGenre,
) (err error) {
	exp = exp.Clone()
	genres := exp.Slice()

	if len(genres) == 0 {
		genres = buildState.defaultGenres.Slice()
	}

	for _, g := range genres {
		existing, ok := query.optimizedQueries[g]

		if !ok {
			existing = buildState.makeQuery()
			existing.Genre = ids.MakeGenre(g)
		}

		if err = existing.Merge(exp); err != nil {
			err = errors.Wrap(err)
			return err
		}

		query.optimizedQueries[g] = existing
	}

	return err
}

func (query *Query) isEmpty() bool {
	return len(query.userQueries) == 0
}

func (queryGroup *Query) getExactlyOneQuery() (query *expSigilAndGenre, err error) {
	if len(queryGroup.optimizedQueries) != 1 {
		err = errors.ErrorWithStackf(
			"expected exactly 1 genre query but got %d",
			len(queryGroup.optimizedQueries),
		)

		return query, err
	}

	for _, query = range queryGroup.optimizedQueries {
		break
	}

	if query.Sigil.ContainsOneOf(ids.SigilHistory) {
		err = errors.ErrorWithStackf(
			"sigil (%s) includes history, which may return multiple objects",
			query.Sigil,
		)

		return query, err
	}

	return query, err
}

func (queryGroup *Query) getExactlyOneExternalObjectId(
	permitInternal bool,
) (objectId domain_interfaces.ObjectId, sigil ids.Sigil, err error) {
	query, err := queryGroup.getExactlyOneQuery()
	if err != nil {
		return objectId, sigil, err
	}

	internalObjectIds := query.expObjectIds.internal
	oidsLen := len(internalObjectIds)

	externalObjectIds := query.expObjectIds.external
	eoidsLen := len(externalObjectIds)

	switch {
	case eoidsLen == 0 && oidsLen == 1 && permitInternal:
		for _, internalObjectId := range internalObjectIds {
			objectId = internalObjectId
		}

	case eoidsLen == 1 && oidsLen == 0:
		for _, externalObjectId := range externalObjectIds {
			objectId = externalObjectId
		}

		sigil.Add(ids.SigilExternal)

	default:
		err = errors.ErrorWithStackf(
			"expected to exactly 1 object id or 1 external object id but got %d object ids and %d external object ids. Permit internal: %t",
			oidsLen,
			eoidsLen,
			permitInternal,
		)

		return objectId, sigil, err
	}

	sigil = query.GetSigil()

	return objectId, sigil, err
}

func (queryGroup *Query) getExactlyOneObjectId() (objectId ObjectId, sigil ids.Sigil, err error) {
	query, err := queryGroup.getExactlyOneQuery()
	if err != nil {
		return objectId, sigil, err
	}

	internalObjectIds := query.expObjectIds.internal
	oidsLen := len(internalObjectIds)

	externalObjectIds := query.expObjectIds.external
	eoidsLen := len(externalObjectIds)

	switch {
	case eoidsLen == 0 && oidsLen == 1:
		for _, internalObjectId := range internalObjectIds {
			objectId = internalObjectId
		}

	default:
		err = errors.ErrorWithStackf(
			"expected to exactly 1 object id or 1 external object id but got %d object ids and %d external object ids",
			oidsLen,
			eoidsLen,
		)

		return objectId, sigil, err
	}

	sigil = query.GetSigil()

	return objectId, sigil, err
}

func (query *Query) sortedUserQueries() []*expSigilAndGenre {
	userQueries := make([]*expSigilAndGenre, 0, len(query.userQueries))

	for _, userQuery := range query.userQueries {
		userQueries = append(userQueries, userQuery)
	}

	sort.Slice(userQueries, func(i, j int) bool {
		left, right := userQueries[i].Genre, userQueries[j].Genre

		if left.IsEmpty() {
			return false
		}

		if right.IsEmpty() {
			return true
		}

		return left < right
	})

	return userQueries
}

func (query *Query) containsSku(objectGetter sku.TransactedGetter) (ok bool) {
	if query.defaultQuery != nil &&
		!query.defaultQuery.containsSku(objectGetter) {
		return ok
	}

	object := objectGetter.GetSku()

	if len(query.optimizedQueries) == 0 && query.matchOnEmpty {
		ok = true
		return ok
	}

	genre := object.GetGenre()

	expSigilAndGenre, ok := query.optimizedQueries[genres.Must(genre)]

	if !ok || !expSigilAndGenre.ContainsSku(objectGetter) {
		ok = false
		return ok
	}

	ok = true

	return ok
}
