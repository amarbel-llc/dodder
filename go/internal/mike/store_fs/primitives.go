package store_fs

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

// Internal may be nil, which means that the external is hydrated without an
// overlay.
func (store *Store) HydrateExternalFromItem(
	options sku.CommitOptions,
	item *sku.FSItem,
	internal *sku.Transacted,
	external *sku.Transacted,
) (err error) {
	if internal != nil {
		external.GetObjectIdMutable().ResetWithObjectId(internal.GetObjectId())
	}

	if err = item.WriteToSku(
		external,
		store.envRepo,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var mode checkout_mode.Mode

	if mode, err = item.GetCheckoutModeOrError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	switch {
	case mode.IsBlobOnly():
		if err = store.readOneExternalBlob(external, internal, item); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case mode.IncludesMetadata():
		if item.Object.IsStdin() {
			if err = store.ReadOneExternalObjectReader(os.Stdin, external); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else {
			if err = store.readOneExternalObject(external, internal, item); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

	case mode.IsBlobRecognized():
		objects.Resetter.ResetWith(
			external.GetMetadataMutable(),
			internal.GetMetadataMutable(),
		)

	default:
		err = checkout_mode.MakeErrInvalidCheckoutModeMode(mode)
		return err
	}

	if options.Clock == nil {
		options.Clock = item
	}

	if err = store.WriteFSItemToExternal(item, external); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Don't apply the proto object as that would artificially create deltas
	options.StoreOptions.ApplyProto = false

	if err = store.storeSupplies.Commit(
		external,
		options,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// Internal can be nil which means that no overlaying is done.
func (store *Store) readOneExternalObject(
	external *sku.Transacted,
	internal *sku.Transacted,
	item *sku.FSItem,
) (err error) {
	if internal != nil {
		objects.Resetter.ResetWith(
			external.GetMetadataMutable(),
			internal.GetMetadataMutable(),
		)
	}

	var f *os.File

	if f, err = files.Open(item.Object.GetPath()); err != nil {
		err = errors.Wrapf(err, "Item: %s", item.Debug())
		return err
	}

	defer errors.DeferredCloser(&err, f)

	if err = store.ReadOneExternalObjectReader(f, external); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) readOneExternalBlob(
	external *sku.Transacted,
	internal *sku.Transacted,
	item *sku.FSItem,
) (err error) {
	objects.Resetter.ResetWith(
		external.GetMetadataMutable(),
		internal.GetMetadataMutable(),
	)

	// TODO use cache
	{
		var writeCloser domain_interfaces.BlobWriter

		if writeCloser, err = store.envRepo.GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, writeCloser)

		var file *os.File

		if file, err = files.OpenExclusiveReadOnly(
			item.Blob.GetPath(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, file)

		if _, err = io.Copy(writeCloser, file); err != nil {
			err = errors.Wrap(err)
			return err
		}

		markl.SetDigester(
			external.GetMetadataMutable().GetBlobDigestMutable(),
			writeCloser,
		)
	}

	return err
}
