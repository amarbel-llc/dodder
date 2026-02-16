package store

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/expansion"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/charlie/store_version"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_provider"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/hotel/file_lock"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

func (store *Store) Commit(
	daughter *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	committer := commitFacilitator{
		Store: store,
		index: store.streamIndex,
	}

	if err = committer.commit(daughter, options); err != nil {
		err = errors.Wrapf(err, "Sku: %q", sku.String(daughter))
		return err
	}

	return err
}

// TODO move to object_finalizer
type commitFacilitator struct {
	*Store
	index sku.Reindexer
}

// Saves the blob if necessary, applies the proto object, runs pre-commit hooks,
// runs the new hook, validates the blob, then calculates the digest for the
// object
func (commitFacilitator commitFacilitator) tryPrecommit(
	daughter *sku.Transacted,
	mother *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if err = commitFacilitator.SaveBlob(daughter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if mother == nil {
		options.Proto.Apply(daughter, daughter)
	}

	// TODO decide if the type proto should actually be applied every time
	if options.ApplyProtoType {
		commitFacilitator.protoZettel.ApplyType(daughter, daughter)
	}

	if genres.Type == daughter.GetGenre() {
		if daughter.GetType().IsEmpty() {
			daughter.GetMetadataMutable().GetTypeMutable().ResetWithObjectId(
				ids.DefaultOrPanic(genres.Type),
			)
		}
	}

	// modify pre commit hooks to support import
	if err = commitFacilitator.tryPreCommitHooks(daughter, mother, options); err != nil {
		if commitFacilitator.storeConfig.GetConfig().IgnoreHookErrors {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	// TODO just just mutter == nil
	if mother == nil {
		if err = commitFacilitator.tryNewHook(daughter, options); err != nil {
			if commitFacilitator.storeConfig.GetConfig().IgnoreHookErrors {
				err = nil
			} else {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	return err
}

// TODO add RealizeAndOrStore result
// TODO switch to using a child context for each object commit
func (commitFacilitator commitFacilitator) commit(
	daughter *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if daughter == nil {
		panic("empty daughter")
	}

	ui.Log().Printf("%s -> %s", options, daughter)

	// TODO remove this lock check and perform it when actually necessary (when
	// persisting the changes on flush).
	if !commitFacilitator.GetEnvRepo().GetLockSmith().IsAcquired() &&
		(options.AddToInventoryList || options.StreamIndexOptions.AddToStreamIndex) {
		err = errors.Wrap(file_lock.ErrLockRequired{
			Operation: "commit",
		})

		return err
	}

	// TAI must be set before calculating object sha
	if options.UpdateTai {
		if options.Clock == nil {
			options.Clock = commitFacilitator
		}

		tai := options.Clock.GetTai()
		daughter.SetTai(tai)
	}

	if options.AddToInventoryList && (daughter.ObjectId.IsEmpty() ||
		daughter.GetGenre() == genres.Unknown ||
		daughter.GetGenre() == genres.Blob) {
		var zettelId *ids.ZettelId

		if zettelId, err = commitFacilitator.zettelIdIndex.CreateZettelId(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = daughter.ObjectId.SetWithSeq(zettelId.ToSeq()); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	var mother *sku.Transacted

	if mother, err = commitFacilitator.fetchMotherIfNecessary(
		daughter,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// mother lifecycle managed by fetchMotherIfNecessary caller

	if err = commitFacilitator.tryPrecommit(daughter, mother, options); err != nil {
		err = errors.Wrap(err)
		return err
	}

	{
		if options.AddToInventoryList {
			if err = commitFacilitator.addMissingTypes(
				options,
				daughter,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if err = commitFacilitator.validateAndFinalize(
			daughter,
			mother,
			options,
		); err != nil {
			err = errors.Wrapf(err, "failed to validate object: %q", sku.String(daughter))
			return err
		}

		if options.AddToInventoryList ||
			options.StreamIndexOptions.AddToStreamIndex {
			if err = commitFacilitator.GetAbbrStore().AddObjectToIdIndex(
				daughter,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		// short circuits if the parent is equal to the child
		if options.AddToInventoryList &&
			mother != nil &&
			ids.Equals(daughter.GetObjectId(), mother.GetObjectId()) &&
			objects.EqualerSansTai.Equals(daughter.GetMetadata(), mother.GetMetadata()) {

			sku.TransactedResetter.ResetWithExceptFields(daughter, mother)

			// TODO why is this condition here
			if commitFacilitator.sunrise.Less(daughter.GetTai()) {
				if err = commitFacilitator.handleUnchanged(daughter); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			return err
		}

		// TODO eventually remove when moving to new model of dormancy and expanded
		// tags
		if err = commitFacilitator.applyDormantAndRealizeTags(
			daughter,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if daughter.GetGenre() == genres.Zettel {
			if err = commitFacilitator.zettelIdIndex.AddZettelId(&daughter.ObjectId); err != nil {
				if errors.Is(err, object_id_provider.ErrDoesNotExist{}) {
					ui.Log().Printf("object id does not contain value: %s", err)
					err = nil
				} else {
					err = errors.Wrapf(err, "failed to write zettel to index: %s", daughter)
					return err
				}
			}
		}
	}

	if options.AddToInventoryList {
		// external.GetSku().Metadata.GetObjectSigMutable().Reset()
		daughter.GetMetadataMutable().GetObjectDigestMutable().Reset()

		if err = commitFacilitator.commitTransacted(daughter, mother); err != nil {
			err = errors.Wrapf(err, "Sku: %s", sku.String(daughter))
			return err
		}
	}

	if options.AddToInventoryList ||
		options.StreamIndexOptions.AddToStreamIndex {
		if store_version.GreaterOrEqual(
			commitFacilitator.storeConfig.GetConfig().GetStoreVersion(),
			store_version.V11,
		) {
			if err = markl.AssertIdIsNotNull(
				daughter.GetMetadata().GetObjectSig()); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if err = commitFacilitator.storeConfig.AddTransacted(
			daughter,
			mother,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = commitFacilitator.index.Add(
			daughter,
			options,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if mother == nil {
			if daughter.GetGenre() == genres.Zettel {
				// TODO if this is a local zettel (i.e., not a different repo
				// and not a
				// different domain)

				// TODO verify that the zettel id consists of our identifiers,
				// otherwise
				// abort
			}

			if err = commitFacilitator.ui.TransactedNew(daughter); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else {
			// [are/kabuto !task project-2021-zit-features zz-inbox] add delta
			// printing to changed objects
			// if err = s.Updated(mutter); err != nil {
			// 	err = errors.Wrap(err)
			// 	return
			// }

			if err = commitFacilitator.ui.TransactedUpdated(daughter); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

	}

	if options.MergeCheckedOut {
		if err = commitFacilitator.ReadExternalAndMergeIfNecessary(
			daughter,
			mother,
			options,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (commitFacilitator commitFacilitator) fetchMotherIfNecessary(
	daughter *sku.Transacted,
) (mother *sku.Transacted, err error) {
	if daughter == nil {
		panic("empty daughter")
	}

	objectId := daughter.GetObjectId()

	if objectId.IsEmpty() {
		return mother, err
	}

	var motherRepool interfaces.FuncRepool
	mother, motherRepool = sku.GetTransactedPool().GetWithRepool()

	// TODO find a way to make this more performant when operating over sshfs
	if !sku.ReadOneObjectId(
		commitFacilitator.index,
		objectId,
		mother,
	) {
		motherRepool()
		mother = nil
		return mother, err
	}

	if err = daughter.SetMother(mother); err != nil {
		err = errors.Wrap(err)
		return mother, err
	}

	return mother, err
}

// TODO add results for which stores had which change types
func (commitFacilitator commitFacilitator) commitTransacted(
	daughter *sku.Transacted,
	mother *sku.Transacted,
) (err error) {
	if err = commitFacilitator.workingList.Add(daughter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (commitFacilitator commitFacilitator) handleUnchanged(
	object *sku.Transacted,
) (err error) {
	return commitFacilitator.ui.TransactedUnchanged(object)
}

func (commitFacilitator commitFacilitator) createType(
	typeId *ids.ObjectId,
) (err error) {
	typeObject, typeObjectRepool := sku.GetTransactedPool().GetWithRepool()
	defer typeObjectRepool()

	switch typeId.GetGenre() {
	default:
		err = genres.MakeErrUnsupportedGenre(typeId.GetGenre())
		return err

	case genres.Type:
		typeObject.GetMetadataMutable().GetTypeMutable().ResetWithObjectId(
			ids.DefaultOrPanic(genres.Type),
		)
	}

	if err = typeObject.ObjectId.SetWithId(typeId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = commitFacilitator.Commit(
		typeObject,
		sku.CommitOptions{
			StoreOptions: sku.GetStoreOptionsUpdate(),
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (commitFacilitator commitFacilitator) addTypeIfNecessary(
	typeId ids.Type,
) (err error) {
	if typeId.IsEmpty() {
		err = errors.ErrorWithStackf("attempting to add empty type")
		return err
	}

	var objectId ids.ObjectId

	if err = ids.SetObjectIdOrBlob(&objectId, typeId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = commitFacilitator.index.ObjectExists(&objectId); err == nil {
		return err
	}

	err = nil

	var typeObjectId ids.ObjectId

	if err = ids.SetObjectIdOrBlob(
		&typeObjectId,
		typeId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = commitFacilitator.createType(&typeObjectId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (commitFacilitator commitFacilitator) addTypeAndExpandedIfNecessary(
	rootType ids.Type,
) (err error) {
	if rootType.IsEmpty() {
		return err
	}

	if ids.IsBuiltin(rootType) {
		return err
	}

	typesExpanded := expansion.ExpandOneIntoIds[ids.SeqId](
		rootType.String(),
		expansion.ExpanderRight,
	)

	for tipe := range typesExpanded {
		if err = commitFacilitator.addTypeIfNecessary(tipe); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (commitFacilitator commitFacilitator) addMissingTypes(
	commitOptions sku.CommitOptions,
	object *sku.Transacted,
) (err error) {
	tipe := object.GetType()

	if !commitOptions.DontAddMissingType {
		if err = commitFacilitator.addTypeAndExpandedIfNecessary(tipe); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		// TODO enforce that object has signature
	}

	return err
}
