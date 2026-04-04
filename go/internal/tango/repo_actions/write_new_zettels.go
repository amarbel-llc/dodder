package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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
