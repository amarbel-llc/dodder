package user_ops

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/delta/script_value"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/src/india/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
)

type CreateFromPaths struct {
	*local_working_copy.Repo
	Proto      sku.Proto
	TextParser object_metadata_fmt_triple_hyphen.Parser
	Filter     script_value.ScriptValue
	Delete     bool
	// ReadHinweisFromPath bool
}

func (op CreateFromPaths) Run(
	args ...string,
) (results sku.TransactedMutableSet, err error) {
	toCreate := make(map[string]*sku.Transacted)
	toDelete := fd.MakeMutableSet()

	commitOptions := sku.CommitOptions{
		StoreOptions: sku.GetStoreOptionsRealizeWithProto(),
	}

	digestWithoutTai, digestWithoutTaiRepool := markl.GetId()
	defer digestWithoutTaiRepool()

	for _, arg := range args {
		var object *sku.Transacted
		var fsItem sku.FSItem

		fsItem.Reset()

		fsItem.ExternalObjectId.SetGenre(genres.Zettel)

		if err = fsItem.Object.Set(arg); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if err = fsItem.FDs.Add(&fsItem.Object); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if object, err = op.GetEnvWorkspace().GetStoreFS().ReadExternalFromItem(
			commitOptions,
			&fsItem,
			nil,
		); err != nil {
			err = errors.Wrapf(
				err,
				"zettel text format error for path: %s",
				arg,
			)
			return results, err
		}

		if err = object.CalculateDigestForPurpose(
			markl.PurposeV5MetadataDigestWithoutTai,
			digestWithoutTai,
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if err = markl.AssertIdIsNotNull(
			digestWithoutTai,
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		digestBytes := digestWithoutTai.GetBytes()
		existing, ok := toCreate[string(digestBytes)]

		if ok {
			if err = existing.GetMetadataMutable().GetDescriptionMutable().Set(
				object.GetMetadata().GetDescription().String(),
			); err != nil {
				err = errors.Wrap(err)
				return results, err
			}
		} else {
			toCreate[string(digestBytes)] = object
		}

		if op.Delete {
			{
				var fdObject *fd.FD

				if fdObject, err = op.GetEnvWorkspace().GetStoreFS().GetObjectOrError(object); err != nil {
					err = errors.Wrap(err)
					return results, err
				}

				toDelete.Add(fdObject)
			}

			{
				var fdBlob *fd.FD

				if fdBlob, err = op.GetEnvWorkspace().GetStoreFS().GetObjectOrError(object); err != nil {
					err = errors.Wrap(err)
					return results, err
				}

				toDelete.Add(fdBlob)
			}
		}
	}

	results = sku.MakeTransactedMutableSet()

	if err = op.Lock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	for _, object := range toCreate {
		if object.GetMetadata().IsEmpty() {
			return results, err
		}

		op.Proto.Apply(object, genres.Zettel)

		if err = op.GetStore().CreateOrUpdateDefaultProto(
			object,
			sku.StoreOptions{
				LockfileOptions: sku.LockfileOptions{
					AllowTagFailure: true,
				},
				ApplyProto: true,
			},
		); err != nil {
			// TODO-P2 add file for error handling
			err = errors.Wrap(err)
			return results, err
		}

		results.Add(object)
	}

	for fdToDelete := range toDelete.All() {
		// TODO-P2 move to checkout store
		if err = op.GetEnvRepo().Delete(fdToDelete.GetPath()); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		pathRel := op.GetEnvRepo().RelToCwdOrSame(fdToDelete.GetPath())

		// TODO-P2 move to printer
		op.GetUI().Printf("[%s] (deleted)", pathRel)
	}

	if err = op.Unlock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}
