package sku

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
)

func makeCheckedOut() (*CheckedOut, interfaces.FuncRepool) {
	return GetCheckedOutPool().GetWithRepool()
}

func cloneFromTransactedCheckedOut(
	src *Transacted,
	newState checked_out_state.State,
) (*CheckedOut, interfaces.FuncRepool) {
	dst, repool := GetCheckedOutPool().GetWithRepool()
	TransactedResetter.ResetWith(dst.GetSku(), src)
	TransactedResetter.ResetWith(dst.GetSkuExternal(), src)
	dst.state = newState
	return dst, repool
}

func cloneCheckedOut(co *CheckedOut) (*CheckedOut, interfaces.FuncRepool) {
	return co.Clone()
}

type objectFactoryCheckedOut struct {
	interfaces.Pool[*CheckedOut]
	interfaces.Resetter[*CheckedOut]
}

func (factory *objectFactoryCheckedOut) SetDefaultsIfNecessary() objectFactoryCheckedOut {
	if factory.Resetter == nil {
		factory.Resetter = pool.BespokeResetter[*CheckedOut]{
			FuncReset: func(e *CheckedOut) {
				CheckedOutResetter.Reset(e)
			},
			FuncResetWith: func(dst, src *CheckedOut) {
				CheckedOutResetter.ResetWith(dst, src)
			},
		}
	}

	if factory.Pool == nil {
		factory.Pool = pool.Bespoke[*CheckedOut]{
			FuncGet: func() *CheckedOut {
				co, _ := GetCheckedOutPool().GetWithRepool()
				return co
			},
			FuncPut: func(e *CheckedOut) {
				// no-op: pool items returned via repool
			},
		}
	}

	return *factory
}
