package queries

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type (
	QueryCheckedOut interface {
		QueryCheckedOut(
			query *Query,
			output interfaces.FuncIter[sku.SkuType],
		) (err error)
	}

	WorkspaceStore interface {
		domain_interfaces.WorkspaceStoreReadAllExternalItems
		sku.ExternalStoreUpdateTransacted
		sku.ExternalStoreReadExternalLikeFromObjectIdLike
		QueryCheckedOut
	}

	ExecutionInfo struct {
		WorkspaceStore
		sku.FuncPrimitiveQuery
		sku.FuncReadOneInto
	}
)

// TODO use ExecutorPrimitive
type Executor struct {
	primitive
	ExecutionInfo
	Out interfaces.FuncIter[sku.ExternalLike]
}

func MakeExecutorWithExternalStore(
	query *Query,
	fpq sku.FuncPrimitiveQuery,
	froi sku.FuncReadOneInto,
	workspaceStore WorkspaceStore,
) Executor {
	return Executor{
		primitive: primitive{query},
		ExecutionInfo: ExecutionInfo{
			WorkspaceStore:     workspaceStore,
			FuncPrimitiveQuery: fpq,
			FuncReadOneInto:    froi,
		},
	}
}

// TODO refactor into methods that have internal in the name
func (executor *Executor) ExecuteExactlyOneExternalObject(
	permitInternal bool,
) (object *sku.Transacted, err error) {
	if executor.WorkspaceStore != nil {
		var externalObjectId domain_interfaces.ObjectId

		if externalObjectId, _, err = executor.Query.getExactlyOneExternalObjectId(
			permitInternal,
		); err != nil {
			err = errors.Wrap(err)
			return object, err
		}

		object, _ = sku.GetTransactedPool().GetWithRepool()

		var external sku.ExternalLike

		// TODO determine if a nil return is ever valid
		if external, err = executor.ReadExternalLikeFromObjectIdLike(
			sku.CommitOptions{
				StoreOptions: sku.StoreOptions{
					UpdateTai: true,
				},
			},
			externalObjectId,
			object,
		); err != nil {
			err = errors.Wrapf(err, "ExternalObjectId: %q", externalObjectId)
			return object, err
		}

		if external != nil {
			sku.TransactedResetter.ResetWith(object, external.GetSku())
		}
	} else {
		var objectId ObjectId

		if objectId, _, err = executor.Query.getExactlyOneObjectId(); err != nil {
			err = errors.Wrap(err)
			return object, err
		}

		object, _ = sku.GetTransactedPool().GetWithRepool()

		if err = executor.FuncReadOneInto(
			objectId,
			object,
		); err != nil {
			err = errors.Wrap(err)
			return object, err
		}
	}

	return object, err
}

func (executor *Executor) ExecuteExactlyOne() (object *sku.Transacted, err error) {
	var objectId ObjectId
	var sigil ids.Sigil

	if objectId, sigil, err = executor.Query.getExactlyOneObjectId(); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	object, _ = sku.GetTransactedPool().GetWithRepool()

	if err = executor.ExecutionInfo.FuncReadOneInto(objectId, object); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	if !sigil.IncludesExternal() {
		return object, err
	}

	var external sku.ExternalLike

	if external, err = executor.ReadExternalLikeFromObjectIdLike(
		sku.CommitOptions{
			StoreOptions: sku.StoreOptions{
				UpdateTai: true,
			},
		},
		objectId,
		object,
	); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	if external != nil {
		sku.TransactedResetter.ResetWith(object, external.GetSku())
	}

	return object, err
}

func (executor *Executor) ExecuteSkuType(
	out interfaces.FuncIter[sku.SkuType],
) (err error) {
	if executor.WorkspaceStore != nil {
		if err = executor.applyDotOperatorIfNecessary(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = executor.executeExternalQueryCheckedOut(out); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if err = executor.executeInternalQuerySkuType(out); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (e *Executor) ExecuteTransacted(
	out interfaces.FuncIter[*sku.Transacted],
) (err error) {
	if err = e.readAllItemsIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// TODO tease apart the reliance on dotOperatorActive here
	if e.dotOperatorActive && e.WorkspaceStore != nil {
		if err = e.executeExternalQuery(out); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if err = e.executeInternalQuery(out); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (e *Executor) ExecuteTransactedAsSkuType(
	out interfaces.FuncIter[sku.SkuType],
) (err error) {
	if err = e.readAllItemsIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if e.isDotOperatorActive() && e.WorkspaceStore != nil {
		if err = e.executeExternalQueryCheckedOut(out); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if err = e.executeInternalQuerySkuType(out); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (e *Executor) executeExternalQueryCheckedOut(
	out interfaces.FuncIter[sku.SkuType],
) (err error) {
	if err = e.WorkspaceStore.QueryCheckedOut(
		e.Query,
		out,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (e *Executor) executeExternalQuery(
	out interfaces.FuncIter[*sku.Transacted],
) (err error) {
	if err = e.executeExternalQueryCheckedOut(
		func(col sku.SkuType) (err error) {
			z := col.GetSkuExternal()

			if err = out(z); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (e *Executor) executeInternalQuerySkuType(
	out interfaces.FuncIter[sku.SkuType],
) (err error) {
	if err = e.FuncPrimitiveQuery(
		primitive{Query: e.Query},
		e.makeEmitSkuSigilLatestSkuType(out),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (e *Executor) executeInternalQuery(
	out interfaces.FuncIter[*sku.Transacted],
) (err error) {
	if err = e.FuncPrimitiveQuery(
		primitive{e.Query},
		e.makeEmitSkuSigilLatest(out),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (e *Executor) makeEmitSkuSigilLatest(
	out interfaces.FuncIter[*sku.Transacted],
) interfaces.FuncIter[*sku.Transacted] {
	return func(object *sku.Transacted) (err error) {
		g := genres.Must(object.GetGenre())

		if !e.containsSku(object) {
			return err
		}

		// TODO cache query with sigil and object id
		genreQuery, ok := e.Get(g)

		if !ok {
			return err
		}

		if genreQuery.GetSigil().IncludesExternal() && e.WorkspaceStore != nil {
			if err = e.UpdateTransacted(object); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if err = out(object); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}

func (e *Executor) makeEmitSkuSigilLatestSkuType(
	out interfaces.FuncIter[sku.SkuType],
) interfaces.FuncIter[*sku.Transacted] {
	return func(internal *sku.Transacted) (err error) {
		g := genres.Must(internal.GetGenre())

		if !e.containsSku(internal) {
			return err
		}

		m, ok := e.Get(g)

		if !ok {
			return err
		}

		if m.GetSigil().IncludesExternal() {
			// TODO update External
			if err = e.UpdateTransacted(internal); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		co, coRepool := sku.GetCheckedOutPool().GetWithRepool()
		defer coRepool()

		sku.TransactedResetter.ResetWith(co.GetSkuExternal(), internal)

		co.SetState(checked_out_state.Internal)

		if err = out(co); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}

func (executor *Executor) applyDotOperatorIfNecessary() (err error) {
	if !executor.isDotOperatorActive() {
		return err
	}

	if err = executor.readAllItemsIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (executor *Executor) readAllItemsIfNecessary() (err error) {
	if executor.WorkspaceStore == nil {
		return err
	}

	if err = executor.WorkspaceStore.ReadAllExternalItems(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
