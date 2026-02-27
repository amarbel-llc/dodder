package stream_index

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/page_id"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type page struct {
	index *Index

	writeLock      sync.Mutex
	pageId         page_id.PageId
	forceFullWrite bool

	additionsHistory additions
	additionsLatest  additions
}

func (page *page) initialize(
	pageId page_id.PageId,
	index *Index,
) {
	page.index = index
	page.pageId = pageId

	history := &pageAdditions{}
	history.initialize(index)
	page.additionsHistory = history

	latest := &pageAdditions{}
	latest.initialize(index)
	page.additionsLatest = latest
}

func (page *page) objectIdStringExists(objectIdString string) bool {
	if page.additionsHistory.containsObjectId(objectIdString) {
		return true
	}

	if page.additionsLatest.containsObjectId(objectIdString) {
		return true
	}

	return false
}

func (page *page) hasChanges() bool {
	return page.additionsHistory.hasChanges() ||
		page.additionsLatest.hasChanges() || page.forceFullWrite
}

func (page *page) lenAdded() int {
	return page.additionsHistory.Len() + page.additionsLatest.Len()
}

func (page *page) add(
	object *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	a := page.additionsHistory

	if page.index.sunrise.Less(object.GetTai()) ||
		options.StreamIndexOptions.ForceLatest {
		a = page.additionsLatest
	}

	a.add(object)

	return err
}
