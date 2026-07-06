package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/lib/bravo/script_value"
	"github.com/amarbel-llc/madder/go/pkgs/fd"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type CreateFromPaths struct {
	*repo
	Proto      sku.Proto
	TextParser object_metadata_fmt_hyphence.Parser
	Filter     script_value.ScriptValue
	Delete     bool
	// ReadHinweisFromPath bool
}

func (op CreateFromPaths) Run(
	args ...string,
) (results sku.TransactedMutableSet, err error) {
	toCreate := make(map[string]*sku.Transacted)
	toDelete := fd.MakeMutableSet()

	digestWithoutTai, digestWithoutTaiRepool := markl.GetId()
	defer digestWithoutTaiRepool()

	for _, arg := range args {
		var object *sku.Transacted
		var fsItem sku.FSItem

		fsItem.Reset()

		fsItem.GetExternalObjectId().SetGenre(genres.Zettel)

		if err = fsItem.Object.Set(arg); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if err = fsItem.FDs.Add(&fsItem.Object); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if object, err = op.GetEnvWorkspace().GetStoreFS().ReadExternalFromItem(
			sku.CommitOptions{
				StoreOptions: sku.GetStoreOptionsRealizeWithProto(),
			},
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

	builder := import_plan.MakeLocalBuilder()

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(
			op.GetStore().GetZettelIdIndex(),
		),
	)

	for _, object := range toCreate {
		if object.GetMetadata().IsEmpty() {
			continue
		}

		op.Proto.Apply(object, genres.Zettel)

		if err = builder.AddObject(object, 0); err != nil {
			err = errors.Wrap(err)
			return results, err
		}
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return results, err
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto: op.GetStore().GetProtoZettel(),
		StoreOptions: sku.StoreOptions{
			LockfileOptions: sku.LockfileOptions{
				AllowTagFailure: true,
			},
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			ApplyProto:         true,
		},
	}

	results, err = op.repo.ExecutePlan(plan)
	if err != nil {
		return results, err
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

	return results, err
}
