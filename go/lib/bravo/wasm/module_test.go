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

	result, err := mod.CallContainsSku(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !result {
		t.Fatal("expected true from always_true module")
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

		result, err := mod.CallContainsSku(ctx, 0)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}

		if !result {
			t.Fatalf("iteration %d: expected true", i)
		}

		repool()
	}
}
