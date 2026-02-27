//go:build chrest

package store_browser

import (
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

// TODO decide how this should behave
func (store *Store) UpdateTransacted(object *sku.Transacted) (err error) {
	if !ids.Equals(object.GetType(), store.tipe) {
		return err
	}

	var yourl *url.URL

	if yourl, err = store.getUrl(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	_, ok := store.urls[*yourl]

	if !ok {
		return err
	}

	return err
}
