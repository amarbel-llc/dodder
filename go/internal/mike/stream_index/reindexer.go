package stream_index

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
)

type Reindexer struct {
	index *Index
	pages [PageCount]*pageAdditionsFileBacked
}

var _ sku.Reindexer = &Reindexer{}

func (reindexer *Reindexer) Add(
	object *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	return reindexer.index.Add(object, options)
}

func (reindexer *Reindexer) ObjectExists(
	objectId *ids.ObjectId,
) (err error) {
	return reindexer.index.ObjectExists(objectId)
}

func (reindexer *Reindexer) ReadOneMarklIdAdded(
	marklId domain_interfaces.MarklId,
	object *sku.Transacted,
) (ok bool) {
	return reindexer.index.ReadOneMarklIdAdded(marklId, object)
}

func (reindexer *Reindexer) ReadOneMarklId(
	marklId domain_interfaces.MarklId,
	object *sku.Transacted,
) (ok bool) {
	return reindexer.index.ReadOneMarklId(marklId, object)
}
