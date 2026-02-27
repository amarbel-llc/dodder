package store

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
)

// TODO remove entirely — callers should use repool from GetWithRepool
func (store *Store) PutCheckedOutLike(co sku.SkuType) {
}
