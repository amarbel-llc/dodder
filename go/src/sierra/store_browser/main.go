package store_browser

import (
	"bufio"
	"net/url"
	"sync"

	"code.linenisgreat.com/chrest/go/src/charlie/browser_items"
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/charlie/collections_value"
	"code.linenisgreat.com/dodder/go/src/charlie/toml"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/sku_json_fmt"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/oscar/store_workspace"
	"code.linenisgreat.com/dodder/go/src/quebec/env_workspace"
	"code.linenisgreat.com/dodder/go/src/romeo/store_config"
)

const DefaultTimeout = 2e9

type transacted struct {
	sync.Mutex
	interfaces.SetMutable[*ids.ObjectId]
}

type checkedOutWithItem struct {
	*sku.CheckedOut
	Item
}

type Store struct {
	config            store_config.Store
	externalStoreInfo store_workspace.Supplies
	tipe              ids.TypeStruct
	browser           browser_items.BrowserProxy

	tabCache cache

	urls map[url.URL][]Item

	lock    sync.Mutex
	deleted map[url.URL][]checkedOutWithItem
	added   map[url.URL][]checkedOutWithItem

	itemsById map[string]Item

	transacted transacted

	transactedUrlIndex  map[url.URL]sku.TransactedMutableSet
	transactedItemIndex map[browser_items.ItemId]*sku.Transacted

	itemDeletedStringFormatWriter interfaces.FuncIter[*sku.CheckedOut]
}

func Make(
	configStore store_config.Store,
	envRepo env_repo.Env,
	itemDeletedStringFormatWriter interfaces.FuncIter[*sku.CheckedOut],
) *Store {
	store := &Store{
		config:    configStore,
		tipe:      ids.MustTypeStruct("toml-bookmark"),
		deleted:   make(map[url.URL][]checkedOutWithItem),
		added:     make(map[url.URL][]checkedOutWithItem),
		itemsById: make(map[string]Item),
		transacted: transacted{
			SetMutable: collections_value.MakeMutableValueSet(
				quiter.StringerKeyer[*ids.ObjectId]{},
			),
		},
		transactedUrlIndex: make(
			map[url.URL]sku.TransactedMutableSet,
		),
		transactedItemIndex: make(
			map[browser_items.ItemId]*sku.Transacted,
		),
		itemDeletedStringFormatWriter: itemDeletedStringFormatWriter,
	}

	return store
}

func (store *Store) GetExternalStoreLike() store_workspace.StoreLike {
	return store
}

func (store *Store) ReadAllExternalItems() error {
	return nil
}

func (store *Store) GetObjectIdsForString(
	value string,
) (ids []sku.ExternalObjectId, err error) {
	item, ok := store.itemsById[value]

	if !ok {
		err = errors.ErrorWithStackf("not a browser item id")
		return ids, err
	}

	ids = append(ids, item.GetExternalObjectId())

	return ids, err
}

func (store *Store) Flush() (err error) {
	waitGropu := errors.MakeWaitGroupParallel()

	waitGropu.Do(store.flushUrls)

	if err = waitGropu.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// TODO limit this to being used only by *Item.ReadFromExternal
func (store *Store) getUrl(object *sku.Transacted) (u *url.URL, err error) {
	var blobReader domain_interfaces.BlobReader

	if blobReader, err = store.externalStoreInfo.GetDefaultBlobStore().MakeBlobReader(
		object.GetBlobDigest(),
	); err != nil {
		err = errors.Wrap(err)
		return u, err
	}

	defer errors.DeferredCloser(&err, blobReader)

	var tomlBookmark sku_json_fmt.TomlBookmark

	dec := toml.NewDecoder(blobReader)

	if err = dec.Decode(&tomlBookmark); err != nil {
		err = errors.Wrapf(
			err,
			"Sha: %s, Object Id: %s",
			object.GetBlobDigest(),
			object.GetObjectId(),
		)
		return u, err
	}

	if u, err = url.Parse(tomlBookmark.Url); err != nil {
		err = errors.Wrap(err)
		return u, err
	}

	return u, err
}

func (store *Store) CheckoutOne(
	options checkout_options.Options,
	tg sku.TransactedGetter,
) (checkedOut sku.SkuType, err error) {
	object := tg.GetSku()

	if !ids.Equals(object.GetMetadata().GetType(), store.tipe) {
		err = env_workspace.ErrUnsupportedType{Type: object.GetMetadata().GetType()}
		err = errors.Wrap(err)
		return checkedOut, err
	}

	var yourl *url.URL

	if yourl, err = store.getUrl(object); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	checkedOut, _ = GetCheckedOutPool().GetWithRepool()
	var item Item

	if err = item.Url.Set(yourl.String()); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	item.ExternalId = object.ObjectId.String()
	item.Id.Type = "tab"

	sku.TransactedResetter.ResetWith(checkedOut.GetSku(), object)
	sku.TransactedResetter.ResetWith(checkedOut.GetSkuExternal().GetSku(), object)
	checkedOut.SetState(checked_out_state.JustCheckedOut)
	checkedOut.GetSkuExternal().ExternalType = ids.MustTypeStruct("!browser-tab")

	if err = item.WriteToExternal(checkedOut.GetSkuExternal()); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	checkedOut.GetSkuExternal().RepoId = store.externalStoreInfo.RepoId

	store.lock.Lock()
	defer store.lock.Unlock()

	clonedCo, _ := checkedOut.Clone()
	existing := store.added[*yourl]
	store.added[*yourl] = append(existing, checkedOutWithItem{
		CheckedOut: clonedCo,
		Item:       item,
	})

	return checkedOut, err
}

func (store *Store) QueryCheckedOut(
	query *queries.Query,
	output interfaces.FuncIter[sku.SkuType],
) (err error) {
	// o := sku.CommitOptions{
	// 	Mode: object_mode.ModeRealizeSansProto,
	// }

	ex := executor{
		store: store,
		query: query,
		out:   output,
	}

	for u, items := range store.urls {
		matchingUrls, exactIndexURLMatch := store.transactedUrlIndex[u]

		for _, item := range items {
			var matchingTabId *sku.Transacted
			var trackedFromBefore bool

			tabId := item.Id
			matchingTabId, trackedFromBefore = store.transactedItemIndex[tabId]

			if trackedFromBefore {
				if err = ex.tryToEmitOneExplicitlyCheckedOut(
					matchingTabId,
					item,
				); err != nil {
					err = errors.Wrapf(err, "Item: %#v", item)
					return err
				}
			} else if !exactIndexURLMatch {
				if err = ex.tryToEmitOneUntracked(item); err != nil {
					err = errors.Wrapf(err, "Item: %#v", item)
					return err
				}
			} else if exactIndexURLMatch {
				for matching := range matchingUrls.All() {
					if err = ex.tryToEmitOneRecognized(
						matching,
						item,
					); err != nil {
						err = errors.Wrapf(err, "Item: %#v", item)
						return err
					}
				}
			}
		}
	}

	return err
}

// TODO support updating bookmarks without overwriting. Maybe move to
// toml-bookmark type
func (store *Store) SaveBlob(object sku.ExternalLike) (err error) {
	var blobWriter domain_interfaces.BlobWriter

	if blobWriter, err = store.externalStoreInfo.GetDefaultBlobStore().MakeBlobWriter(
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	var item Item

	if err = item.ReadFromExternal(object.GetSku()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	tomlBookmark := sku_json_fmt.TomlBookmark{
		Url: item.Url.String(),
	}

	func() {
		bw := bufio.NewWriter(blobWriter)
		defer errors.DeferredFlusher(&err, bw)

		enc := toml.NewEncoder(bw)

		if err = enc.Encode(tomlBookmark); err != nil {
			err = errors.Wrap(err)
			return
		}
	}()

	markl.SetDigester(
		object.GetSku().GetMetadataMutable().GetBlobDigestMutable(),
		blobWriter,
	)

	return err
}

func (store *Store) asBlobSaver() sku.BlobSaver {
	return store
}

func (store *Store) UpdateCheckoutFromCheckedOut(
	options checkout_options.OptionsWithoutMode,
	col sku.SkuType,
) (err error) {
	return err
}
