package import_plan

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
)

type Entry struct {
	object         sku.Transacted
	Classification Classification
	SourceIndex    int
	Height         int
	OriginalTai    ids.Tai
	ErrorCause     string
}

func (e *Entry) GetObject() *sku.Transacted {
	return &e.object
}
