package queries

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type ExecutorPrimitive struct {
	pqg sku.PrimitiveQueryGroup
	ei  ExecutionInfo
	out interfaces.FuncIter[sku.ExternalLike]
}

func MakeExecutorPrimitive(
	qg sku.PrimitiveQueryGroup,
	fpq sku.FuncPrimitiveQuery,
	froi sku.FuncReadOneInto,
) ExecutorPrimitive {
	return ExecutorPrimitive{
		pqg: qg,
		ei: ExecutionInfo{
			FuncPrimitiveQuery: fpq,
			FuncReadOneInto:    froi,
		},
	}
}

func (e *ExecutorPrimitive) ExecuteExternalLike(
	out interfaces.FuncIter[sku.ExternalLike],
) (err error) {
	if err = e.executeInternalQueryExternalLike(out); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (e *ExecutorPrimitive) ExecuteTransacted(
	out interfaces.FuncIter[*sku.Transacted],
) (err error) {
	out1 := func(el sku.ExternalLike) (err error) {
		sk := el.GetSku()

		if err = out(sk); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	if err = e.executeInternalQueryExternalLike(out1); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (e *ExecutorPrimitive) executeInternalQueryExternalLike(
	out interfaces.FuncIter[sku.ExternalLike],
) (err error) {
	if err = e.ei.FuncPrimitiveQuery(
		e.pqg,
		e.makeEmitSkuSigilLatest(out),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (e *ExecutorPrimitive) makeEmitSkuSigilLatest(
	out interfaces.FuncIter[sku.ExternalLike],
) interfaces.FuncIter[*sku.Transacted] {
	return func(z *sku.Transacted) (err error) {
		g := genres.Must(z.GetGenre())
		m, ok := e.pqg.Get(g)

		if !ok {
			return err
		}

		if m.GetSigil().IncludesExternal() {
			if err = e.ei.UpdateTransacted(z); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if !m.ContainsSku(z) {
			return err
		}

		if err = out(z); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}
