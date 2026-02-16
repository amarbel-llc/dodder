package store_browser

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

func (store *Store) DeleteCheckedOut(co *sku.CheckedOut) (err error) {
	external := co.GetSkuExternal()

	var item Item

	if err = item.ReadFromExternal(external); err != nil {
		err = errors.Wrap(err)
		return err
	}

	item.ExternalId = external.GetSkuExternal().GetExternalObjectId().String()

	clonedCo, _ := co.Clone()
	store.deleted[item.Url.Url()] = append(store.deleted[item.Url.Url()], checkedOutWithItem{
		CheckedOut: clonedCo,
		Item:       item,
	})

	return err
}
