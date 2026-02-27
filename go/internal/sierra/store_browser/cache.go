//go:build chrest

package store_browser

import (
	"encoding/gob"
	"net/http"
	"os"
	"path"

	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
)

type cache struct {
	LaunchTime ids.Tai
	Rows       map[string]browser_items.ItemId // map[browserItem.ExternalId]browserItemId
}

func (store *Store) getCachePath() string {
	return path.Join(store.externalStoreInfo.DirCache, "tab_cache")
}

func (store *Store) initializeCache() (err error) {
	store.tabCache.Rows = make(map[string]browser_items.ItemId)

	var file *os.File

	if file, err = files.OpenExclusiveReadOnly(
		store.getCachePath(),
	); err != nil {
		if errors.IsNotExist(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedReader, repool := pool.GetBufferedReader(file)
	defer repool()

	dec := gob.NewDecoder(bufferedReader)

	if err = dec.Decode(&store.tabCache); err != nil {
		ui.Err().Printf("browser tab cache parse failed: %s", err)
		err = nil
		return err
	}

	return err
}

func (store *Store) resetCacheIfNecessary(
	resp *http.Response,
) (err error) {
	if resp == nil {
		return err
	}

	timeRaw := resp.Header.Get("X-Chrest-Startup-Time")

	var newLaunchTime ids.Tai

	if err = newLaunchTime.SetFromRFC3339(timeRaw); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if newLaunchTime.Equals(store.tabCache.LaunchTime) {
		return err
	}

	store.tabCache.LaunchTime = newLaunchTime
	clear(store.tabCache.Rows)

	return err
}

func (store *Store) flushCache() (err error) {
	var file *os.File

	if file, err = files.OpenExclusiveWriteOnly(
		store.getCachePath(),
	); err != nil {
		if errors.IsNotExist(err) {
			if file, err = files.TryOrMakeDirIfNecessary(
				store.getCachePath(),
				files.CreateExclusiveWriteOnly,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	defer errors.DeferredCloser(&err, file)

	bufferedWriter, repool := pool.GetBufferedWriter(file)
	defer repool()

	defer errors.DeferredFlusher(&err, bufferedWriter)

	dec := gob.NewEncoder(bufferedWriter)

	if err = dec.Encode(&store.tabCache); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
