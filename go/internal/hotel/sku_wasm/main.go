package sku_wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/wasm"
	"github.com/tetratelabs/wazero/api"
)

type WasmVMV1 struct {
	*wasm.Module
	containsSku api.Function
}

func (vm *WasmVMV1) CallContainsSku(
	ctx context.Context,
	recordPtr uint32,
) (bool, error) {
	results, err := vm.containsSku.Call(ctx, uint64(recordPtr))
	if err != nil {
		return false, errors.Wrap(err)
	}

	return results[0] != 0, nil
}

type WasmVMPoolV1 = interfaces.PoolPtr[WasmVMV1, *WasmVMV1]

func MakeWasmVMPoolV1(modulePool *wasm.ModulePool) WasmVMPoolV1 {
	return pool.Make(
		func() (out *WasmVMV1) {
			mod, _ := modulePool.GetWithRepool() //repool:owned

			containsSku := mod.Inner().ExportedFunction("contains-sku")
			if containsSku == nil {
				panic("WASM module missing export: contains-sku")
			}

			out = &WasmVMV1{
				Module:      mod,
				containsSku: containsSku,
			}

			return out
		},
		nil,
	)
}

// MarshalTransactedToModule writes an sku.Transacted into a WASM module's
// linear memory as a canonical ABI SKU record. Returns the record pointer.
func MarshalTransactedToModule(
	ctx context.Context,
	mod *wasm.Module,
	tg sku.TransactedGetter,
) (recordPtr uint32, err error) {
	object := tg.GetSku()

	genre := object.GetGenre().String()
	objectId := object.GetObjectId().String()
	tipe := object.GetType().String()
	blobDigest := object.GetBlobDigest().String()
	description := object.GetMetadata().GetDescription().String()

	var tags []string
	for tag := range object.GetMetadata().AllTags() {
		tags = append(tags, tag.String())
	}

	var tagsImplicit []string
	for tag := range object.GetMetadata().GetIndex().GetImplicitTags().All() {
		tagsImplicit = append(tagsImplicit, tag.String())
	}

	return MarshalSkuToModule(
		ctx, mod,
		genre, objectId, tipe,
		tags, tagsImplicit,
		blobDigest, description,
	)
}
