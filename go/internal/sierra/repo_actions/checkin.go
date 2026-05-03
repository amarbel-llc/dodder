package repo_actions

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type Checkin struct {
	*repo
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
	query *queries.Query,
) (err error) {
	var lock sync.Mutex

	results := sku.MakeSkuTypeSetMutable()

	if err = op.GetStore().QuerySkuType(
		query,
		func(co sku.SkuType) (err error) {
			lock.Lock()
			defer lock.Unlock()

			cloned, _ := co.Clone() //repool:owned
			return results.Add(cloned)
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if op.Organize {
		if err = op.runOrganize(query, results); err != nil {
			err = errors.Wrap(err)
			return err
		}

		objects.Resetter.Reset(&op.Proto.Metadata)
	}

	var processed sku.TransactedMutableSet

	if processed, err = op.repo.Checkin(
		results,
		op.Proto,
		op.Delete,
		op.RefreshCheckout,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = op.openBlobIfNecessary(processed); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (op Checkin) runOrganize(
	query *queries.Query,
	results sku.SkuTypeSetMutable,
) (err error) {
	flagDelete := orgie.OptionCommentBooleanFlag{
		Value:   &op.Delete,
		Comment: "delete once checked in",
	}

	opOrganize := MakeOrganize2(
		op.repo,
		orgie.Metadata{
			TagSet: op.Proto.Metadata.GetTags(),
			Type:   op.Proto.Metadata.GetType().ToType(),
			RepoId: query.RepoId,
			OptionCommentSet: orgie.MakeOptionCommentSet(
				map[string]orgie.OptionComment{
					"delete": flagDelete,
				},
				&orgie.OptionCommentUnknown{
					Value: "instructions: to prevent an object from being checked in, delete it entirely",
				},
				orgie.OptionCommentWithKey{
					Key:           "delete",
					OptionComment: flagDelete,
				},
			),
		},
	)

	ui.Log().Print(query)

	var organizeResults orgie.OrganizeResults

	if organizeResults, err = opOrganize.Run(results); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var changes orgie.Changes

	if changes, err = orgie.ChangesFromResults(
		op.GetConfig().GetPrintOptions(),
		organizeResults,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, co := range changes.After.AllSkuAndIndex() {
		clonedCo, _ := co.Clone() //repool:owned
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
	objects sku.TransactedSet,
) (err error) {
	if !c.OpenBlob && c.CheckoutBlobAndRun == "" {
		return err
	}

	opCheckout := MakeCheckout(c.repo)
	opCheckout.Options = checkout_options.Options{
		CheckoutMode: checkout_mode.Make(checkout_mode.Blob),
	}
	opCheckout.Utility = c.CheckoutBlobAndRun

	if _, err = opCheckout.Run(objects); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
