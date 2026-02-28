package sku_wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/wasm"
)

type WasmVMV1 struct {
	*wasm.Module
}

type WasmVMPoolV1 = interfaces.PoolPtr[WasmVMV1, *WasmVMV1]

func MakeWasmVMPoolV1(modulePool *wasm.ModulePool) WasmVMPoolV1 {
	return pool.Make(
		func() (out *WasmVMV1) {
			mod, _ := modulePool.GetWithRepool() //repool:owned

			out = &WasmVMV1{
				Module: mod,
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

	return wasm.MarshalSkuToModule(
		ctx, mod,
		genre, objectId, tipe,
		tags, tagsImplicit,
		blobDigest, description,
	)
}
