package store_fs

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

func (store *Store) checkoutOneIfNecessary(
	options checkout_options.Options,
	transactedGetter sku.TransactedGetter,
) (checkedOut *sku.CheckedOut, item *sku.FSItem, err error) {
	internal := transactedGetter.GetSku()
	checkedOut, _ = GetCheckedOutPool().GetWithRepool()

	sku.Resetter.ResetWith(checkedOut.GetSku(), internal)

	var alreadyCheckedOut bool

	if item, alreadyCheckedOut, err = store.prepareFSItemForCheckOut(
		options,
		checkedOut,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, item, err
	}

	if alreadyCheckedOut && !store.shouldCheckOut(options, checkedOut, true) {
		if err = store.WriteFSItemToExternal(
			item,
			checkedOut.GetSkuExternal(),
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, item, err
		}

		// FSItem does not have the object ID for certain so we need to add it to the
		// external on checkout
		checkedOut.GetSkuExternal().GetObjectId().ResetWithObjectId(checkedOut.GetSku().GetObjectId())
		checkedOut.SetState(checked_out_state.CheckedOut)

		return checkedOut, item, err
	}

	if err = store.checkoutOneForReal(
		options,
		checkedOut,
		item,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, item, err
	}

	// FSItem does not have the object ID for certain so we need to add it to the
	// external on checkout
	checkedOut.GetSkuExternal().GetObjectId().ResetWithObjectId(checkedOut.GetSku().GetObjectId())

	return checkedOut, item, err
}

func (store *Store) prepareFSItemForCheckOut(
	options checkout_options.Options,
	co *sku.CheckedOut,
) (item *sku.FSItem, alreadyCheckedOut bool, err error) {
	fsOptions := GetCheckoutOptionsFromOptions(options)

	if store.config.IsDryRun() ||
		fsOptions.Path == PathOptionTempLocal {
		item = &sku.FSItem{}
		item.Reset()
		return item, alreadyCheckedOut, err
	}

	if item, alreadyCheckedOut = store.Get(co.GetSku().GetObjectId()); alreadyCheckedOut {
		if err = store.HydrateExternalFromItem(
			sku.CommitOptions{
				StoreOptions: sku.GetStoreOptionsRealizeSansProto(),
			},
			item,
			co.GetSku(),
			co.GetSkuExternal(),
		); err != nil {
			if sku.IsErrMergeConflict(err) && options.AllowConflicted {
				err = nil
			} else {
				err = errors.Wrap(err)
				return item, alreadyCheckedOut, err
			}
		}
	} else {
		if item, err = store.ReadFSItemFromExternal(co.GetSkuExternal()); err != nil {
			err = errors.Wrap(err)
			return item, alreadyCheckedOut, err
		}
	}

	// sku.DetermineState(co, true)

	return item, alreadyCheckedOut, err
}
