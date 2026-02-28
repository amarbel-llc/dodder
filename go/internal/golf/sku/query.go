package sku

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type (
	Queryable interface {
		ContainsSku(TransactedGetter) bool
	}

	SigilGetter interface {
		GetSigil() ids.Sigil
	}

	Query interface {
		Queryable
		interfaces.Stringer
	}

	QueryWithSigilAndObjectId interface {
		Query
		SigilGetter
		ContainsObjectId(*ids.ObjectId) bool
	}

	// Used by store_verzeichnisse.binary*
	PrimitiveQueryGroup interface {
		Get(genres.Genre) (QueryWithSigilAndObjectId, bool)
		SigilGetter
		HasHidden() bool
	}

	FuncQuery = func(
		QueryGroup,
		interfaces.FuncIter[*Transacted],
	) (err error)

	FuncPrimitiveQuery = func(
		PrimitiveQueryGroup,
		interfaces.FuncIter[*Transacted],
	) (err error)

	QueryGroup interface {
		PrimitiveQueryGroup
		Query
		SetIncludeHistory()
		GetTags() ids.TagSet
		GetTypes() ids.TypeSet
	}

	ExternalQueryOptions struct {
		ids.RepoId
		ExcludeUntracked  bool
		ExcludeRecognized bool
	}

	ExternalQuery struct {
		ids.RepoId
		QueryGroup
		ExternalQueryOptions
	}
)

func MakePrimitiveQueryGroup() PrimitiveQueryGroup {
	return MakePrimitiveQueryGroupWithSigils(ids.SigilHistory, ids.SigilHidden)
}

func MakePrimitiveQueryGroupWithSigils(ss ...ids.Sigil) PrimitiveQueryGroup {
	return &primitiveQueryGroup{Sigil: ids.MakeSigil(ss...)}
}

type primitiveQueryGroup struct {
	ids.Sigil
}

func (qg *primitiveQueryGroup) SetIncludeHistory() {
	qg.Add(ids.SigilHistory)
}

func (qg *primitiveQueryGroup) HasHidden() bool {
	return false
}

func (qg *primitiveQueryGroup) Get(_ genres.Genre) (QueryWithSigilAndObjectId, bool) {
	return qg, true
}

func (s *primitiveQueryGroup) ContainsSku(_ TransactedGetter) bool {
	return true
}

func (s *primitiveQueryGroup) ContainsObjectId(_ *ids.ObjectId) bool {
	return false
}

func (s *primitiveQueryGroup) GetSigil() ids.Sigil {
	return s.Sigil
}

func (s *primitiveQueryGroup) Each(_ interfaces.FuncIter[Query]) error {
	return nil
}
