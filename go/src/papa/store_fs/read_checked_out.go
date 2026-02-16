package store_fs

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

// TODO what does this even do. This caused [cervicis/marshall.zettel !task pom-2 project-2021-zit-bugs project-25q1-zit_workspaces-crit] fix issue with tags other than workspace in `checkin -organize` bei…
// likely due to this method overriding tags that were set by organize. maybe
// this bug existed before workspaces?
func (store *Store) RefreshCheckedOut(
	checkedOut *sku.CheckedOut,
) (err error) {
	var item *sku.FSItem

	if item, err = store.ReadFSItemFromExternal(
		checkedOut.GetSkuExternal(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.HydrateExternalFromItem(
		sku.CommitOptions{
			StoreOptions: sku.StoreOptions{
				UpdateTai: true,
			},
		},
		item,
		checkedOut.GetSku(),
		checkedOut.GetSkuExternal(),
	); err != nil {
		if sku.IsErrMergeConflict(err) {
			checkedOut.SetState(checked_out_state.Conflicted)

			if err = checkedOut.GetSkuExternal().ObjectId.SetWithId(
				&checkedOut.GetSku().ObjectId,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else {
			err = errors.Wrapf(err, "Cwd: %#v", item.Debug())
			return err
		}
	}

	return err
}

func (store *Store) ReadCheckedOutFromTransacted(
	object *sku.Transacted,
) (checkedOut *sku.CheckedOut, err error) {
	checkedOut, _ = GetCheckedOutPool().GetWithRepool()

	if err = store.readIntoCheckedOutFromTransacted(
		object,
		checkedOut,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	return checkedOut, err
}

func (store *Store) readIntoCheckedOutFromTransacted(
	object *sku.Transacted,
	checkedOut *sku.CheckedOut,
) (err error) {
	if checkedOut.GetSku() != object {
		sku.Resetter.ResetWith(checkedOut.GetSku(), object)
	}

	ok := false

	var fsItem *sku.FSItem

	if fsItem, ok = store.Get(&object.ObjectId); !ok {
		err = errors.MakeErrNotFound(object.GetObjectId())
		return err
	}

	if err = store.HydrateExternalFromItem(
		sku.CommitOptions{
			StoreOptions: sku.StoreOptions{
				LockfileOptions: sku.LockfileOptions{
					AllowTypeFailure: true,
					AllowTagFailure:  true,
				},
				UpdateTai: true,
			},
		},
		fsItem,
		object,
		checkedOut.GetSkuExternal(),
	); err != nil {
		if errors.IsNotExist(err) {
			// no-op
		} else if sku.IsErrMergeConflict(err) {
			checkedOut.SetState(checked_out_state.Conflicted)

			if err = checkedOut.GetSkuExternal().ObjectId.SetWithId(
				&object.ObjectId,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else {
			err = errors.Wrapf(err, "Cwd: %#v", fsItem)
		}

		return err
	}

	return err
}
