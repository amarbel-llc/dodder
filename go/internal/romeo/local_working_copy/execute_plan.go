package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func (local *Repo) ExecutePlan(
	plan *import_plan.Plan,
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

		options := plan.DefaultCommitOptions

		if entry.Options != nil {
			options = *entry.Options
		}

		object := entry.GetObject()

		if err = local.GetStore().Commit(object, options); err != nil {
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
