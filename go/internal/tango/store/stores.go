package store

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/delta/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	pkg_query "code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/romeo/env_workspace"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
)

func (store *Store) SaveBlob(el sku.ExternalLike) (err error) {
	es := store.envWorkspace.GetStore()

	if err = es.SaveBlob(el); err != nil {
		if errors.Is(err, env_workspace.ErrUnsupportedOperation{}) {
			err = nil
		} else {
			err = errors.Wrapf(err, "Sku: %s", el)
			return err
		}
	}

	return err
}

func (store *Store) DeleteCheckedOut(col *sku.CheckedOut) (err error) {
	es := store.envWorkspace.GetStore()

	if err = es.DeleteCheckedOut(col); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) CheckoutQuery(
	options checkout_options.Options,
	query *pkg_query.Query,
	out interfaces.FuncIter[sku.SkuType],
) (err error) {
	externalStore := store.envWorkspace.GetStore()

	qf := func(t *sku.Transacted) (err error) {
		var co sku.SkuType

		// TODO include a "query complete" signal for the external store to
		// batch
		// the checkout if necessary
		if co, err = externalStore.CheckoutOne(options, t); err != nil {
			if errors.Is(err, env_workspace.ErrUnsupportedType{}) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}

		if !store.envWorkspace.IsTemporary() {
			if err = store.ui.CheckedOut(co); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if err = out(co); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	if err = store.QueryTransacted(query, qf); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) CheckoutOne(
	repoId ids.RepoId,
	options checkout_options.Options,
	sz *sku.Transacted,
) (cz sku.SkuType, err error) {
	es := store.envWorkspace.GetStore()

	if cz, err = es.CheckoutOne(
		options,
		sz,
	); err != nil {
		err = errors.Wrap(err)
		return cz, err
	}

	return cz, err
}

func (store *Store) UpdateCheckoutFromCheckedOut(
	options checkout_options.OptionsWithoutMode,
	col sku.SkuType,
) (err error) {
	es := store.envWorkspace.GetStore()

	if err = es.UpdateCheckoutFromCheckedOut(
		options,
		col,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) Open(
	repoId ids.RepoId,
	m checkout_mode.Mode,
	ph interfaces.FuncIter[string],
	zsc sku.SkuTypeSet,
) (err error) {
	es := store.envWorkspace.GetStore()

	if err = es.Open(m, ph, zsc); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) makeQueryExecutor(
	queryGroup *pkg_query.Query,
) (executor pkg_query.Executor, err error) {
	if queryGroup == nil {
		if queryGroup, err = store.queryBuilder.BuildQueryGroup(); err != nil {
			err = errors.Wrap(err)
			return executor, err
		}
	}

	externalStore := store.envWorkspace.GetStore()

	executor = pkg_query.MakeExecutorWithExternalStore(
		queryGroup,
		store.streamIndex.ReadPrimitiveQuery,
		store.ReadOneInto,
		externalStore,
	)

	return executor, err
}

// TODO make this configgable
func (store *Store) MergeConflicted(
	conflicted sku.Conflicted,
) (err error) {
	es := store.envWorkspace.GetStore()

	if err = es.Merge(conflicted); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) RunMergeTool(
	conflicted sku.Conflicted,
) (err error) {
	tool := store.storeConfig.GetConfig().ToolOptions.Merge

	switch conflicted.GetSkuExternal().GetRepoId().GetRepoIdString() {
	case "browser":
		err = comments.Implement()

	default:
		var checkedOut sku.SkuType

		if checkedOut, err = store.envWorkspace.GetStoreFS().RunMergeTool(
			tool,
			conflicted,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer store.PutCheckedOutLike(checkedOut)

		if err = store.CreateOrUpdateCheckedOut(checkedOut, false); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (store *Store) UpdateTransactedWithExternal(
	repoId ids.RepoId,
	z *sku.Transacted,
) (err error) {
	es := store.envWorkspace.GetStore()

	if err = es.UpdateTransacted(z); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *Store) ReadCheckedOutFromTransacted(
	repoId ids.RepoId,
	object *sku.Transacted,
) (checkedOut *sku.CheckedOut, err error) {
	workspaceStore := store.envWorkspace.GetStore()

	if checkedOut, err = workspaceStore.ReadCheckedOutFromTransacted(
		object,
	); err != nil {
		err = errors.Wrap(err)
		return checkedOut, err
	}

	return checkedOut, err
}

func (store *Store) UpdateTransactedFromBlobs(
	co *sku.CheckedOut,
) (err error) {
	external := co.GetSkuExternal()

	es := store.envWorkspace.GetStore()

	if err = es.UpdateTransactedFromBlobs(external); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
