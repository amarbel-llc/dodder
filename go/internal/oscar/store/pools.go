package store

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
)

// TODO remove entirely — callers should use repool from GetWithRepool
func (store *Store) PutCheckedOutLike(co sku.SkuType) {
}
