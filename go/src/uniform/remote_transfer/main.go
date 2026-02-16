package remote_transfer

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/blob_transfers"
	"code.linenisgreat.com/dodder/go/src/kilo/object_finalizer"
	"code.linenisgreat.com/dodder/go/src/lima/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/src/oscar/store_workspace"
	"code.linenisgreat.com/dodder/go/src/tango/repo"
)

func Make(
	options repo.ImporterOptions,
	storeOptions sku.StoreOptions,
	envRepo env_repo.Env,
	typedInventoryListBlobStore inventory_list_coders.Closet,
	indexObject sku.Index,
	storeExternalMergeCheckedOut store_workspace.MergeCheckedOut,
	storeObject sku.RepoStore,
) repo.Importer {
	if options.BlobGenres.IsEmpty() {
		options.BlobGenres = ids.MakeGenreAll()
	}

	importer := &importer{
		typedInventoryListBlobStore: typedInventoryListBlobStore,
		index:                       indexObject,
		storeExternal:               storeExternalMergeCheckedOut,
		storeObject:                 storeObject,
		envRepo:                     envRepo,
		blobGenres:                  options.BlobGenres,
		excludeObjects:              options.ExcludeObjects,
		remoteBlobStore:             options.RemoteBlobStore,
		blobCopierDelegate:          options.BlobCopierDelegate,
		allowMergeConflicts:         options.AllowMergeConflicts,
		parentNegotiator:            options.ParentNegotiator,
		checkedOutPrinter:           options.CheckedOutPrinter,
		storeOptions:                storeOptions,
	}

	importer.committer.initialize(options, envRepo, storeObject)

	if importer.blobCopierDelegate == nil &&
		importer.remoteBlobStore.BlobStore != nil &&
		options.PrintCopies {
		importer.blobCopierDelegate = sku.MakeBlobCopierDelegate(
			envRepo.GetUI(),
			false,
		)
	}

	importer.blobImporter = blob_transfers.MakeBlobImporter(
		envRepo.GetEnvBlobStore(),
		importer.remoteBlobStore,
		blob_stores.MakeBlobStoreMap(envRepo.GetDefaultBlobStore()),
	)

	importer.blobImporter.CopierDelegate = importer.blobCopierDelegate

	return importer
}

type importer struct {
	committer committer

	blobImporter blob_transfers.BlobImporter

	finalizer                   object_finalizer.Finalizer
	typedInventoryListBlobStore inventory_list_coders.Closet
	index                       sku.Index
	storeExternal               store_workspace.MergeCheckedOut
	storeObject                 sku.RepoStore
	envRepo                     env_repo.Env
	blobGenres                  ids.Genre
	excludeObjects              bool
	remoteBlobStore             blob_stores.BlobStoreInitialized
	blobCopierDelegate          interfaces.FuncIter[sku.BlobCopyResult]
	storeOptions                sku.StoreOptions
	allowMergeConflicts         bool
	parentNegotiator            sku.ParentNegotiator
	checkedOutPrinter           interfaces.FuncIter[*sku.CheckedOut]
}

func (importer importer) GetCheckedOutPrinter() interfaces.FuncIter[*sku.CheckedOut] {
	return importer.checkedOutPrinter
}

func (importer *importer) SetCheckedOutPrinter(
	printer interfaces.FuncIter[*sku.CheckedOut],
) {
	importer.checkedOutPrinter = printer
}

func (importer importer) Import(
	external *sku.Transacted,
) (checkedOut *sku.CheckedOut, err error) {
	errors.ContextContinueOrPanic(importer.envRepo)

	if err = importer.ImportBlobIfNecessary(external); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	if external.GetGenre() == genres.InventoryList {
		if checkedOut, err = importer.importInventoryList(external); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	} else {
		if checkedOut, err = importer.importLeaf(external); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	return checkedOut, err
}

func (importer importer) importInventoryList(
	list *sku.Transacted,
) (checkedOut *sku.CheckedOut, err error) {
	if err = genres.InventoryList.AssertGenre(list.GetGenre()); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	blobDigest := list.GetBlobDigest()

	if !importer.envRepo.GetDefaultBlobStore().HasBlob(blobDigest) {
		err = env_dir.ErrBlobMissing{
			BlobId: func() domain_interfaces.MarklId { c, _ := markl.Clone(blobDigest); return c }(),
		}

		return checkedOut, err
	}

	seq := importer.typedInventoryListBlobStore.StreamInventoryListBlobSkus(
		list,
	)

	for object, errIter := range seq {
		if errIter != nil {
			err = errors.Wrap(errIter)
			return checkedOut, err
		}

		if _, err = importer.Import(
			object,
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	// TODO decide whether we should rewrite the imported inventory list
	// according to this repo's inventory list type
	// inventoryListTypeString :=
	// importer.envRepo.GetConfigPublic().Blob.GetInventoryListTypeString()

	// if listObject.GetType().String() != inventoryListTypeString {
	// 	listObject.Metadata.Type = ids.GetOrPanic(inventoryListTypeString).Type
	// }

	if checkedOut, err = importer.importLeaf(
		list,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	return checkedOut, err
}

func (importer importer) importLeaf(
	external *sku.Transacted,
) (checkedOut *sku.CheckedOut, err error) {
	if importer.excludeObjects {
		err = ErrSkipped
		return checkedOut, err
	}

	// TODO address this terrible hack? How should config objects be handled by
	// remotes?
	if external.GetGenre() == genres.Config {
		err = genres.MakeErrUnsupportedGenre(external.GetGenre())
		return checkedOut, err
	}

	checkedOut, _ = sku.GetCheckedOutPool().GetWithRepool()

	sku.Resetter.ResetWith(checkedOut.GetSkuExternal(), external)

	checkedOut.GetSkuExternal().GetMetadataMutable().GetObjectDigestMutable().Reset()
	configGenesis := importer.envRepo.GetConfigPrivate().Blob

	// TODO confirm repo pub key

	// TODO set this as an importer option
	if checkedOut.GetSkuExternal().GetMetadata().GetObjectSig().IsNull() {
		if err = importer.finalizer.FinalizeAndSignOverwrite(
			checkedOut.GetSkuExternal(),
			configGenesis,
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	} else {
		if err = importer.finalizer.FinalizeUsingObject(
			checkedOut.GetSkuExternal(),
			importer.envRepo.GetObjectDigestType(),
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	if importer.index != nil {
		_, err = importer.index.ReadOneObjectIdTai(
			checkedOut.GetSkuExternal().GetObjectId(),
			checkedOut.GetSkuExternal().GetTai(),
		)

		if err == nil {
			err = errors.ErrExists
			return checkedOut, err
		} else if errors.IsErrNotFound(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	ui.TodoP4("cleanup")
	if err = importer.storeObject.ReadOneInto(
		checkedOut.GetSkuExternal().GetObjectId(),
		checkedOut.GetSku(),
	); err != nil {
		if errors.IsErrNotFound(err) {
			if err = importer.importNewObject(
				checkedOut.GetSkuExternal(),
			); err != nil {
				err = errors.Wrap(err)
				return checkedOut, err
			}

			return checkedOut, err
		} else {
			err = errors.Wrapf(err, "ObjectId: %s", external.GetObjectId())
		}

		return checkedOut, err
	}

	var commitOptions sku.CommitOptions

	// TODO extra commit option setting into its own function
	if importer.storeExternal != nil {
		if commitOptions, err = importer.storeExternal.MergeCheckedOut(
			checkedOut,
			importer.parentNegotiator,
			importer.allowMergeConflicts,
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}

		if checkedOut.GetState() == checked_out_state.Conflicted {
			if !importer.allowMergeConflicts {
				if err = importer.checkedOutPrinter(checkedOut); err != nil {
					err = errors.Wrap(err)
					return checkedOut, err
				}

				return checkedOut, err
			}
		}
	}

	commitOptions.Validate = false

	if err = importer.committer.Commit(
		checkedOut.GetSkuExternal(),
		commitOptions,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	if err = importer.checkedOutPrinter(checkedOut); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	return checkedOut, err
}

func (importer importer) importNewObject(
	object *sku.Transacted,
) (err error) {
	options := sku.CommitOptions{
		Clock:              object,
		StoreOptions:       importer.storeOptions,
		DontAddMissingType: true,
	}

	options.UpdateTai = false

	if err = importer.committer.Commit(
		object,
		options,
	); err != nil {
		err = errors.WrapExceptSentinel(err, errors.ErrExists)
		return err
	}

	return err
}

func (importer importer) ImportBlobIfNecessary(
	object *sku.Transacted,
) (err error) {
	copyResult := importer.blobImporter.ImportBlobToStoreIfNecessary(
		importer.envRepo.GetDefaultBlobStore(),
		object.GetMetadata().GetBlobDigest(),
		object,
	)

	if err = copyResult.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
