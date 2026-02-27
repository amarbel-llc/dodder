package queries

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type primitive struct {
	*Query
}

func (qg primitive) HasHidden() bool {
	return qg.hidden != nil
}

// TODO migrate this to the query executor
func (qg primitive) Get(g genres.Genre) (sku.QueryWithSigilAndObjectId, bool) {
	q, ok := qg.optimizedQueries[g]
	return q, ok
}

func (qg primitive) GetSigil() (s ids.Sigil) {
	for _, q := range qg.optimizedQueries {
		s.Add(q.Sigil)
	}

	return s
}
