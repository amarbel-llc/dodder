package remote_transfer

import (
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/tango/env_box"
	"code.linenisgreat.com/dodder/go/internal/uniform/repo"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

// TODO create an open list and resolve the graph as necessary
func (importer importer) ImportSeq(
	ctx interfaces.ActiveContext,
	local repo.LocalRepo,
	envBox env_box.Env,
	seq interfaces.SeqError[*sku.Transacted],
) (err error) {
	ctx.Must(errors.MakeFuncContextFromFuncErr(local.Lock))

	var hasConflicts bool

	checkedOutPrinter := importer.GetCheckedOutPrinter()

	importer.SetCheckedOutPrinter(
		func(checkedOut *sku.CheckedOut) (err error) {
			if checkedOut.GetState() == checked_out_state.Conflicted {
				hasConflicts = true
			}

			return checkedOutPrinter(checkedOut)
		},
	)

	importErrors := errors.MakeGroupBuilder()
	missingBlobs := sku.MakeListCheckedOut()

	for object, iterErr := range seq {
		if iterErr != nil {
			err = errors.Wrap(iterErr)
			return err
		}

		var hasOneConflict bool

		if hasOneConflict, err = importer.importOne(
			local,
			object,
			missingBlobs,
		); err != nil {
			err = errors.Wrapf(err, "Object: %s", sku.String(object))
			importErrors.Add(err)
			err = nil
		}

		if hasOneConflict {
			hasConflicts = true
		}
	}

	checkedOutPrinter = envBox.GetUIStorePrinters().CheckedOut

	if missingBlobs.Len() > 0 {
		ui.Err().Printf(
			"could not import %d objects (blobs missing):\n",
			missingBlobs.Len(),
		)

		for missing := range missingBlobs.All() {
			if err = checkedOutPrinter(missing); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	if hasConflicts {
		importErrors.Add(ErrNeedsMerge)
	}

	if importErrors.Len() > 0 {
		err = importErrors.GetError()
	}

	ctx.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))

	return err
}

func (importer importer) importOne(
	repo repo.LocalRepo,
	object *sku.Transacted,
	missingBlobs *sku.HeapCheckedOut,
) (hasConflicts bool, err error) {
	var checkedOut *sku.CheckedOut
	checkedOut, err = importer.Import(object)
	// checkedOut lifecycle managed by caller

	if err == nil {
		if checkedOut.GetState() == checked_out_state.Conflicted {
			hasConflicts = true
		}

		return hasConflicts, err
	}

	if errors.Is(err, ErrSkipped) {
		err = nil
		return hasConflicts, err
	} else if errors.Is(err, errors.ErrExists) {
		err = nil
		return hasConflicts, err
	} else if genres.IsErrUnsupportedGenre(err) {
		err = nil
		return hasConflicts, err
	} else if env_dir.IsErrBlobMissing(err) {
		checkedOut, _ := sku.GetCheckedOutPool().GetWithRepool()
		sku.TransactedResetter.ResetWith(
			checkedOut.GetSkuExternal(),
			object,
		)
		checkedOut.SetState(checked_out_state.Untracked)

		missingBlobs.Add(checkedOut)

		return hasConflicts, err
	}

	return hasConflicts, err
}
