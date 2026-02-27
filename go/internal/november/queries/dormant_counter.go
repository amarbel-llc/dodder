package queries

import (
	"sync/atomic"

	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type DormantCounter interface {
	sku.Query
	CountArchiviert() int64
	Count() int64
}

type dormantCounter struct {
	count, countArchiviert int64
}

func MakeDormantCounter() DormantCounter {
	return &dormantCounter{}
}

func (m dormantCounter) MatcherLen() int {
	return 0
}

func (m dormantCounter) String() string {
	return ""
}

func (m dormantCounter) Count() int64 {
	return m.count
}

func (m dormantCounter) CountArchiviert() int64 {
	return m.countArchiviert
}

func (matcher *dormantCounter) ContainsSku(tg sku.TransactedGetter) bool {
	matchable := tg.GetSku()

	if !matchable.GetMetadataMutable().GetIndexMutable().GetDormant().Bool() {
		atomic.AddInt64(&matcher.countArchiviert, 1)
		return false
	}

	atomic.AddInt64(&matcher.count, 1)

	return true
}
