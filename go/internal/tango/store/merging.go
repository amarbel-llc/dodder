package store

import (
	"code.linenisgreat.com/dodder/go/internal/delta/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/hotel/objects"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func (store *Store) ReadExternalAndMergeIfNecessary(
	left, mother *sku.Transacted,
	options sku.CommitOptions,
) (err error) {
	if mother == nil {
		return err
	}

	var checkedOut *sku.CheckedOut

	if checkedOut, err = store.ReadCheckedOutFromTransacted(
		options.RepoId,
		mother,
	); err != nil {
		if errors.IsNotExist(err) || errors.IsErrNotFound(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	defer store.PutCheckedOutLike(checkedOut)

	right := checkedOut.GetSkuExternal().GetSku()

	// TODO switch to using mother
	motherEqualsExternal := objects.EqualerSansTai.Equals(
		right.GetMetadata(),
		checkedOut.GetSku().GetMetadata(),
	)

	if motherEqualsExternal {
		checkoutOptions := checkout_options.OptionsWithoutMode{
			Force: true,
		}

		sku.TransactedResetter.ResetWithExceptFields(right, left)

		if err = store.UpdateCheckoutFromCheckedOut(
			checkoutOptions,
			checkedOut,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	if err = right.SetMother(mother); err != nil {
		err = errors.Wrap(err)
		return err
	}

	conflicted := sku.Conflicted{
		CheckedOut: checkedOut,
		Local:      left,
		Base:       mother,
		Remote:     right,
	}

	if err = store.MergeConflicted(conflicted); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
