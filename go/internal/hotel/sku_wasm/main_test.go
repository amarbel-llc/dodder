package sku_wasm

import (
	"context"
	_ "embed"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/charlie/wasm"
)

//go:embed testdata/always_true.wasm
var alwaysTrueWasm []byte

func TestWasmVMPoolV1GetWithRepool(t *testing.T) {
	ctx := context.Background()

	rt, err := wasm.MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

	modulePool, err := wasm.MakeModulePoolBuilder(rt).
		WithBytes(alwaysTrueWasm).
		Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	vmPool := MakeWasmVMPoolV1(modulePool)

	vm, repool := vmPool.GetWithRepool()
	defer repool()

	if vm.Module == nil {
		t.Fatal("expected non-nil module")
	}
}
