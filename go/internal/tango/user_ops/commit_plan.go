package user_ops

import (
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func CommitPlan(
	local *local_working_copy.Repo,
	plan *import_plan.Plan,
	storeOptions sku.StoreOptions,
) (results sku.TransactedMutableSet, err error) {
	if err = local.Lock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	results = sku.MakeTransactedMutableSet()

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if !entry.Classification.IsCommittable() {
			continue
		}

		object := entry.GetObject()

		if err = local.GetStore().CreateOrUpdateDefaultProto(
			object,
			storeOptions,
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if err = results.Add(object); err != nil {
			err = errors.Wrap(err)
			return results, err
		}
	}

	if err = local.Unlock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}
