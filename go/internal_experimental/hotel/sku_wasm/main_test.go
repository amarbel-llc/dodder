package sku_wasm

import (
	"context"
	_ "embed"
	"testing"

	"code.linenisgreat.com/dodder/go/internal_experimental/charlie/wasm"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

//go:embed testdata/always_true.wasm
var alwaysTrueWasm []byte

//go:embed testdata/genre_filter.wasm
var genreFilterWasm []byte

func TestWasmVMPoolV1GetWithRepool(t1 *testing.T) {
	t := ui.MakeT(t1)
	ctx := context.Background()

	rt, err := wasm.MakeRuntime(ctx)
	t.AssertNoError(err)
	defer rt.Close(ctx) //defer:err-checked

	modulePool, err := wasm.MakeModulePoolBuilder(rt).
		WithBytes(alwaysTrueWasm).
		Build(ctx)
	t.AssertNoError(err)

	vmPool := MakeWasmVMPoolV1(modulePool)

	vm, repool := vmPool.GetWithRepool()
	defer repool()

	t.AssertNotNil(vm.Module, "vm.Module")
}

func TestGenreFilterAcceptsZettel(t1 *testing.T) {
	t := ui.MakeT(t1)
	ctx := context.Background()

	rt, err := wasm.MakeRuntime(ctx)
	t.AssertNoError(err)
	defer rt.Close(ctx) //defer:err-checked

	pool, err := wasm.MakeModulePoolBuilder(rt).WithBytes(genreFilterWasm).Build(ctx)
	t.AssertNoError(err)

	vmPool := MakeWasmVMPoolV1(pool)

	vm, repool := vmPool.GetWithRepool()
	defer repool()

	recordPtr, err := MarshalSkuToModule(ctx, vm.Module,
		"zettel", "test/object", "!text",
		[]string{"project"}, nil,
		"abc123", "a test zettel")
	t.AssertNoError(err)

	result, err := vm.CallContainsSku(ctx, recordPtr)
	t.AssertNoError(err)

	t.AssertTrue(result, "expected genre_filter to accept genre=zettel")
}

func TestGenreFilterRejectsNonZettel(t1 *testing.T) {
	t := ui.MakeT(t1)
	ctx := context.Background()

	rt, err := wasm.MakeRuntime(ctx)
	t.AssertNoError(err)
	defer rt.Close(ctx) //defer:err-checked

	pool, err := wasm.MakeModulePoolBuilder(rt).WithBytes(genreFilterWasm).Build(ctx)
	t.AssertNoError(err)

	vmPool := MakeWasmVMPoolV1(pool)

	vm, repool := vmPool.GetWithRepool()
	defer repool()

	recordPtr, err := MarshalSkuToModule(ctx, vm.Module,
		"tag", "some-tag", "!toml-tag-v1",
		nil, nil, "", "")
	t.AssertNoError(err)

	result, err := vm.CallContainsSku(ctx, recordPtr)
	t.AssertNoError(err)

	t.AssertFalse(result, "expected genre_filter to reject genre=tag")
}
