package remote_transfer

import (
	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_transfers"
	"code.linenisgreat.com/dodder/go/internal/golf/object_finalizer"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/kilo/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	mad_blob_io "code.linenisgreat.com/madder/go/pkgs/blob_io"
	"code.linenisgreat.com/madder/go/pkgs/blob_stores"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func Make(
	options repo.ImporterOptions,
	storeOptions sku.StoreOptions,
	envRepo env_repo.Env,
	typedInventoryListBlobStore inventory_list_coders.Closet,
	indexObject sku.Index,
	storeExternalMergeCheckedOut store_workspace.MergeCheckedOut,
	storeObject sku.RepoStore,
	storeCommitter sku.StoreCommitter,
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
		continueOnError:             options.ContinueOnError,
		forbidBloblessTypes:         options.ForbidBloblessTypes,
		remoteBlobStore:             options.RemoteBlobStore,
		blobCopierDelegate:          options.BlobCopierDelegate,
		allowMergeConflicts:         options.AllowMergeConflicts,
		parentNegotiator:            options.ParentNegotiator,
		checkedOutPrinter:           options.CheckedOutPrinter,
		storeOptions:                storeOptions,
	}

	importer.committer.initialize(options, storeCommitter)

	if importer.blobCopierDelegate == nil &&
		importer.remoteBlobStore != nil &&
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
	continueOnError             bool
	forbidBloblessTypes         bool
	remoteBlobStore             mad_domain_interfaces.BlobStore
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

	if !importer.envRepo.GetReadBlobStore().HasBlob(blobDigest) {
		err = mad_blob_io.ErrBlobMissing{
			BlobId: func() mad_domain_interfaces.MarklId { c, _ := markl.Clone(blobDigest); return c }(), //repool:owned
		}

		return checkedOut, err
	}

	seq := importer.typedInventoryListBlobStore.StreamInventoryListBlobSkus(
		list,
	)

	subObjectErrors := errors.MakeGroupBuilder()

	for object, errIter := range seq {
		if errIter != nil {
			err = errors.Wrap(errIter)
			return checkedOut, err
		}

		if _, importErr := importer.Import(
			object,
		); importErr != nil {
			if importer.skipBloblessType(importErr) {
				continue
			}

			if importer.continueOnError {
				subObjectErrors.Add(
					errors.Wrapf(importErr, "Object: %s", sku.String(object)),
				)
			} else {
				err = errors.Wrap(importErr)
				return checkedOut, err
			}
		}
	}

	if subObjectErrors.Len() > 0 {
		err = subObjectErrors.GetError()
		return checkedOut, err
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

// skipBloblessType reports whether importErr is a benign blobless-type skip
// that should be tolerated rather than propagated. A blobless type definition
// (a type object with a null blob digest) is a documented skip (#291); by
// default it is skipped with a stderr notice so a single such object in history
// does not abort an entire push/import. The -forbid-blobless-types option opts
// into treating it as fatal again.
func (importer importer) skipBloblessType(importErr error) bool {
	if importer.forbidBloblessTypes || !IsErrBloblessTypeSkipped(importErr) {
		return false
	}

	ui.Err().Print(importErr)

	return true
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

	checkedOut, _ = sku.GetCheckedOutPool().GetWithRepool() //repool:owned

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
		// recomputes the digest of an imported object that already
		// carries a sig: derive the digest purpose from that sig so
		// v2-signed imports into a v3 repo (and vice versa) verify
		if err = importer.finalizer.FinalizeUsingObject(
			checkedOut.GetSkuExternal(),
			checkedOut.GetSkuExternal().ObjectDigestPurposeOrDefault(
				importer.envRepo.GetObjectDigestType(),
			),
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	if checkedOut.GetSkuExternal().GetMetadata().GetBlobDigest().IsNull() &&
		checkedOut.GetSkuExternal().GetGenre() == genres.Type {
		err = ErrBloblessTypeSkipped{
			ObjectId: checkedOut.GetSkuExternal().GetObjectId().String(),
			TypeId:   checkedOut.GetSkuExternal().GetType().String(),
		}
		return checkedOut, err
	}

	if importer.index != nil {
		var existing *sku.Transacted

		existing, err = importer.index.ReadOneObjectIdTai(
			checkedOut.GetSkuExternal().GetObjectId(),
			checkedOut.GetSkuExternal().GetTai(),
		)

		if err == nil {
			localBlobDigest := existing.GetBlobDigest().String()
			remoteBlobDigest := checkedOut.GetSkuExternal().GetBlobDigest().String()

			if localBlobDigest != remoteBlobDigest {
				err = ErrObjectIdTaiCollision{
					ObjectId:     checkedOut.GetSkuExternal().GetObjectId().String(),
					Tai:          checkedOut.GetSkuExternal().GetTai().String(),
					LocalDigest:  existing.GetObjectDigest().String(),
					RemoteDigest: checkedOut.GetSkuExternal().GetObjectDigest().String(),
				}
				return checkedOut, err
			}

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
		if importer.parentNegotiator == nil {
			// Without a parent negotiator, 3-way merge uses an empty base and
			// always produces false conflicts. Accept the remote version directly.
			if err = importer.importNewObject(
				checkedOut.GetSkuExternal(),
			); err != nil {
				err = errors.Wrap(err)
				return checkedOut, err
			}

			return checkedOut, err
		}

		if commitOptions, err = importer.storeExternal.MergeCheckedOut(
			checkedOut,
			importer.parentNegotiator,
			importer.allowMergeConflicts,
		); err != nil {
			if checkout_mode.IsErrInvalidCheckoutMode(err) {
				err = ErrCrossPubKeyMerge{
					ObjectId:     checkedOut.GetSkuExternal().GetObjectId().String(),
					LocalPubKey:  checkedOut.GetSku().GetMetadata().GetRepoPubKey().String(),
					RemotePubKey: checkedOut.GetSkuExternal().GetMetadata().GetRepoPubKey().String(),
				}
				return checkedOut, err
			}

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

	// https://github.com/amarbel-llc/dodder/issues/325
	// An object's typed blob references are part of its closure: copy each
	// referenced content blob alongside the object's own blob. This is the
	// receive-side guarantee for transfers whose list contains inventory
	// list objects (the default clone/pull query): expand-edges never
	// traverses into a list's contained objects, so a reference from a
	// contained object would otherwise be silently dropped. Gated on a
	// remote blob store being present: drtp pre-streams its whole manifest
	// (no remote store on the receiver), and the HTTP push server cannot
	// fetch from the pushing client. A reference blob the remote does not
	// hold is skipped, matching the edges copy loop in local_op_pull.
	if importer.remoteBlobStore != nil {
		for blobDigest := range object.GetMetadata().AllBlobReferences() {
			blobCopy := blobDigest

			refResult := importer.blobImporter.ImportBlobToStoreIfNecessary(
				importer.envRepo.GetDefaultBlobStore(),
				&blobCopy,
				object,
			)

			if refErr := refResult.GetError(); refErr != nil {
				if errors.IsErrNotFound(refErr) {
					continue
				}

				err = errors.Wrapf(
					refErr,
					"blob reference %s",
					blobCopy.String(),
				)
				return err
			}
		}
	}

	return err
}
