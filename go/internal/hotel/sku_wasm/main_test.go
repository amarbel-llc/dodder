package sku_wasm

import (
	"context"
	_ "embed"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/charlie/wasm"
)

//go:embed testdata/always_true.wasm
var alwaysTrueWasm []byte

//go:embed testdata/genre_filter.wasm
var genreFilterWasm []byte

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

func TestGenreFilterAcceptsZettel(t *testing.T) {
	ctx := context.Background()

	rt, err := wasm.MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

	pool, err := wasm.MakeModulePoolBuilder(rt).WithBytes(genreFilterWasm).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	vmPool := MakeWasmVMPoolV1(pool)

	vm, repool := vmPool.GetWithRepool()
	defer repool()

	recordPtr, err := MarshalSkuToModule(ctx, vm.Module,
		"zettel", "test/object", "!text",
		[]string{"project"}, nil,
		"abc123", "a test zettel")
	if err != nil {
		t.Fatal(err)
	}

	result, err := vm.CallContainsSku(ctx, recordPtr)
	if err != nil {
		t.Fatal(err)
	}

	if !result {
		t.Fatal("expected genre_filter to accept genre=zettel")
	}
}

func TestGenreFilterRejectsNonZettel(t *testing.T) {
	ctx := context.Background()

	rt, err := wasm.MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

	pool, err := wasm.MakeModulePoolBuilder(rt).WithBytes(genreFilterWasm).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	vmPool := MakeWasmVMPoolV1(pool)

	vm, repool := vmPool.GetWithRepool()
	defer repool()

	recordPtr, err := MarshalSkuToModule(ctx, vm.Module,
		"tag", "some-tag", "!toml-tag-v1",
		nil, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := vm.CallContainsSku(ctx, recordPtr)
	if err != nil {
		t.Fatal(err)
	}

	if result {
		t.Fatal("expected genre_filter to reject genre=tag")
	}
}
