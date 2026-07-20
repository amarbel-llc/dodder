package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type CreateFromShas struct {
	*repo
	sku.Proto
}

func (op CreateFromShas) Run(
	args ...string,
) (results sku.TransactedMutableSet, err error) {
	var lookupStored map[string][]string

	if lookupStored, err = op.GetStore().MakeBlobDigestObjectIdsMap(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	toCreate := make(map[string]*sku.Transacted)

	for _, arg := range args {
		var digest markl.Id

		if err = markl.SetMaybeSha256(
			&digest,
			arg,
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		digestBytes := digest.GetBytes()

		if _, ok := toCreate[string(digestBytes)]; ok {
			ui.Err().Printf(
				"%s appears in arguments more than once. Ignoring",
				&digest,
			)
			continue
		}

		if oids, ok := lookupStored[string(digestBytes)]; ok {
			ui.Err().Printf(
				"%s appears in object already checked in (%q). Ignoring",
				&digest,
				oids,
			)
			continue
		}

		object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

		object.GetObjectIdMutable().SetGenre(genres.Zettel)
		object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(&digest)

		op.Proto.Apply(object, genres.Zettel)

		toCreate[string(digestBytes)] = object
	}

	builder := import_plan.MakeLocalBuilder()

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(
			op.GetStore().GetZettelIdIndex(),
		),
	)

	for _, object := range toCreate {
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
