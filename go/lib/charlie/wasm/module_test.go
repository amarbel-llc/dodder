package wasm

import (
	"context"
	_ "embed"
	"testing"
)

//go:embed testdata/always_true.wasm
var alwaysTrueWasm []byte

func TestModulePoolBuilderRoundTrip(t *testing.T) {
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

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

func TestModulePoolReuse(t *testing.T) {
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

	builder := MakeModulePoolBuilder(rt)

	pool, err := builder.WithBytes(alwaysTrueWasm).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Borrow, use, return, borrow again -- verify pool reuse works.
	for i := 0; i < 3; i++ {
		mod, repool := pool.GetWithRepool()

		if _, err := mod.CallCabiRealloc(ctx, 0, 0, 4, 64); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}

		repool()
	}
}
