package store_browser

import (
	"net/url"
	"sync"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

// TODO abstract and regenerate on commit / reindex
func (store *Store) initializeIndex() (err error) {
	if err = store.initializeCache(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var lock sync.Mutex

	if err = store.externalStoreInfo.ReadPrimitiveQuery(
		nil,
		func(object *sku.Transacted) (err error) {
			if !ids.Equals(object.GetType(), store.tipe) {
				return err
			}

			var u *url.URL

			if u, err = store.getUrl(object); err != nil {
				err = nil
				return err
			}

			cl, _ := sku.GetTransactedPool().GetWithRepool()
			sku.TransactedResetter.ResetWith(cl, object)

			lock.Lock()
			defer lock.Unlock()

			{
				existing, ok := store.transactedUrlIndex[*u]

				if !ok {
					existing = sku.MakeTransactedMutableSet()
					store.transactedUrlIndex[*u] = existing
				}

				if err = existing.Add(cl); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			{
				existing, ok := store.tabCache.Rows[object.ObjectId.String()]

				if ok {
					store.transactedItemIndex[existing] = cl
				}
			}

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
