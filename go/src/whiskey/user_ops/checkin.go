package user_ops

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/oscar/organize_text"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
)

type Checkin struct {
	Proto sku.Proto

	// TODO make flag family disambiguate these options
	// and use with other commands too
	Delete             bool
	RefreshCheckout    bool
	Organize           bool
	CheckoutBlobAndRun string
	OpenBlob           bool
	Edit               bool // TODO add support back for this
}

func (op Checkin) Run(
	repo *local_working_copy.Repo,
	query *queries.Query,
) (err error) {
	var lock sync.Mutex

	results := sku.MakeSkuTypeSetMutable()

	if err = repo.GetStore().QuerySkuType(
		query,
		func(co sku.SkuType) (err error) {
			lock.Lock()
			defer lock.Unlock()

			cloned, _ := co.Clone()
			return results.Add(cloned)
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if op.Organize {
		if err = op.runOrganize(repo, query, results); err != nil {
			err = errors.Wrap(err)
			return err
		}

		objects.Resetter.Reset(&op.Proto.Metadata)
	}

	var processed sku.TransactedMutableSet

	if processed, err = repo.Checkin(
		results,
		op.Proto,
		op.Delete,
		op.RefreshCheckout,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = op.openBlobIfNecessary(repo, processed); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (op Checkin) runOrganize(
	repo *local_working_copy.Repo,
	query *queries.Query,
	results sku.SkuTypeSetMutable,
) (err error) {
	flagDelete := organize_text.OptionCommentBooleanFlag{
		Value:   &op.Delete,
		Comment: "delete once checked in",
	}

	opOrganize := Organize2{
		Repo: repo,
		Metadata: organize_text.Metadata{
			TagSet: op.Proto.Metadata.GetTags(),
			Type:   op.Proto.Metadata.GetType().ToType(),
			RepoId: query.RepoId,
			OptionCommentSet: organize_text.MakeOptionCommentSet(
				map[string]organize_text.OptionComment{
					"delete": flagDelete,
				},
				&organize_text.OptionCommentUnknown{
					Value: "instructions: to prevent an object from being checked in, delete it entirely",
				},
				organize_text.OptionCommentWithKey{
					Key:           "delete",
					OptionComment: flagDelete,
				},
			),
		},
	}

	ui.Log().Print(query)

	var organizeResults organize_text.OrganizeResults

	if organizeResults, err = opOrganize.Run(results); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var changes organize_text.Changes

	if changes, err = organize_text.ChangesFromResults(
		repo.GetConfig().GetPrintOptions(),
		organizeResults,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, co := range changes.After.AllSkuAndIndex() {
		clonedCo, _ := co.Clone()
		if err = results.Add(clonedCo); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for _, co := range changes.Removed.AllSkuAndIndex() {
		quiter_set.Del(results, co)
	}

	return err
}

func (c Checkin) openBlobIfNecessary(
	repo *local_working_copy.Repo,
	objects sku.TransactedSet,
) (err error) {
	if !c.OpenBlob && c.CheckoutBlobAndRun == "" {
		return err
	}

	opCheckout := Checkout{
		Repo: repo,
		Options: checkout_options.Options{
			CheckoutMode: checkout_mode.Make(checkout_mode.Blob),
		},
		Utility: c.CheckoutBlobAndRun,
	}

	if _, err = opCheckout.Run(objects); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
