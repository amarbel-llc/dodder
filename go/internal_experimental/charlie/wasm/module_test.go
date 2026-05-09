package wasm

import (
	"context"
	_ "embed"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

//go:embed testdata/always_true.wasm
var alwaysTrueWasm []byte

func TestModulePoolBuilderRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx) //defer:err-checked

	builder := MakeModulePoolBuilder(rt)

	pool, err := builder.WithBytes(alwaysTrueWasm).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mod, repool := pool.GetWithRepool()
	defer repool()

	// Verify generic WASM functionality via cabi_realloc.
	if _, err := mod.CallCabiRealloc(ctx, 0, 0, 4, 64); err != nil {
		t.Fatal(err)
	}
}

func TestModulePoolReuse(t1 *testing.T) {
	t := ui.MakeT(t1)
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx) //defer:err-checked

	builder := MakeModulePoolBuilder(rt)

	pool, err := builder.WithBytes(alwaysTrueWasm).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Borrow, use, return, borrow again -- verify pool reuse works.
	for i := range 3 {
		func() {
			mod, repool := pool.GetWithRepool()
			defer repool()

			if _, err := mod.CallCabiRealloc(ctx, 0, 0, 4, 64); err != nil {
				t.Fatalf("iteration %d: %v", i, err)
			}
		}()
	}
}
