package inventory_list_store

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/options_print"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/env_ui"
	"code.linenisgreat.com/dodder/go/src/hotel/file_lock"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/box_format"
	"code.linenisgreat.com/dodder/go/src/kilo/object_finalizer"
	"code.linenisgreat.com/dodder/go/src/lima/inventory_list_coders"
)

var ErrEmptyInventoryList = newPkgError("empty inventory list")

type Store struct {
	lock sync.Mutex

	envRepo      env_repo.Env
	lockSmith    interfaces.LockSmith
	storeVersion interfaces.StoreVersion
	clock        ids.Clock

	inventoryListBlobStore
	blobBlobStore domain_interfaces.BlobStore

	box *box_format.BoxTransacted

	ui sku.UIStorePrinters
}

var _ sku.InventoryListStore = &Store{}

type inventoryListBlobStore interface {
	domain_interfaces.BlobStore
	object_finalizer.FinalizerGetter

	getType() ids.TypeStruct
	getFormat() sku.ListCoder
	GetInventoryListCoderCloset() inventory_list_coders.Closet

	ReadOneBlobId(domain_interfaces.MarklId) (*sku.Transacted, error)
	WriteInventoryListObject(*sku.Transacted) error

	AllInventoryLists() interfaces.SeqError[*sku.Transacted]
}

func (store *Store) Initialize(
	envRepo env_repo.Env,
	clock ids.Clock,
	inventoryListCoderCloset inventory_list_coders.Closet,
) (err error) {
	*store = Store{
		envRepo:       envRepo,
		lockSmith:     envRepo.GetLockSmith(),
		storeVersion:  envRepo.GetStoreVersion(),
		blobBlobStore: envRepo.GetDefaultBlobStore(),
		clock:         clock,
		box: box_format.MakeBoxTransactedArchive(
			envRepo,
			options_print.Options{}.WithPrintTai(true),
		),
	}

	blobType := ids.MustTypeStruct(
		store.envRepo.GetConfigPublic().Blob.GetInventoryListTypeId(),
	)

	inventoryListBlobStore := envRepo.GetInventoryListBlobStore()
	coder := inventoryListCoderCloset.GetCoderForType(blobType)

	store.inventoryListBlobStore = &blobStoreV1{
		envRepo:                  envRepo,
		pathLog:                  envRepo.FileInventoryListLog(),
		blobType:                 blobType,
		BlobStore:                inventoryListBlobStore,
		listFormat:               coder,
		inventoryListCoderCloset: inventoryListCoderCloset,
	}

	return err
}

func (store *Store) SetUIDelegate(ud sku.UIStorePrinters) {
	store.ui = ud
}

func (store *Store) GetEnv() env_ui.Env {
	return store.GetEnvRepo()
}

func (store *Store) Flush() (err error) {
	wg := errors.MakeWaitGroupParallel()
	return wg.GetError()
}

// TODO pass errors.Context
func (store *Store) FormatForVersion(
	storeVersion interfaces.StoreVersion,
) sku.ListCoder {
	tipe := ids.GetOrPanic(
		store.envRepo.GetConfigPublic().Blob.GetInventoryListTypeId(),
	).TypeStruct

	return store.GetInventoryListCoderCloset().GetCoderForType(tipe)
}

func (store *Store) GetTai() ids.Tai {
	if store.clock == nil {
		return ids.NowTai()
	} else {
		return store.clock.GetTai()
	}
}

func (store *Store) GetEnvRepo() env_repo.Env {
	return store.envRepo
}

func (store *Store) MakeWorkingList() (workingList *sku.WorkingList, err error) {
	var mover domain_interfaces.BlobWriter

	if mover, err = store.blobBlobStore.MakeBlobWriter(
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return workingList, err
	}

	workingList = sku.MakeWorkingList(
		store.getFormat(),
		mover,
		func(object *sku.Transacted) (err error) {
			// TODO swap this to not overwrite, as when importing from remotes, we want to
			// keep their signatures
			if err = store.inventoryListBlobStore.GetObjectFinalizer().FinalizeAndSignOverwrite(
				object,
				store.envRepo.GetConfigPrivate().Blob,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		})

	return workingList, err
}

func (store *Store) Create(
	openList *sku.WorkingList,
) (object *sku.Transacted, err error) {
	if openList.Len() == 0 {
		err = errors.Wrap(ErrEmptyInventoryList)
		return object, err
	}

	if !store.lockSmith.IsAcquired() {
		err = file_lock.ErrLockRequired{
			Operation: "create inventory list",
		}

		return object, err
	}

	object, _ = sku.GetTransactedPool().GetWithRepool()

	object.GetMetadataMutable().GetTypeMutable().ResetWithType(store.getType())
	object.GetMetadataMutable().GetDescriptionMutable().ResetWith(
		openList.GetDescription(),
	)

	tai := store.GetTai()

	if err = object.ObjectId.SetWithSeq(tai.ToSeq()); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	object.SetTai(tai)

	if err = openList.Close(); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	actual := openList.GetMarklId()
	// expected := &merkle.Id{}

	expected := object.GetBlobDigest()
	// if err = expected.SetMerkleId(merkle.HRPObjectBlobDigestSha256V0,
	// actual.GetBytes()); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

	ui.Log().Print("expected", expected, "actual", actual)

	if expected.IsNull() {
		object.SetBlobDigest(actual)
	} else {
		if err = markl.AssertEqual(expected, actual); err != nil {
			err = errors.Wrap(err)
			return object, err
		}
	}

	// if err = object.Metadata.GetRepoPubKeyMutable().SetMerkleId(
	// 	merkle.HRPRepoPubKeyV1,
	// 	store.envRepo.GetConfigPublic().Blob.GetPublicKey(),
	// ); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

	if err = store.WriteInventoryListObject(object); err != nil {
		err = errors.Wrapf(err, "OpenList: %d", openList.Len())
		return object, err
	}

	return object, err
}

func (store *Store) WriteInventoryListBlob(
	remoteBlobStore domain_interfaces.BlobStore,
	object *sku.Transacted,
	list *sku.HeapTransacted,
) (err error) {
	if list.Len() == 0 {
		if !object.GetBlobDigest().IsNull() {
			err = errors.ErrorWithStackf(
				"inventory list has non-empty blob but passed in list is empty. %q",
				sku.String(object),
			)

			return err
		}

		return err
	}

	var writeCloser domain_interfaces.BlobWriter

	if writeCloser, err = store.envRepo.GetDefaultBlobStore().MakeBlobWriter(
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, writeCloser)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(writeCloser)
	defer repoolBufferedWriter()

	if _, err = store.GetInventoryListCoderCloset().WriteBlobToWriter(
		store.envRepo,
		object.GetType(),
		quiter.MakeSeqErrorFromSeq(list.All()),
		bufferedWriter,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	actual := writeCloser.GetMarklId()
	expected := object.GetBlobDigest()

	ui.Log().Print("expected", expected, "actual", actual)

	if expected.IsNull() {
		object.SetBlobDigest(actual)
	} else {
		if err = markl.AssertEqual(expected, actual); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	// if !s.af.HasBlob(t.GetBlobSha()) {
	// 	err = errors.Errorf(
	// 		"inventory list blob missing after write (%d bytes, %d skus): %q",
	// 		n,
	// 		skus.Len(),
	// 		sku.String(t),
	// 	)

	// 	return
	// }

	// if _, _, err = s.blobStore.GetTransactedWithBlob(
	// 	t,
	// ); err != nil {
	// 	err = errors.Wrapf(err, "Blob Sha: %q", actual)
	// 	return
	// }

	return err
}

func (store *Store) AllInventoryListContents(
	blobSha domain_interfaces.MarklId,
) sku.Seq {
	return store.GetInventoryListCoderCloset().IterInventoryListBlobSkusFromBlobStore(
		store.getType(),
		store.blobBlobStore,
		blobSha,
	)
}

func (store *Store) ReadLast() (*sku.Transacted, error) {
	max, _ := sku.GetTransactedPool().GetWithRepool()

	for list, err := range store.AllInventoryLists() {
		if err != nil {
			return nil, errors.Wrap(err)
		}

		if sku.TransactedLessor.LessPtr(max, list) {
			sku.TransactedResetter.ResetWith(max, list)
		}
	}

	return max, nil
}

func (store *Store) AllInventoryListObjectsAndContents() interfaces.SeqError[sku.ObjectWithList] {
	return func(yield func(sku.ObjectWithList, error) bool) {
		var objectWithList sku.ObjectWithList

		for listObject, iterErr := range store.AllInventoryLists() {
			objectWithList.List = listObject
			objectWithList.Object = listObject

			if iterErr != nil {
				if !yield(objectWithList, iterErr) {
					return
				}

				continue
			}

			if !yield(objectWithList, nil) {
				return
			}

			iter := store.AllInventoryListContents(
				listObject.GetBlobDigest(),
			)

			for object, iterErr := range iter {
				objectWithList.Object = object

				if !yield(objectWithList, iterErr) {
					return
				}
			}
		}
	}
}

func (store *Store) AllInventoryListsSorted() sku.Seq {
	return func(yield func(*sku.Transacted, error) bool) {
		// TODO optimize space by storing digest and tai but not anything else
		var lists collections_slice.Slice[*sku.Transacted]

		for list, iterErr := range store.AllInventoryLists() {
			if iterErr != nil {
				if !yield(nil, errors.Wrap(iterErr)) {
					return
				}
			}

			lists.Append(list)
		}

		lists.SortWithComparer(sku.TransactedCompare)

		for _, list := range lists {
			if !yield(list, nil) {
				return
			}
		}
	}
}
