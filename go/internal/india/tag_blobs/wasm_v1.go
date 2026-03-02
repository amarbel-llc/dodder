package tag_blobs

import (
	"context"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/sku_wasm"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type WasmV1 struct {
	sku_wasm.WasmVMPoolV1
	Ctx context.Context
}

func (a *WasmV1) GetQueryable() sku.Queryable {
	return a
}

func (a *WasmV1) Reset() {
}

func (a *WasmV1) ResetWith(b WasmV1) {
}

func (tb *WasmV1) ContainsSku(tg sku.TransactedGetter) bool {
	vm, vmRepool := tb.GetWithRepool()
	defer vmRepool()

	recordPtr, err := sku_wasm.MarshalTransactedToModule(
		tb.Ctx,
		vm.Module,
		tg,
	)
	if err != nil {
		ui.Err().Print(err)
		return false
	}

	result, err := vm.CallContainsSku(tb.Ctx, recordPtr)
	if err != nil {
		ui.Err().Print(err)
		return false
	}

	return result
}
