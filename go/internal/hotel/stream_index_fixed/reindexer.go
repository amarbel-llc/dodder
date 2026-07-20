package stream_index_fixed

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

// Reindexer rebuilds the fixed-size index from scratch. Unlike the
// variable-length index it does not swap in file-backed page additions; the
// fixed index keeps its additions in memory and is flushed page-by-page, so
// the reindexer simply delegates to the underlying index.
type Reindexer struct {
	index *Index
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
	marklId mad_domain_interfaces.MarklId,
	object *sku.Transacted,
) (ok bool) {
	return reindexer.index.ReadOneMarklIdAdded(marklId, object)
}

func (reindexer *Reindexer) ReadOneMarklId(
	marklId mad_domain_interfaces.MarklId,
	object *sku.Transacted,
) (ok bool) {
	return reindexer.index.ReadOneMarklId(marklId, object)
}

func (index *Index) MakeReindexer(
	ctx interfaces.ActiveContext,
) (reindexer sku.Reindexer, err error) {
	if err = index.Initialize(); err != nil {
		err = errors.Wrap(err)
		return reindexer, err
	}

	reindexer = &Reindexer{index: index}

	return reindexer, err
}
