//go:build chrest

package store_browser

import (
	"context"
	"syscall"

	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"golang.org/x/exp/maps"
)

func (store *Store) flushUrls() (err error) {
	if len(store.deleted) == 0 && len(store.added) == 0 {
		return err
	}

	var resp browser_items.HTTPResponseWithRequestPayloadPut

	deleted := make(map[string]checkedOutWithItem, len(store.deleted))

	var req browser_items.BrowserRequestPut
	req.Deleted = make([]browser_items.Item, 0, len(store.deleted))

	for _, is := range store.deleted {
		for _, i := range is {
			req.Deleted = append(req.Deleted, i.Item.Item)
			deleted[i.Item.Item.ExternalId] = i
		}
	}

	for _, is := range store.added {
		for _, i := range is {
			req.Added = append(req.Added, i.Item.Item)
		}
	}

	if !store.config.GetConfig().IsDryRun() {
		ctx := context.Background()
		ctxWithTimeout, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()

		if resp, err = store.browser.PutAll(
			ctxWithTimeout,
			req,
		); err != nil {
			if errors.IsErrno(err, syscall.ECONNREFUSED) {
				ui.Err().Print("chrest offline")
				err = nil
			} else {
				err = errors.Wrap(err)
				return err
			}
		}

		if err = store.resetCacheIfNecessary(resp.Response); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		for _, is := range store.deleted {
			for _, i := range is {
				resp.Deleted = append(resp.Deleted, i.Item.Item)
			}
		}

		for _, is := range store.added {
			for _, i := range is {
				resp.Added = append(resp.Added, i.Item.Item)
			}
		}
	}

	for _, i := range resp.RequestPayloadPut.Added {
		// TODO emit changes
		store.tabCache.Rows[i.ExternalId] = i.Id
	}

	for _, item := range resp.RequestPayloadPut.Deleted {
		delete(store.tabCache.Rows, item.ExternalId)

		originalItem, ok := deleted[item.ExternalId]

		if !ok {
			err = errors.ErrorWithStackf(
				"missing item with id %q from deleted cache: %q",
				item.ExternalId,
				maps.Keys(deleted),
			)

			return err
		}

		if err = store.itemDeletedStringFormatWriter(
			originalItem.CheckedOut,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	clear(store.added)
	clear(store.deleted)

	if err = store.flushCache(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
