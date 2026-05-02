package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func (store *Store) ReadExternalLikeFromObjectIdLike(
	commitOptions sku.CommitOptions,
	objectIdMaybeExternal interfaces.Stringer,
	internal *sku.Transacted,
) (external sku.ExternalLike, err error) {
	var items []*sku.FSItem

	oidString := objectIdMaybeExternal.String()
	_, isExternal := objectIdMaybeExternal.(domain_interfaces.ExternalObjectId)

	if !isExternal {
		oidString = store.keyForObjectIdString(oidString)
	}

	if items, err = store.GetFSItemsForString(
		store.fsOps.GetCwd(),
		oidString,
		true,
	); err != nil {
		err = errors.Wrapf(err, "ObjectIdString: %q", oidString)
		return external, err
	}

	switch len(items) {
	case 0:
		if !isExternal {
			external, _ = sku.GetTransactedPool().GetWithRepool() //repool:owned

			var objectId ids.ObjectId

			if err = objectId.Set(oidString); err != nil {
				err = errors.Wrap(err)
				return external, err
			}

			if err = store.storeSupplies.ReadOneInto(
				&objectId,
				external.GetSku(),
			); err != nil {
				err = errors.Wrap(err)
				return external, err
			}
		}

		return external, err

	case 1:
		break

	default:
		err = errors.ErrorWithStackf(
			"more than one FSItem (%q) matches object id (%q).",
			items,
			objectIdMaybeExternal,
		)

		return external, err
	}

	item := items[0]

	if external, err = store.ReadExternalFromItem(
		commitOptions,
		item,
		internal,
	); err != nil {
		err = errors.Wrap(err)
		return external, err
	}

	return external, err
}

// Given a sku and an FSItem, return the overlayed external variant. Internal
// can be nil and then only the external data is used.
func (store *Store) ReadExternalFromItem(
	commitOptions sku.CommitOptions,
	item *sku.FSItem,
	internal *sku.Transacted,
) (external *sku.Transacted, err error) {
	external, _ = GetExternalPool().GetWithRepool() //repool:owned

	if err = store.HydrateExternalFromItem(
		commitOptions,
		item,
		internal,
		external,
	); err != nil {
		err = errors.Wrap(err)
		return external, err
	}

	return external, err
}
