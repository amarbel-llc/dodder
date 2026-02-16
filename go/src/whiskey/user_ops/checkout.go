package user_ops

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/oscar/organize_text"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
)

type Checkout struct {
	*local_working_copy.Repo
	Organize bool
	checkout_options.Options
	Open            bool
	Edit            bool
	Utility         string
	RefreshCheckout bool
}

func (op Checkout) Run(
	transactedObjects sku.TransactedSet,
) (checkedOutObjects sku.SkuTypeSetMutable, err error) {
	var repoId ids.RepoId

	if checkedOutObjects, err = op.RunWithRepoId(
		repoId,
		transactedObjects,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOutObjects, err
	}

	return checkedOutObjects, err
}

func (op Checkout) RunWithRepoId(
	repoId ids.RepoId,
	transactedObjects sku.TransactedSet,
) (checkedOutObjects sku.SkuTypeSetMutable, err error) {
	queryBuilder := op.Repo.MakeQueryBuilder(
		ids.MakeGenre(genres.Zettel),
		nil,
	).WithTransacted(
		transactedObjects,
		ids.SigilExternal,
	).WithOptions(queries.BuilderOptionRequireNonEmptyQuery())

	var query *queries.Query

	if query, err = queryBuilder.BuildQueryGroup(); err != nil {
		err = errors.Wrap(err)
		return checkedOutObjects, err
	}

	if checkedOutObjects, err = op.RunQuery(
		query,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOutObjects, err
	}

	return checkedOutObjects, err
}

func (op Checkout) RunQuery(
	query *queries.Query,
) (checkedOut sku.SkuTypeSetMutable, err error) {
	checkedOut = sku.MakeSkuTypeSetMutable()

	var lock sync.Mutex

	onCheckedOut := func(col sku.SkuType) (err error) {
		lock.Lock()
		defer lock.Unlock()

		cl, _ := col.Clone()

		if err = checkedOut.Add(cl); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	if op.Organize {
		if query, err = op.runOrganize(query, onCheckedOut); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	if err = op.Repo.GetStore().CheckoutQuery(
		op.Options,
		query,
		onCheckedOut,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	if op.Utility != "" {
		eachBlobOp := EachBlob{
			Utility: op.Utility,
			Repo:    op.Repo,
		}

		if err = eachBlobOp.Run(checkedOut); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	if op.Open || op.Edit {
		if err = op.GetStore().Open(
			query.RepoId,
			op.CheckoutMode,
			op.PrinterHeader(),
			checkedOut,
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	if op.Edit {
		if err = op.Reset(); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}

		if _, err = op.Checkin(
			checkedOut,
			sku.Proto{},
			false,
			op.RefreshCheckout,
		); err != nil {
			err = errors.Wrap(err)
			return checkedOut, err
		}
	}

	return checkedOut, err
}

func (op Checkout) runOrganize(
	qgOriginal *queries.Query,
	onCheckedOut interfaces.FuncIter[sku.SkuType],
) (qgModified *queries.Query, err error) {
	opOrganize := Organize{
		Repo: op.Repo,
		Metadata: organize_text.Metadata{
			RepoId: qgOriginal.RepoId,
			OptionCommentSet: organize_text.MakeOptionCommentSet(
				// TODO add other OptionComments
				nil,
				&organize_text.OptionCommentUnknown{
					Value: "instructions: to prevent an object from being checked out, delete it entirely",
				},
			),
		},
		DontUseQueryGroupForOrganizeMetadata: true,
	}

	ui.Log().Print(qgOriginal)

	originalRepoId := qgOriginal.RepoId
	qgOriginal.RepoId.Reset()

	var organizeResults organize_text.OrganizeResults

	if organizeResults, err = opOrganize.RunWithQueryGroup(
		qgOriginal,
	); err != nil {
		err = errors.Wrap(err)
		return qgModified, err
	}

	var changeResults organize_text.Changes

	if changeResults, err = organize_text.ChangesFromResults(
		op.GetConfig().GetPrintOptions(),
		organizeResults,
	); err != nil {
		err = errors.Wrap(err)
		return qgModified, err
	}

	b := op.MakeQueryBuilder(
		ids.MakeGenre(genres.All()...),
		nil,
	).WithTransacted(
		changeResults.After.AsTransactedSet(),
		ids.SigilExternal,
	).WithOptions(queries.BuilderOptions(
		queries.BuilderOptionDoNotMatchEmpty(),
		queries.BuilderOptionRequireNonEmptyQuery(),
	))

	if qgModified, err = b.BuildQueryGroup(); err != nil {
		err = errors.Wrap(err)
		return qgModified, err
	}

	qgModified.RepoId = originalRepoId

	return qgModified, err
}
