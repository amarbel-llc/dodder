package wasm

import (
	"context"
	_ "embed"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

//go:embed testdata/always_true.wasm
var alwaysTrueWasm []byte

func TestModulePoolBuilderRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	t.AssertNoError(err)
	defer rt.Close(ctx) //defer:err-checked

	builder := MakeModulePoolBuilder(rt)

	pool, err := builder.WithBytes(alwaysTrueWasm).Build(ctx)
	t.AssertNoError(err)

	mod, repool := pool.GetWithRepool()
	defer repool()

	// Verify generic WASM functionality via cabi_realloc.
	_, err = mod.CallCabiRealloc(ctx, 0, 0, 4, 64)
	t.AssertNoError(err)
}

func TestModulePoolReuse(t1 *testing.T) {
	t := ui.MakeT(t1)
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	t.AssertNoError(err)
	defer rt.Close(ctx) //defer:err-checked

	builder := MakeModulePoolBuilder(rt)

	pool, err := builder.WithBytes(alwaysTrueWasm).Build(ctx)
	t.AssertNoError(err)

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
