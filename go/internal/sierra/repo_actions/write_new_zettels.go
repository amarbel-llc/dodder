package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type WriteNewZettels struct {
	*repo
}

func (op WriteNewZettels) RunMany(
	proto sku.Proto,
	count int,
) (results sku.TransactedMutableSet, err error) {
	zettelIdIndex := op.GetStore().GetZettelIdIndex()

	builder := import_plan.MakeLocalBuilder()

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(zettelIdIndex),
	)

	for range count {
		object, _ := proto.Make() //repool:owned

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
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			ApplyProto:         true,
		},
	}

	results, err = op.repo.ExecutePlan(plan)

	return results, err
}

// RunOneWithObjectId creates exactly one object, honoring a caller-chosen
// object-id and/or an inline blob body. An empty objectId falls through to the
// auto-assigned-zettel-id path (the AllocateZettelId transform skips non-empty
// ids, so it only fires when objectId is empty). When the chosen id's genre is
// not Zettel (e.g. a type `!task` or a tag), the object's meta-type is reset to
// that genre's default builtin type (`!toml-type-v2` / `!toml-tag-v1`), since
// proto.Make stamps the workspace zettel default and commit-time defaulting only
// fills an empty type. An empty blob writes no blob body.
func (op WriteNewZettels) RunOneWithObjectId(
	proto sku.Proto,
	objectId *ids.ObjectId,
	blob string,
) (result *sku.Transacted, err error) {
	zettelIdIndex := op.GetStore().GetZettelIdIndex()

	builder := import_plan.MakeLocalBuilder()

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(zettelIdIndex),
	)

	object, _ := proto.Make() //repool:owned

	if objectId != nil && !objectId.IsEmpty() {
		if err = object.GetObjectIdMutable().SetWithId(objectId); err != nil {
			err = errors.Wrap(err)
			return result, err
		}

		if genre := genres.Make(object.GetGenre()); genre != genres.Zettel {
			object.GetMetadataMutable().GetTypeMutable().ResetWithType(
				ids.DefaultOrPanic(genre),
			)
		}
	}

	if blob != "" {
		if err = writeBlobContent(op.repo, object, blob); err != nil {
			err = errors.Wrap(err)
			return result, err
		}
	}

	if err = builder.AddObject(object, 0); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return result, err
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto: op.GetStore().GetProtoZettel(),
		StoreOptions: sku.StoreOptions{
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			ApplyProto:         true,
		},
	}

	results, err := op.repo.ExecutePlan(plan)
	if err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	for t := range results.All() {
		result = t
		break
	}

	return result, err
}

func (op WriteNewZettels) RunOne(
	z sku.Proto,
) (result *sku.Transacted, err error) {
	results, err := op.RunMany(z, 1)
	if err != nil {
		return result, err
	}

	for t := range results.All() {
		result = t
		break
	}

	return result, err
}
