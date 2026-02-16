package store_fs

import (
	"os"
	"os/exec"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/object_finalizer"
)

// TODO combine with other method in this file
// Makes hard assumptions about the availability of the blobs associated with
// the *sku.CheckedOut.
func (store *Store) MergeCheckedOut(
	checkedOut *sku.CheckedOut,
	parentNegotiator sku.ParentNegotiator,
	allowMergeConflicts bool,
) (commitOptions sku.CommitOptions, err error) {
	commitOptions.StoreOptions = sku.GetStoreOptionsImport()

	// TODO determine why the internal can ever be null
	if checkedOut.GetSku().GetMetadata().GetObjectDigest().IsNull() ||
		allowMergeConflicts {
		return commitOptions, err
	}

	var conflicts checkout_mode.Mode

	// TODO add checkout_mode.BlobOnly
	if markl.Equals(
		checkedOut.GetSku().GetMetadata().GetObjectDigest(),
		checkedOut.GetSkuExternal().GetMetadata().GetObjectDigest(),
	) {
		commitOptions.StoreOptions = sku.StoreOptions{}
		return commitOptions, err
	} else if objects.EqualerSansTai.Equals(
		checkedOut.GetSku().GetMetadata(),
		checkedOut.GetSkuExternal().GetMetadata(),
	) {
		if !checkedOut.GetSku().GetMetadata().GetTai().Less(
			checkedOut.GetSkuExternal().GetMetadata().GetTai(),
		) {
			// TODO implement retroactive change
		}

		return commitOptions, err
	} else if markl.Equals(
		checkedOut.GetSku().GetMetadata().GetBlobDigest(),
		checkedOut.GetSkuExternal().GetMetadata().GetBlobDigest(),
	) {
		conflicts = checkout_mode.Make(checkout_mode.Metadata)
	} else {
		conflicts = checkout_mode.Make(checkout_mode.MetadataAndBlob)
	}

	// TODO write conflicts
	switch {
	case conflicts.IsBlobOnly():
	case conflicts.IsMetadataOnly():
	default:
	}

	conflicted := sku.Conflicted{
		CheckedOut: checkedOut,
		Local:      checkedOut.GetSku(),
		Remote:     checkedOut.GetSkuExternal(),
	}

	if err = conflicted.FindBestCommonAncestor(parentNegotiator); err != nil {
		err = errors.Wrap(err)
		return commitOptions, err
	}

	if err = conflicted.Remote.SetMother(conflicted.Base); err != nil {
		err = errors.Wrap(err)
		return commitOptions, err
	}

	var skuReplacement *sku.Transacted

	// TODO pass mode / conflicts
	if skuReplacement, err = store.MakeMergedTransacted(
		conflicted,
	); err != nil {
		if sku.IsErrMergeConflict(err) {
			err = nil

			if !allowMergeConflicts {
				if err = store.GenerateConflictMarker(
					conflicted,
					conflicted.CheckedOut,
				); err != nil {
					err = errors.Wrap(err)
					return commitOptions, err
				}
			}

			checkedOut.SetState(checked_out_state.Conflicted)
		} else {
			err = errors.Wrap(err)
		}

		return commitOptions, err
	}

	sku.TransactedResetter.ResetWith(
		checkedOut.GetSkuExternal(),
		skuReplacement,
	)

	return commitOptions, err
}

func (store *Store) Merge(conflicted sku.Conflicted) (err error) {
	var original *sku.FSItem

	if original, err = store.ReadFSItemFromExternal(
		conflicted.CheckedOut.GetSkuExternal(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var skuReplacement *sku.Transacted

	if skuReplacement, err = store.MakeMergedTransacted(conflicted); err != nil {
		if sku.IsErrMergeConflict(err) {
			if err = store.GenerateConflictMarker(
				conflicted,
				conflicted.CheckedOut,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	if original.FDs.Len() == 0 {
		// generate check out item
		// TODO if original is empty, it means this was not a checked out
		// conflict but
		// a remote conflict
	}

	var replacement *sku.FSItem

	if replacement, err = store.ReadFSItemFromExternal(skuReplacement); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if !original.Object.IsEmpty() && !replacement.Object.IsEmpty() {
		if err = files.Rename(
			replacement.Object.GetPath(),
			original.Object.GetPath(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if !original.Blob.IsEmpty() && !replacement.Blob.IsEmpty() {
		if err = files.Rename(
			replacement.Blob.GetPath(),
			original.Blob.GetPath(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (store *Store) checkoutConflictedForMerge(
	tm sku.Conflicted,
	mode checkout_mode.Mode,
) (local, base, remote *sku.FSItem, err error) {
	if _, local, err = store.checkoutOneForMerge(mode, tm.Local); err != nil {
		err = errors.Wrap(err)
		return local, base, remote, err
	}

	if _, base, err = store.checkoutOneForMerge(mode, tm.Base); err != nil {
		err = errors.Wrap(err)
		return local, base, remote, err
	}

	if _, remote, err = store.checkoutOneForMerge(mode, tm.Remote); err != nil {
		err = errors.Wrap(err)
		return local, base, remote, err
	}

	return local, base, remote, err
}

func (store *Store) MakeMergedTransacted(
	conflicted sku.Conflicted,
) (merged *sku.Transacted, err error) {
	// tags have to be manually merged at the moment, even though there is a
	// simple algorithm to merge them automatically. this is because the tags
	// get merged before running diff3, but then local, base, and remote have
	// the merged tag set, and then their signatures become incorrect.
	// if err = conflicted.MergeTags(); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

	var localItem, baseItem, remoteItem *sku.FSItem

	inlineBlob := conflicted.IsAllInlineType(store.config)

	mode := checkout_mode.Make(checkout_mode.MetadataAndBlob)

	if !inlineBlob {
		mode = checkout_mode.Make(checkout_mode.Metadata)
	}

	if localItem, baseItem, remoteItem, err = store.checkoutConflictedForMerge(
		conflicted,
		mode,
	); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	var mergedItem *sku.FSItem
	var diff3Error error

	mergedItem, diff3Error = store.runDiff3(
		localItem,
		baseItem,
		remoteItem,
	)

	if diff3Error != nil {
		err = errors.Wrap(diff3Error)
		return merged, err
	}

	localItem.ResetWith(mergedItem)

	merged, _ = GetExternalPool().GetWithRepool()

	merged.ObjectId.ResetWithObjectId(&conflicted.GetSku().ObjectId)

	if err = store.WriteFSItemToExternal(localItem, merged); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	if err = store.HydrateExternalFromItem(
		sku.CommitOptions{
			StoreOptions: sku.StoreOptions{
				UpdateTai: true,
			},
		},
		mergedItem,
		conflicted.GetSku(),
		conflicted.GetSkuExternal(),
	); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	return merged, err
}

func (store *Store) checkoutOneForMerge(
	mode checkout_mode.Mode,
	sk *sku.Transacted,
) (co *sku.CheckedOut, i *sku.FSItem, err error) {
	if sk == nil {
		i = &sku.FSItem{}
		i.Reset()
		return co, i, err
	}

	options := checkout_options.Options{
		CheckoutMode: mode,
		OptionsWithoutMode: checkout_options.OptionsWithoutMode{
			Force: true,
			StoreSpecificOptions: CheckoutOptions{
				AllowConflicted: true,
				Path:            PathOptionTempLocal,
				// TODO handle binary blobs
				ForceInlineBlob: true,
			},
		},
	}

	co, _ = GetCheckedOutPool().GetWithRepool()
	sku.Resetter.ResetWith(co.GetSku(), sk)

	if i, err = store.ReadFSItemFromExternal(co.GetSku()); err != nil {
		err = errors.Wrap(err)
		return co, i, err
	}

	if err = store.checkoutOneForReal(
		options,
		co,
		i,
	); err != nil {
		err = errors.Wrap(err)
		return co, i, err
	}

	if err = store.WriteFSItemToExternal(i, co.GetSkuExternal()); err != nil {
		err = errors.Wrap(err)
		return co, i, err
	}

	return co, i, err
}

func (store *Store) GenerateConflictMarker(
	conflicted sku.Conflicted,
	checkedOut *sku.CheckedOut,
) (err error) {
	var file *os.File

	if file, err = store.envRepo.GetTempLocal().FileTemp(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(file)
	defer repoolBufferedWriter()
	defer errors.DeferredFlusher(&err, bufferedWriter)

	blobStore := store.storeSupplies.BlobStore.InventoryList
	// TODO assert that left and right both have a mother sig
	finalizer := object_finalizer.Make()

	for object := range conflicted.All() {
		if err = finalizer.FinalizeAndSignIfNecessary(
			object,
			store.envRepo.GetConfigPrivate().Blob,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if _, err = blobStore.WriteBlobToWriter(
		store.envRepo,
		ids.DefaultOrPanic(genres.InventoryList),
		quiter.MakeSeqErrorFromSeq(conflicted.All()),
		bufferedWriter,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var item *sku.FSItem

	if item, err = store.ReadFSItemFromExternal(
		checkedOut.GetSkuExternal(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// TODO make this section less fragile around cwd
	{
		if err = item.GenerateConflictFD(store.envRepo.GetCwd()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if checkedOut.GetSkuExternal().GetGenre() == genres.Zettel {
			var zettelId ids.ZettelId

			if err = zettelId.Set(checkedOut.GetSkuExternal().GetObjectId().String()); err != nil {
				err = errors.Wrap(err)
				return err
			}

			if _, err = env_dir.MakeDirIfNecessaryForStringerWithHeadAndTail(
				zettelId,
				store.envRepo.GetCwd(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if err = os.Rename(
			file.Name(),
			item.Conflict.GetPath(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	checkedOut.SetState(checked_out_state.Conflicted)

	return err
}

func (store *Store) RunMergeTool(
	tool []string,
	conflicted sku.Conflicted,
) (checkedOut *sku.CheckedOut, err error) {
	if len(tool) == 0 {
		err = errors.ErrorWithStackf("no utility provided")
		return checkedOut, err
	}

	checkedOut = conflicted.CheckedOut

	inlineBlob := conflicted.IsAllInlineType(store.config)

	mode := checkout_mode.Make(checkout_mode.All)

	if !inlineBlob {
		mode = checkout_mode.Make(checkout_mode.Metadata)
	}

	var localItem, baseItem, remoteItem *sku.FSItem

	if localItem, baseItem, remoteItem, err = store.checkoutConflictedForMerge(
		conflicted,
		mode,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	var replacementObject *sku.Transacted
	var replacementItem *sku.FSItem

	if replacementObject, err = store.MakeMergedTransacted(conflicted); err != nil {
		var mergeConflict *sku.ErrMergeConflict

		if errors.As(err, &mergeConflict) {
			err = nil
			replacementItem = &mergeConflict.FSItem
		} else {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	} else {
		if replacementItem, err = store.ReadFSItemFromExternal(replacementObject); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	tool = append(
		tool,
		localItem.Object.GetPath(),
		baseItem.Object.GetPath(),
		remoteItem.Object.GetPath(),
		replacementItem.Object.GetPath(),
	)

	// TODO merge blobs

	cmd := exec.Command(tool[0], tool[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Log().Print(cmd.Env)

	if err = cmd.Run(); err != nil {
		err = errors.Wrapf(err, "Cmd: %q", tool)
		return checkedOut, err
	}

	external, externalRepool := GetExternalPool().GetWithRepool()
	defer externalRepool()

	external.ObjectId.ResetWithObjectId(&checkedOut.GetSkuExternal().ObjectId)

	if err = store.WriteFSItemToExternal(localItem, external); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	var file *os.File

	if file, err = files.Open(replacementItem.Object.GetPath()); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	// TODO open blob

	defer errors.DeferredCloser(&err, file)

	if err = store.ReadOneExternalObjectReader(file, external); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	if err = store.DeleteCheckedOut(
		conflicted.CheckedOut,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	checkedOut, _ = GetCheckedOutPool().GetWithRepool()

	sku.TransactedResetter.ResetWith(checkedOut.GetSkuExternal(), external)

	return checkedOut, err
}
