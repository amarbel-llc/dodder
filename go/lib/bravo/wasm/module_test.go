package wasm

import (
	"context"
	_ "embed"
	"testing"
)

//go:embed testdata/always_true.wasm
var alwaysTrueWasm []byte

//go:embed testdata/genre_filter.wasm
var genreFilterWasm []byte

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

func TestGenreFilterAcceptsZettel(t *testing.T) {
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

	pool, err := MakeModulePoolBuilder(rt).WithBytes(genreFilterWasm).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mod, repool := pool.GetWithRepool()
	defer repool()

	recordPtr, err := MarshalSkuToModule(ctx, mod,
		"zettel", "test/object", "!text",
		[]string{"project"}, nil,
		"abc123", "a test zettel")
	if err != nil {
		t.Fatal(err)
	}

	result, err := mod.CallContainsSku(ctx, recordPtr)
	if err != nil {
		t.Fatal(err)
	}

	if !result {
		t.Fatal("expected genre_filter to accept genre=zettel")
	}
}

func TestGenreFilterRejectsNonZettel(t *testing.T) {
	ctx := context.Background()

	rt, err := MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

	pool, err := MakeModulePoolBuilder(rt).WithBytes(genreFilterWasm).Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mod, repool := pool.GetWithRepool()
	defer repool()

	recordPtr, err := MarshalSkuToModule(ctx, mod,
		"tag", "some-tag", "!toml-tag-v1",
		nil, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := mod.CallContainsSku(ctx, recordPtr)
	if err != nil {
		t.Fatal(err)
	}

	if result {
		t.Fatal("expected genre_filter to reject genre=tag")
	}
}
