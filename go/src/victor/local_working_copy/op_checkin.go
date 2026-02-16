package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/quebec/env_workspace"
)

func (local *Repo) Checkin(
	skus sku.SkuTypeSetMutable,
	proto sku.Proto,
	delete bool,
	refreshCheckout bool,
) (processed sku.TransactedMutableSet, err error) {
	local.Must(errors.MakeFuncContextFromFuncErr(local.Lock))

	processed = sku.MakeTransactedMutableSet()
	sortedResults := quiter.ElementsSorted(
		skus,
		func(left, right sku.SkuType) bool {
			return left.String() < right.String()
		},
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
			(co.GetSkuExternal().GetGenre() == genres.Zettel ||
				co.GetSkuExternal().GetGenre() == genres.Blob) {
			if external.GetMetadata().IsEmpty() {
				continue
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

			external.ObjectId.Reset()

			proto.Apply(external, genres.Zettel)

			if err = local.GetStore().CreateOrUpdate(
				external,
				sku.CommitOptions{
					Proto: proto,
				},
			); err != nil {
				err = errors.Wrap(err)
				return processed, err
			}
		} else {
			if err = local.GetStore().CreateOrUpdateCheckedOut(
				co,
				!delete,
			); err != nil {
				err = errors.Wrapf(err, "CheckedOut: %s", co)
				return processed, err
			}
		}

		if !delete {
			continue
		}

		if err = local.GetStore().DeleteCheckedOut(co); err != nil {
			err = errors.Wrap(err)
			return processed, err
		}

		cloned, _ := co.GetSkuExternal().CloneTransacted()
		if err = processed.Add(cloned); err != nil {
			err = errors.Wrap(err)
			return processed, err
		}
	}

	local.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))

	return processed, err
}
