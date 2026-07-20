package store

import (
	"fmt"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/file_lock"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

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

type RevertId struct {
	*ids.ObjectId
	Sig mad_domain_interfaces.MarklId
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

	// MergeCheckedOut refreshes a clean checked-out working copy to the
	// reverted state (and conflict-merges a dirty one), so `dodder revert`
	// of a checked-out object does not leave its working copy stale. Mirrors
	// UpdateObject / organize; a no-op when the object is not checked out.
	storeOptions := sku.GetStoreOptionsUpdate()
	storeOptions.MergeCheckedOut = true

	if err = store.Commit(
		object,
		sku.CommitOptions{StoreOptions: storeOptions},
	); err != nil {
		err = errors.WrapExceptSentinel(err, errors.ErrExists)
		return err
	}

	return err
}

func (store *Store) createOrUpdateCheckedOut(
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
