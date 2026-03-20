package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/internal/november/env_workspace"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func (local *Repo) Checkin(
	skus sku.SkuTypeSetMutable,
	proto sku.Proto,
	delete bool,
	refreshCheckout bool,
) (processed sku.TransactedMutableSet, err error) {
	processed = sku.MakeTransactedMutableSet()
	sortedResults := quiter.ElementsSorted(
		skus,
		func(left, right sku.SkuType) bool {
			return left.String() < right.String()
		},
	)

	// Side map for post-plan correlation: object ID string -> CheckedOut
	checkedOutByObjectId := make(map[string]sku.SkuType)

	// Pre-plan phase (no lock)

	zettelIdIndex := local.GetStore().GetZettelIdIndex()

	builder := import_plan.MakeBuilder(
		local.GetStore().GetStreamIndex(),
		"",
	)

	for _, co := range sortedResults {
		if refreshCheckout {
			if err = local.GetEnvWorkspace().GetStoreFS().RefreshCheckedOut(
				co,
			); err != nil {
				err = errors.Wrap(err)
				return processed, err
			}
		}

		external := co.GetSkuExternal()

		if co.GetState() == checked_out_state.Untracked &&
			(external.GetGenre() == genres.Zettel ||
				external.GetGenre() == genres.Blob) {
			if external.GetMetadata().IsEmpty() {
				continue
			}

			external.GetObjectIdMutable().Reset()

			zettelId, idErr := zettelIdIndex.CreateZettelId()
			if idErr != nil {
				err = errors.Wrap(idErr)
				return processed, err
			}

			if err = external.GetObjectIdMutable().SetWithSeq(
				zettelId.ToSeq(),
			); err != nil {
				err = errors.Wrap(err)
				return processed, err
			}

			if err = local.GetStore().UpdateTransactedFromBlobs(
				co,
			); err != nil {
				if errors.Is(err, env_workspace.ErrUnsupportedOperation{}) {
					err = nil
				} else {
					err = errors.Wrap(err)
					return processed, err
				}
			}

			proto.Apply(external, genres.Zettel)

			untrackedOptions := sku.CommitOptions{
				Proto: proto,
				StoreOptions: sku.GetStoreOptionsCreate(),
			}

			builder.AddObject(external, 0)

			// Set per-entry options after AddObject appends the entry
			entries := builder.PeekEntries()
			entries[len(entries)-1].Options = &untrackedOptions
		} else {
			trackedOptions := sku.CommitOptions{
				StoreOptions: sku.GetStoreOptionsCreate(),
			}

			builder.AddObject(external, 0)

			entries := builder.PeekEntries()
			entries[len(entries)-1].Options = &trackedOptions

			// Track for post-plan checkout update
			checkedOutByObjectId[external.GetObjectId().String()] = co
		}
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return processed, err
	}

	// Execute phase
	results, execErr := local.ExecutePlan(plan)
	if execErr != nil {
		err = errors.Wrap(execErr)
		return processed, err
	}

	// Post-plan phase (no lock): checkout updates and deletes

	for committedObject := range results.All() {
		objectIdStr := committedObject.GetObjectId().String()

		if co, ok := checkedOutByObjectId[objectIdStr]; ok && !delete {
			if err = local.GetStore().UpdateCheckoutFromCheckedOut(
				checkout_options.OptionsWithoutMode{Force: true},
				co,
			); err != nil {
				err = errors.Wrap(err)
				return processed, err
			}
		}

		if delete {
			// Find the original CheckedOut for deletion
			for _, co := range sortedResults {
				if co.GetSkuExternal().GetObjectId().String() == objectIdStr {
					if err = local.GetStore().DeleteCheckedOut(co); err != nil {
						err = errors.Wrap(err)
						return processed, err
					}

					cloned, _ := co.GetSkuExternal().CloneTransacted() //repool:owned
					if err = processed.Add(cloned); err != nil {
						err = errors.Wrap(err)
						return processed, err
					}

					break
				}
			}
		}
	}

	return processed, err
}
