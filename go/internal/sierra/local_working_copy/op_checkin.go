package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
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

	// Phase 1: pre-process untracked objects and allocate zettel IDs before
	// acquiring lock
	zettelIdIndex := local.GetStore().GetZettelIdIndex()

	for _, co := range sortedResults {
		if co.GetState() != checked_out_state.Untracked {
			continue
		}

		external := co.GetSkuExternal()

		if external.GetGenre() != genres.Zettel && external.GetGenre() != genres.Blob {
			continue
		}

		if external.GetMetadata().IsEmpty() {
			continue
		}

		external.GetObjectIdMutable().Reset()

		zettelId, idErr := zettelIdIndex.CreateZettelId()
		if idErr != nil {
			err = errors.Wrap(idErr)
			return processed, err
		}

		if err = external.GetObjectIdMutable().SetWithSeq(zettelId.ToSeq()); err != nil {
			err = errors.Wrap(err)
			return processed, err
		}
	}

	// Phase 2: commit all objects under lock
	local.Must(errors.MakeFuncContextFromFuncErr(local.Lock))

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

		cloned, _ := co.GetSkuExternal().CloneTransacted() //repool:owned
		if err = processed.Add(cloned); err != nil {
			err = errors.Wrap(err)
			return processed, err
		}
	}

	local.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))

	return processed, err
}
