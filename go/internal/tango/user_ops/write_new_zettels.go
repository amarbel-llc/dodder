package user_ops

import (
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type WriteNewZettels struct {
	*local_working_copy.Repo
}

func (op WriteNewZettels) RunMany(
	proto sku.Proto,
	count int,
) (results sku.TransactedMutableSet, err error) {
	zettelIdIndex := op.GetStore().GetZettelIdIndex()

	builder := import_plan.MakeBuilder(
		op.GetStore().GetStreamIndex(),
		"",
	)

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(zettelIdIndex),
	)

	for range count {
		object, _ := proto.Make() //repool:owned

		builder.AddObject(object, 0)
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return results, err
	}

	results, err = CommitPlan(
		op.Repo,
		plan,
		sku.StoreOptions{ApplyProto: true},
	)

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

