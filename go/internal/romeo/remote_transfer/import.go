package remote_transfer

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/echo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/env_box"
	"code.linenisgreat.com/dodder/go/internal/quebec/repo"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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
	var dedupCount int

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

	var errorLog *importErrorLog

	if importer.continueOnError {
		errorLog = &importErrorLog{}
	}

	for object, iterErr := range seq {
		if iterErr != nil {
			err = errors.Wrap(iterErr)

			if errorLog != nil {
				errorLog.Close()
			}

			return err
		}

		var hasOneConflict bool

		if hasOneConflict, err = importer.importOne(
			local,
			object,
			missingBlobs,
			&dedupCount,
		); err != nil {
			wrappedErr := errors.Wrapf(err, "Object: %s", sku.String(object))

			if importer.continueOnError {
				ui.Err().Print(wrappedErr)

				if logErr := errorLog.LogError(object, err); logErr != nil {
					ui.Err().Printf("failed to write error log: %s", logErr)
				}
			} else {
				importErrors.Add(wrappedErr)
			}

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

				if errorLog != nil {
					errorLog.Close()
				}

				return err
			}
		}
	}

	if dedupCount > 0 {
		ui.Err().Printf("%d objects deduped during import\n", dedupCount)
	}

	if hasConflicts {
		importErrors.Add(ErrNeedsMerge)
	}

	if errorLog != nil {
		if closeErr := errorLog.Close(); closeErr != nil {
			ui.Err().Printf("failed to close error log: %s", closeErr)
		}

		if errorLog.Count() > 0 {
			importErrors.Add(errors.WithHelp(
				fmt.Errorf("%d objects failed to import", errorLog.Count()),
				[]string{"One or more objects encountered errors during import"},
				[]string{
					fmt.Sprintf(
						"Review error log: %s",
						errorLog.Path(),
					),
				},
			))
		}
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
	dedupCount *int,
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
	} else if IsErrDeduped(err) {
		*dedupCount++
		err = nil
		return hasConflicts, err
	} else if env_dir.IsErrBlobMissing(err) {
		checkedOut, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned
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
