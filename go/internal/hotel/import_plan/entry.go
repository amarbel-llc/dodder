package import_plan

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
)

type Entry struct {
	object         sku.Transacted
	Classification Classification
	SourceIndex    int
	Height         int
	OriginalTai    ids.Tai
	ErrorCause     string
	Options        *sku.CommitOptions
	// SkipLuaHooks commits this entry with StoreOptions.SkipLuaHooks set,
	// on top of whatever options the committing path uses. Honored by both
	// ExecutePlan and remote_transfer.CommitPlan.
	SkipLuaHooks bool
}

func (e *Entry) GetObject() *sku.Transacted {
	return &e.object
}
