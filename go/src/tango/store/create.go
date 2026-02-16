package store

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/hotel/file_lock"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

func (store *Store) CreateOrUpdateDefaultProto(
	external *sku.Transacted,
	storeOptions sku.StoreOptions,
) (err error) {
	options := sku.CommitOptions{
		Proto:        store.protoZettel,
		StoreOptions: storeOptions,
	}

	if err = store.CreateOrUpdate(external, options); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) CreateOrUpdate(
	external *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	options.AddToInventoryList = true
	options.UpdateTai = true
	options.RunHooks = true
	options.Validate = true

	if err = store.Commit(
		external,
		options,
	); err != nil {
		err = errors.WrapExceptSentinel(err, errors.ErrExists)
		return err
	}

	return err
}

func (store *Store) CreateOrUpdateBlobDigest(
	objectId domain_interfaces.ObjectId,
	blobDigest domain_interfaces.MarklId,
) (object *sku.Transacted, err error) {
	if !store.GetEnvRepo().GetLockSmith().IsAcquired() {
		err = file_lock.ErrLockRequired{
			Operation: fmt.Sprintf(
				"create or update %s",
				objectId.GetGenre(),
			),
		}

		return object, err
	}

	var objectRepool interfaces.FuncRepool
	object, objectRepool = sku.GetTransactedPool().GetWithRepool()

	if err = object.ObjectId.Set(objectId.String()); err != nil {
		objectRepool()
		object = nil

		err = errors.Wrap(err)
		return object, err
	}

	if err = store.ReadOneInto(objectId, object); err != nil {
		objectRepool()
		object = nil

		if errors.IsErrNotFound(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return object, err
		}
	}

	object.SetBlobDigest(blobDigest)

	if err = store.Commit(
		object,
		sku.CommitOptions{StoreOptions: sku.GetStoreOptionsUpdate()},
	); err != nil {
		err = errors.WrapExceptSentinel(err, errors.ErrExists)
		return object, err
	}

	return object, err
}

type RevertId struct {
	*ids.ObjectId
	Sig domain_interfaces.MarklId
}

func (store *Store) RevertTo(
	revertId RevertId,
) (err error) {
	if revertId.Sig.IsEmpty() {
		return err
	}

	if !store.GetEnvRepo().GetLockSmith().IsAcquired() {
		err = file_lock.ErrLockRequired{
			Operation: "revert",
		}

		return err
	}

	object, objectRepool := sku.GetTransactedPool().GetWithRepool()
	defer objectRepool()

	if !store.streamIndex.ReadOneMarklId(
		revertId.Sig,
		object,
	) {
		err = errors.Errorf("object with sig %q not found", revertId.Sig)
		return err
	}

	if err = store.Commit(
		object,
		sku.CommitOptions{StoreOptions: sku.GetStoreOptionsUpdate()},
	); err != nil {
		err = errors.WrapExceptSentinel(err, errors.ErrExists)
		return err
	}

	return err
}

func (store *Store) CreateOrUpdateCheckedOut(
	object sku.SkuType,
	updateCheckout bool,
) (err error) {
	external := object.GetSkuExternal()
	internal := external.GetSku()

	if !store.GetEnvRepo().GetLockSmith().IsAcquired() {
		err = file_lock.ErrLockRequired{
			Operation: fmt.Sprintf(
				"create or update %s",
				internal.GetObjectId(),
			),
		}

		return err
	}

	if err = store.Commit(
		external,
		sku.CommitOptions{StoreOptions: sku.GetStoreOptionsCreate()},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if !updateCheckout {
		return err
	}

	if err = store.UpdateCheckoutFromCheckedOut(
		checkout_options.OptionsWithoutMode{Force: true},
		object,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
