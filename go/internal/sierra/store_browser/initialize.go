//go:build chrest

package store_browser

import (
	"context"
	"net/url"
	"syscall"

	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func (store *Store) Initialize(esi store_workspace.Supplies) (err error) {
	store.externalStoreInfo = esi

	if err = store.browser.Read(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	wg := errors.MakeWaitGroupParallel()

	wg.Do(store.initializeUrls)
	wg.Do(store.initializeIndex)

	if err = wg.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) initializeUrls() (err error) {
	var req browser_items.BrowserRequestGet
	var resp browser_items.HTTPResponseWithRequestPayloadGet

	ui.Log().Print("getting all")

	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	if resp, err = store.browser.GetAll(
		ctxWithTimeout,
		req,
	); err != nil {
		if errors.IsErrno(err, syscall.ECONNREFUSED) {
			if !store.config.GetConfig().Quiet {
				ui.Err().Print("chrest offline")
			}

			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	ui.Log().Print("got all")

	store.urls = make(map[url.URL][]Item, len(resp.RequestPayloadGet))

	if err = store.resetCacheIfNecessary(resp.Response); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, item := range resp.RequestPayloadGet {
		i := Item{Item: item}

		u := i.Url.Url()

		store.urls[u] = append(store.urls[u], i)
		store.itemsById[i.String()] = i
	}

	return err
}
