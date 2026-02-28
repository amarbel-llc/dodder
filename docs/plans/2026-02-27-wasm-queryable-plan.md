# WASM Queryable Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a new `sku.Queryable` backed by WebAssembly modules, letting users
write query filters in any language that compiles to WASM.

**Architecture:** Uses wazero (pure Go, zero CGo) as the WASM runtime. A
WIT-defined contract (`dodder:query@0.1.0`) specifies the `contains-sku`
export. The Go host marshals SKU data into WASM linear memory using the
canonical ABI subset (strings, lists, records, bools). Module instances are
pooled following the same `interfaces.PoolPtr` pattern as Lua VMs.

**Tech Stack:** wazero, WIT/canonical ABI, wit-bindgen (guest side), Go generics
for pool types.

**Design doc:** `docs/plans/2026-02-27-wasm-queryable-design.md`

**Skills:** @design_patterns-pool_repool @design_patterns-horizontal_versioning
@design_patterns-hamster_style @dodder-development

---

### Task 1: Add wazero dependency

**Files:**
- Modify: `go/go.mod`
- Modify: `go/go.sum`

**Step 1: Add the wazero module**

Run:
```bash
cd go && go get github.com/tetratelabs/wazero@latest
```

**Step 2: Verify it resolves**

Run:
```bash
cd go && go mod tidy
```
Expected: clean exit, `go.mod` and `go.sum` updated with wazero.

**Step 3: Commit**

```bash
git add go/go.mod go/go.sum
git commit -m "build: add wazero dependency for WASM queryable"
```

---

### Task 2: Create WIT contract file

**Files:**
- Create: `wit/dodder-query.wit`

**Step 1: Write the WIT file**

```wit
package dodder:query@0.1.0;

interface types {
    record sku {
        genre: string,
        object-id: string,
        %type: string,
        tags: list<string>,
        tags-implicit: list<string>,
        blob-digest: string,
        description: string,
    }
}

world query-filter {
    use types.{sku};
    export contains-sku: func(object: sku) -> bool;
}
```

**Step 2: Commit**

```bash
git add wit/dodder-query.wit
git commit -m "feat: add WIT contract for WASM query filter"
```

---

### Task 3: Create lib/bravo/wasm/ — canonical ABI helpers

This package provides low-level helpers for reading/writing canonical ABI types
in WASM linear memory. These are domain-agnostic building blocks.

**Files:**
- Create: `go/lib/bravo/wasm/canonical_abi.go`
- Create: `go/lib/bravo/wasm/canonical_abi_test.go`

**Step 1: Write failing tests for canonical ABI string read/write**

Test that `WriteString` writes UTF-8 bytes into a byte slice at an offset and
returns the `(ptr, len)` pair, and `ReadString` reverses it. Test
`WriteStringList` for `list<string>` lowering.

```go
package wasm

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteStringRoundTrip(t *testing.T) {
	memory := make([]byte, 1024)
	allocator := MakeBumpAllocator(memory, 0)

	input := "hello"
	ptr, length := WriteString(&allocator, input)

	if length != 5 {
		t.Fatalf("expected length 5, got %d", length)
	}

	got := ReadString(memory, ptr, length)
	if got != input {
		t.Fatalf("expected %q, got %q", input, got)
	}
}

func TestWriteStringListRoundTrip(t *testing.T) {
	memory := make([]byte, 4096)
	allocator := MakeBumpAllocator(memory, 0)

	input := []string{"alpha", "bravo", "charlie"}
	ptr, count := WriteStringList(&allocator, input)

	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	got := ReadStringList(memory, ptr, count)
	if diff := cmp.Diff(input, got); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./lib/bravo/wasm/`
Expected: FAIL — package doesn't exist yet.

**Step 3: Implement canonical ABI helpers**

```go
package wasm

import (
	"encoding/binary"
	"unsafe"
)

// BumpAllocator is a simple arena allocator over a byte slice. Mirrors the
// canonical ABI's cabi_realloc pattern but operates on the host side for
// writing data before copying to WASM memory.
type BumpAllocator struct {
	memory []byte
	offset uint32
}

func MakeBumpAllocator(memory []byte, offset uint32) BumpAllocator {
	return BumpAllocator{memory: memory, offset: offset}
}

func (a *BumpAllocator) Alloc(size, align uint32) uint32 {
	// Align up
	a.offset = (a.offset + align - 1) &^ (align - 1)
	ptr := a.offset
	a.offset += size
	return ptr
}

// WriteString writes a UTF-8 string into the allocator's memory and returns
// the (ptr, len) pair for canonical ABI string representation.
func WriteString(alloc *BumpAllocator, s string) (ptr, length uint32) {
	length = uint32(len(s))
	ptr = alloc.Alloc(length, 1)
	copy(alloc.memory[ptr:ptr+length], s)
	return ptr, length
}

// ReadString reads a canonical ABI string from memory at (ptr, length).
func ReadString(memory []byte, ptr, length uint32) string {
	return string(memory[ptr : ptr+length])
}

// WriteStringList writes a list<string> in canonical ABI format. Each element
// is a (ptr: u32, len: u32) pair. Returns the pointer to the list header and
// the element count.
func WriteStringList(
	alloc *BumpAllocator,
	strings []string,
) (listPtr, count uint32) {
	count = uint32(len(strings))

	// First, write all string data and collect (ptr, len) pairs
	type stringPair struct{ ptr, length uint32 }
	pairs := make([]stringPair, count)
	for i, s := range strings {
		pairs[i].ptr, pairs[i].length = WriteString(alloc, s)
	}

	// Then write the list elements: array of (ptr: u32, len: u32)
	listPtr = alloc.Alloc(count*8, 4)
	for i, p := range pairs {
		offset := listPtr + uint32(i)*8
		binary.LittleEndian.PutUint32(alloc.memory[offset:], p.ptr)
		binary.LittleEndian.PutUint32(alloc.memory[offset+4:], p.length)
	}

	return listPtr, count
}

// ReadStringList reads a canonical ABI list<string> from memory.
func ReadStringList(
	memory []byte,
	listPtr, count uint32,
) []string {
	result := make([]string, count)
	for i := uint32(0); i < count; i++ {
		offset := listPtr + i*8
		ptr := binary.LittleEndian.Uint32(memory[offset:])
		length := binary.LittleEndian.Uint32(memory[offset+4:])
		result[i] = ReadString(memory, ptr, length)
	}
	return result
}

// RecordSize returns the byte size of the flat SKU record layout:
// 5 strings (ptr+len each = 2*4 = 8 bytes) + 2 lists (ptr+count each = 8
// bytes) = 7 * 8 = 56 bytes.
const SkuRecordSize = 7 * 8

// WriteSkuRecord writes the flat record layout for a canonical ABI sku record.
// Fields are laid out sequentially: genre, object-id, type, tags,
// tags-implicit, blob-digest, description. Each string is (ptr: u32, len: u32)
// and each list is (ptr: u32, count: u32).
func WriteSkuRecord(
	alloc *BumpAllocator,
	genre, objectId, tipe string,
	tags, tagsImplicit []string,
	blobDigest, description string,
) uint32 {
	// Write all data first
	genrePtr, genreLen := WriteString(alloc, genre)
	objectIdPtr, objectIdLen := WriteString(alloc, objectId)
	tipePtr, tipeLen := WriteString(alloc, tipe)
	tagsPtr, tagsCount := WriteStringList(alloc, tags)
	tagsImplicitPtr, tagsImplicitCount := WriteStringList(alloc, tagsImplicit)
	blobDigestPtr, blobDigestLen := WriteString(alloc, blobDigest)
	descriptionPtr, descriptionLen := WriteString(alloc, description)

	// Write the flat record struct
	recordPtr := alloc.Alloc(SkuRecordSize, 4)
	m := alloc.memory[recordPtr:]

	fields := []uint32{
		genrePtr, genreLen,
		objectIdPtr, objectIdLen,
		tipePtr, tipeLen,
		tagsPtr, tagsCount,
		tagsImplicitPtr, tagsImplicitCount,
		blobDigestPtr, blobDigestLen,
		descriptionPtr, descriptionLen,
	}

	for i, v := range fields {
		binary.LittleEndian.PutUint32(m[i*4:], v)
	}

	return recordPtr
}

// Suppress unused import warning for unsafe (used conceptually for alignment).
var _ = unsafe.Sizeof
```

**Step 4: Run tests**

Run: `cd go && go test -v -tags test,debug ./lib/bravo/wasm/`
Expected: PASS

**Step 5: Commit**

```bash
git add go/lib/bravo/wasm/canonical_abi.go go/lib/bravo/wasm/canonical_abi_test.go
git commit -m "feat(bravo/wasm): add canonical ABI helpers for string and list marshaling"
```

---

### Task 4: Create lib/bravo/wasm/ — Runtime and Module types

**Files:**
- Create: `go/lib/bravo/wasm/main.go`
- Create: `go/lib/bravo/wasm/module.go`
- Create: `go/lib/bravo/wasm/module_pool.go`
- Create: `go/lib/bravo/wasm/module_pool_builder.go`
- Create: `go/lib/bravo/wasm/module_test.go`

**Step 1: Write failing test for ModulePoolBuilder with a trivial WASM module**

We need a pre-compiled `.wasm` binary for tests. Create a test fixture by
compiling a minimal WAT module using `wat2wasm` (available in the nix devshell
via `wabt`). The test module exports `contains-sku` that always returns 1, plus
`cabi_realloc` and `reset`.

For the test, embed a pre-compiled `.wasm` as a Go byte literal (or use
`//go:embed` with a `.wasm` test fixture file).

Create a test fixture WAT file at `go/lib/bravo/wasm/testdata/always_true.wat`:

```wat
(module
  ;; Memory for canonical ABI
  (memory (export "memory") 1)

  ;; Bump allocator state
  (global $arena_offset (mut i32) (i32.const 0))

  ;; cabi_realloc: canonical ABI allocator
  ;; (old_ptr, old_size, align, new_size) -> ptr
  (func (export "cabi_realloc")
    (param $old_ptr i32) (param $old_size i32)
    (param $align i32) (param $new_size i32)
    (result i32)
    (local $ptr i32)
    ;; Align up: ptr = (offset + align - 1) & ~(align - 1)
    (local.set $ptr
      (i32.and
        (i32.add (global.get $arena_offset) (i32.sub (local.get $align) (i32.const 1)))
        (i32.xor (i32.sub (local.get $align) (i32.const 1)) (i32.const -1))
      )
    )
    (global.set $arena_offset (i32.add (local.get $ptr) (local.get $new_size)))
    (local.get $ptr)
  )

  ;; reset: clear arena
  (func (export "reset")
    (global.set $arena_offset (i32.const 0))
  )

  ;; contains-sku: always returns true (1)
  ;; Takes a pointer to a flat SKU record, ignores it
  (func (export "contains-sku") (param $record_ptr i32) (result i32)
    (i32.const 1)
  )
)
```

Compile with: `wat2wasm go/lib/bravo/wasm/testdata/always_true.wat -o go/lib/bravo/wasm/testdata/always_true.wasm`

Then write the test in `go/lib/bravo/wasm/module_test.go`:

```go
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

	// Borrow, use, return, borrow again — verify pool reuse works.
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
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./lib/bravo/wasm/`
Expected: FAIL — types don't exist yet.

**Step 3: Implement Runtime, Module, ModulePool, ModulePoolBuilder**

`go/lib/bravo/wasm/main.go`:

```go
package wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"github.com/tetratelabs/wazero"
)

type Runtime struct {
	inner wazero.Runtime
}

func MakeRuntime(ctx context.Context) (rt *Runtime, err error) {
	inner := wazero.NewRuntimeWithConfig(
		ctx,
		wazero.NewRuntimeConfigCompiler(),
	)

	rt = &Runtime{inner: inner}
	return rt, err
}

func (rt *Runtime) Close(ctx context.Context) error {
	return errors.Wrap(rt.inner.Close(ctx))
}
```

`go/lib/bravo/wasm/module.go`:

```go
package wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"github.com/tetratelabs/wazero/api"
)

type Module struct {
	mod         api.Module
	memory      api.Memory
	containsSku api.Function
	cabiRealloc api.Function
	resetFn     api.Function
}

func (m *Module) CallContainsSku(
	ctx context.Context,
	recordPtr uint32,
) (bool, error) {
	results, err := m.containsSku.Call(ctx, uint64(recordPtr))
	if err != nil {
		return false, errors.Wrap(err)
	}

	return results[0] != 0, nil
}

func (m *Module) CallCabiRealloc(
	ctx context.Context,
	oldPtr, oldSize, align, newSize uint32,
) (uint32, error) {
	results, err := m.cabiRealloc.Call(
		ctx,
		uint64(oldPtr), uint64(oldSize),
		uint64(align), uint64(newSize),
	)
	if err != nil {
		return 0, errors.Wrap(err)
	}

	return uint32(results[0]), nil
}

func (m *Module) CallReset(ctx context.Context) error {
	_, err := m.resetFn.Call(ctx)
	return errors.Wrap(err)
}

func (m *Module) Memory() api.Memory {
	return m.memory
}

func (m *Module) WriteBytes(offset uint32, data []byte) bool {
	return m.memory.Write(offset, data)
}

func (m *Module) ReadBytes(offset, size uint32) ([]byte, bool) {
	return m.memory.Read(offset, size)
}
```

`go/lib/bravo/wasm/module_pool.go`:

Reference `go/lib/bravo/lua/vm_pool.go` for the pattern. Uses
`interfaces.PoolPtr[Module, *Module]` and `pool.Make`.

```go
package wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"github.com/tetratelabs/wazero"
	wazero_api "github.com/tetratelabs/wazero/api"
)

type ModulePool struct {
	interfaces.PoolPtr[Module, *Module]
	compiled wazero.CompiledModule
	runtime  *Runtime
	ctx      context.Context
}

func makeModulePool(
	ctx context.Context,
	rt *Runtime,
	compiled wazero.CompiledModule,
) *ModulePool {
	mp := &ModulePool{
		compiled: compiled,
		runtime:  rt,
		ctx:      ctx,
	}

	mp.PoolPtr = pool.Make(
		func() (mod *Module) {
			m, err := rt.inner.InstantiateModule(
				ctx,
				compiled,
				wazero.NewModuleConfig().WithName(""),
			)
			if err != nil {
				panic(errors.Wrap(err))
			}

			mod = &Module{
				mod:         m,
				memory:      m.Memory(),
				containsSku: m.ExportedFunction("contains-sku"),
				cabiRealloc: m.ExportedFunction("cabi_realloc"),
				resetFn:     m.ExportedFunction("reset"),
			}

			if mod.containsSku == nil {
				panic("WASM module missing export: contains-sku")
			}

			if mod.cabiRealloc == nil {
				panic("WASM module missing export: cabi_realloc")
			}

			return mod
		},
		func(mod *Module) {
			if mod.resetFn != nil {
				if err := mod.CallReset(ctx); err != nil {
					panic(errors.Wrap(err))
				}
			}
		},
	)

	return mp
}
```

`go/lib/bravo/wasm/module_pool_builder.go`:

Reference `go/lib/bravo/lua/vm_pool_builder.go` for the pattern.

```go
package wasm

import (
	"context"
	"io"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"github.com/tetratelabs/wazero"
)

type ModulePoolBuilder struct {
	runtime  *Runtime
	wasmData []byte
	compiled wazero.CompiledModule
}

func MakeModulePoolBuilder(rt *Runtime) *ModulePoolBuilder {
	return &ModulePoolBuilder{runtime: rt}
}

func (b *ModulePoolBuilder) WithBytes(data []byte) *ModulePoolBuilder {
	b.wasmData = data
	return b
}

func (b *ModulePoolBuilder) WithReader(r io.Reader) *ModulePoolBuilder {
	data, err := io.ReadAll(r)
	if err != nil {
		panic(errors.Wrap(err))
	}

	b.wasmData = data
	return b
}

func (b *ModulePoolBuilder) WithCompiled(
	compiled wazero.CompiledModule,
) *ModulePoolBuilder {
	b.compiled = compiled
	return b
}

func (b *ModulePoolBuilder) Build(
	ctx context.Context,
) (mp *ModulePool, err error) {
	if b.compiled == nil && b.wasmData == nil {
		err = errors.ErrorWithStackf("no WASM data or compiled module set")
		return mp, err
	}

	if b.compiled == nil {
		if b.compiled, err = b.runtime.inner.CompileModule(
			ctx,
			b.wasmData,
		); err != nil {
			err = errors.Wrap(err)
			return mp, err
		}
	}

	mp = makeModulePool(ctx, b.runtime, b.compiled)

	// Verify the module works by borrowing and returning one instance.
	mod, repool := mp.GetWithRepool()
	defer repool()
	_ = mod

	return mp, err
}
```

**Step 4: Compile the WAT test fixture**

Run: `wat2wasm go/lib/bravo/wasm/testdata/always_true.wat -o go/lib/bravo/wasm/testdata/always_true.wasm`

If `wat2wasm` is not available, add `wabt` to the nix devshell or install it.
Verify: `ls go/lib/bravo/wasm/testdata/always_true.wasm`

**Step 5: Run tests**

Run: `cd go && go test -v -tags test,debug ./lib/bravo/wasm/`
Expected: PASS

**Step 6: Commit**

```bash
git add go/lib/bravo/wasm/ wit/
git commit -m "feat(bravo/wasm): add wazero runtime, module pool, and builder"
```

---

### Task 5: Create test fixture — tag filter WASM module

A second `.wat` test fixture that actually inspects the SKU record: returns true
only if the genre string equals "zettel".

**Files:**
- Create: `go/lib/bravo/wasm/testdata/genre_filter.wat`
- Create: `go/lib/bravo/wasm/testdata/genre_filter.wasm` (compiled)
- Modify: `go/lib/bravo/wasm/module_test.go`

**Step 1: Write the WAT module**

```wat
(module
  (memory (export "memory") 1)
  (global $arena_offset (mut i32) (i32.const 0))

  (func (export "cabi_realloc")
    (param $old_ptr i32) (param $old_size i32)
    (param $align i32) (param $new_size i32) (result i32)
    (local $ptr i32)
    (local.set $ptr
      (i32.and
        (i32.add (global.get $arena_offset) (i32.sub (local.get $align) (i32.const 1)))
        (i32.xor (i32.sub (local.get $align) (i32.const 1)) (i32.const -1))
      )
    )
    (global.set $arena_offset (i32.add (local.get $ptr) (local.get $new_size)))
    (local.get $ptr)
  )

  (func (export "reset")
    (global.set $arena_offset (i32.const 0))
  )

  ;; contains-sku: returns true if genre == "zettel" (6 bytes)
  ;; Record layout: genre_ptr(i32), genre_len(i32), ...
  (func (export "contains-sku") (param $record_ptr i32) (result i32)
    (local $genre_ptr i32)
    (local $genre_len i32)

    ;; Read genre_ptr and genre_len from record
    (local.set $genre_ptr (i32.load (local.get $record_ptr)))
    (local.set $genre_len (i32.load (i32.add (local.get $record_ptr) (i32.const 4))))

    ;; Check length == 6
    (if (i32.ne (local.get $genre_len) (i32.const 6))
      (then (return (i32.const 0)))
    )

    ;; Compare "zettel" byte by byte
    ;; z=122 e=101 t=116 t=116 e=101 l=108
    (if (i32.ne (i32.load8_u (local.get $genre_ptr)) (i32.const 122))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 1))) (i32.const 101))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 2))) (i32.const 116))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 3))) (i32.const 116))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 4))) (i32.const 101))
      (then (return (i32.const 0))))
    (if (i32.ne (i32.load8_u (i32.add (local.get $genre_ptr) (i32.const 5))) (i32.const 108))
      (then (return (i32.const 0))))

    (i32.const 1)
  )
)
```

**Step 2: Compile**

Run: `wat2wasm go/lib/bravo/wasm/testdata/genre_filter.wat -o go/lib/bravo/wasm/testdata/genre_filter.wasm`

**Step 3: Add tests that marshal a real SKU record into WASM memory**

```go
//go:embed testdata/genre_filter.wasm
var genreFilterWasm []byte

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

	// Allocate space in WASM memory and write the SKU record
	recordPtr, err := MarshalSkuToModule(
		ctx, mod,
		"zettel", "test/object", "!text",
		[]string{"project"}, nil,
		"abc123", "a test zettel",
	)
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

	recordPtr, err := MarshalSkuToModule(
		ctx, mod,
		"tag", "some-tag", "!toml-tag-v1",
		nil, nil,
		"", "",
	)
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
```

**Step 4: Implement MarshalSkuToModule**

Add to `go/lib/bravo/wasm/canonical_abi.go`:

```go
// MarshalSkuToModule writes an SKU record into a WASM module's linear memory
// using the guest-exported cabi_realloc for allocation. Returns the pointer to
// the record in WASM memory.
func MarshalSkuToModule(
	ctx context.Context,
	mod *Module,
	genre, objectId, tipe string,
	tags, tagsImplicit []string,
	blobDigest, description string,
) (recordPtr uint32, err error) {
	// Calculate total size needed
	totalStringBytes := uint32(len(genre) + len(objectId) + len(tipe) +
		len(blobDigest) + len(description))

	for _, t := range tags {
		totalStringBytes += uint32(len(t))
	}
	for _, t := range tagsImplicit {
		totalStringBytes += uint32(len(t))
	}

	// List element arrays: each string in a list is (ptr: u32, len: u32) = 8 bytes
	listOverhead := uint32((len(tags) + len(tagsImplicit)) * 8)

	// Record itself: 7 fields * 8 bytes = 56 bytes
	totalSize := totalStringBytes + listOverhead + SkuRecordSize + 256 // padding

	// Allocate a block in WASM memory
	basePtr, err := mod.CallCabiRealloc(ctx, 0, 0, 4, totalSize)
	if err != nil {
		return 0, err
	}

	// Get a view of WASM memory
	memData, ok := mod.ReadBytes(basePtr, totalSize)
	if !ok {
		return 0, errors.ErrorWithStackf("failed to read WASM memory at %d", basePtr)
	}

	// Use our host-side bump allocator to lay out data within the block
	buf := make([]byte, totalSize)
	alloc := MakeBumpAllocator(buf, 0)

	WriteSkuRecord(
		&alloc,
		genre, objectId, tipe,
		tags, tagsImplicit,
		blobDigest, description,
	)

	// Adjust all pointers to be relative to basePtr in WASM memory
	// The allocator wrote pointers relative to 0 in our buffer, but they
	// need to be relative to basePtr in WASM linear memory.
	adjustPointers(buf, basePtr, 7)

	// Copy the buffer into WASM memory
	mod.WriteBytes(basePtr, buf[:alloc.offset])

	// The record is the last thing written by WriteSkuRecord
	recordPtr = basePtr + alloc.offset - SkuRecordSize

	return recordPtr, nil
}
```

Note: The `adjustPointers` function and the exact pointer adjustment logic will
need refinement during implementation — the key idea is that all pointers in the
canonical ABI record must be absolute addresses in WASM linear memory, not
relative to the buffer start. This may require refactoring `WriteSkuRecord` to
accept a `basePtr` offset, or making `BumpAllocator` start at `basePtr`.

**Step 5: Run tests**

Run: `cd go && go test -v -tags test,debug ./lib/bravo/wasm/`
Expected: PASS

**Step 6: Commit**

```bash
git add go/lib/bravo/wasm/
git commit -m "feat(bravo/wasm): add genre_filter test fixture and SKU marshaling to WASM memory"
```

---

### Task 6: Register TypeWasmTagV1

**Files:**
- Modify: `go/internal/echo/ids/types_builtin.go:24-26` (add constant)
- Modify: `go/internal/echo/ids/types_builtin.go:83-84` (add registration)

**Step 1: Add the type constant**

In `go/internal/echo/ids/types_builtin.go`, after line 25 (`TypeLuaTagV2`), add:

```go
TypeWasmTagV1 = "!wasm-tag-v1"
```

**Step 2: Register in init()**

After line 84 (`registerBuiltinTypeString(TypeLuaTagV2, genres.Tag, false)`),
add:

```go
registerBuiltinTypeString(TypeWasmTagV1, genres.Tag, false)
```

**Step 3: Verify compilation**

Run: `cd go && go build ./internal/echo/ids/`
Expected: clean build.

**Step 4: Commit**

```bash
git add go/internal/echo/ids/types_builtin.go
git commit -m "feat(echo/ids): register TypeWasmTagV1 builtin type"
```

---

### Task 7: Create internal/kilo/sku_wasm/ — SKU-to-WASM bridge

**Files:**
- Create: `go/internal/kilo/sku_wasm/main.go`
- Create: `go/internal/kilo/sku_wasm/main_test.go`

**Step 1: Write failing test**

```go
package sku_wasm

import (
	"context"
	_ "embed"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/bravo/wasm"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
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
```

Symlink or copy the test WASM fixture:
`mkdir -p go/internal/kilo/sku_wasm/testdata && cp go/lib/bravo/wasm/testdata/always_true.wasm go/internal/kilo/sku_wasm/testdata/`

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./internal/kilo/sku_wasm/`
Expected: FAIL — package doesn't exist.

**Step 3: Implement**

`go/internal/kilo/sku_wasm/main.go`:

```go
package sku_wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/wasm"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type WasmVMV1 struct {
	*wasm.Module
}

type WasmVMPoolV1 = interfaces.PoolPtr[WasmVMV1, *WasmVMV1]

func MakeWasmVMPoolV1(modulePool *wasm.ModulePool) WasmVMPoolV1 {
	return pool.Make(
		func() (out *WasmVMV1) {
			mod, _ := modulePool.GetWithRepool()

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
```

**Step 4: Run tests**

Run: `cd go && go test -v -tags test,debug ./internal/kilo/sku_wasm/`
Expected: PASS

**Step 5: Commit**

```bash
git add go/internal/kilo/sku_wasm/
git commit -m "feat(kilo/sku_wasm): add WasmVMV1 pool and SKU marshaling bridge"
```

---

### Task 8: Create internal/lima/tag_blobs/wasm_v1.go — Queryable implementation

**Files:**
- Create: `go/internal/lima/tag_blobs/wasm_v1.go`

**Step 1: Write failing test**

Add to a test file `go/internal/lima/tag_blobs/wasm_v1_test.go`:

```go
package tag_blobs

import (
	"context"
	_ "embed"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/bravo/wasm"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku_wasm"
)

//go:embed testdata/genre_filter.wasm
var genreFilterWasm []byte

func TestWasmV1ContainsSkuAcceptsZettel(t *testing.T) {
	ctx := context.Background()

	rt, err := wasm.MakeRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(ctx)

	modulePool, err := wasm.MakeModulePoolBuilder(rt).
		WithBytes(genreFilterWasm).
		Build(ctx)
	if err != nil {
		t.Fatal(err)
	}

	wasmBlob := &WasmV1{
		WasmVMPoolV1: sku_wasm.MakeWasmVMPoolV1(modulePool),
		ctx:          ctx,
	}

	// Create a test transacted with genre "zettel"
	object := sku.GetTransactedPool().Get()
	defer sku.GetTransactedPool().Put(object)

	// Set genre to zettel via ObjectId
	// (Exact setup depends on how ObjectId.SetGenre works)
	object.ObjectId.SetGenre(genres.Zettel)

	if !wasmBlob.ContainsSku(object) {
		t.Fatal("expected WasmV1 to accept genre=zettel")
	}
}
```

Copy the genre_filter.wasm fixture:
`cp go/lib/bravo/wasm/testdata/genre_filter.wasm go/internal/lima/tag_blobs/testdata/`

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./internal/lima/tag_blobs/`
Expected: FAIL — `WasmV1` type doesn't exist.

**Step 3: Implement WasmV1**

`go/internal/lima/tag_blobs/wasm_v1.go`:

```go
package tag_blobs

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku_wasm"
)

type WasmV1 struct {
	sku_wasm.WasmVMPoolV1
	ctx context.Context
}

func (a *WasmV1) GetQueryable() sku.Queryable {
	return a
}

func (a *WasmV1) Reset() {
}

func (a *WasmV1) ResetWith(b WasmV1) {
}

func (tb *WasmV1) ContainsSku(tg sku.TransactedGetter) bool {
	vm, vmRepool := tb.GetWithRepool()
	defer vmRepool()

	recordPtr, err := sku_wasm.MarshalTransactedToModule(
		tb.ctx,
		vm.Module,
		tg,
	)
	if err != nil {
		ui.Err().Print(err)
		return false
	}

	result, err := vm.Module.CallContainsSku(tb.ctx, recordPtr)
	if err != nil {
		ui.Err().Print(err)
		return false
	}

	return result
}
```

**Step 4: Run tests**

Run: `cd go && go test -v -tags test,debug ./internal/lima/tag_blobs/`
Expected: PASS

**Step 5: Commit**

```bash
git add go/internal/lima/tag_blobs/wasm_v1.go go/internal/lima/tag_blobs/wasm_v1_test.go
git commit -m "feat(lima/tag_blobs): add WasmV1 queryable for WASM tag filters"
```

---

### Task 9: Integrate into typed blob store

**Files:**
- Modify: `go/internal/mike/typed_blob_store/tag.go:18-25` (add wasm_v1 field)
- Modify: `go/internal/mike/typed_blob_store/tag.go:27-81` (update MakeTagStore and GetBlob)

**Step 1: Add wasm_v1 field to Tag struct**

In `go/internal/mike/typed_blob_store/tag.go`, add to the `Tag` struct:

```go
type Tag struct {
	envRepo env_repo.Env
	envLua  env_lua.Env
	wasmRt  *wasm.Runtime
	toml_v0 domain_interfaces.TypedStore[tag_blobs.V0, *tag_blobs.V0]
	toml_v1 domain_interfaces.TypedStore[tag_blobs.TomlV1, *tag_blobs.TomlV1]
	lua_v1  domain_interfaces.TypedStore[tag_blobs.LuaV1, *tag_blobs.LuaV1]
	lua_v2  domain_interfaces.TypedStore[tag_blobs.LuaV2, *tag_blobs.LuaV2]
	wasm_v1 domain_interfaces.TypedStore[tag_blobs.WasmV1, *tag_blobs.WasmV1]
}
```

**Step 2: Update MakeTagStore signature**

Add `wasmRt *wasm.Runtime` parameter. Update `MakeStores` in
`go/internal/mike/typed_blob_store/main.go` to pass it through. Update the call
site in `go/internal/victor/local_working_copy/main.go:189` to create and pass
a WASM runtime.

**Step 3: Add GetBlob case for TypeWasmTagV1**

After the `TypeLuaTagV2` case (line 179), add:

```go
case ids.TypeWasmTagV1:
	var readCloser domain_interfaces.BlobReader

	if readCloser, err = store.envRepo.GetDefaultBlobStore().MakeBlobReader(
		blobId,
	); err != nil {
		err = errors.Wrap(err)
		return blobGeneric, repool, err
	}

	defer errors.DeferredCloser(&err, readCloser)

	var modulePool *wasm.ModulePool

	ctx := context.Background()

	builder := wasm.MakeModulePoolBuilder(store.wasmRt).WithReader(readCloser)

	if modulePool, err = builder.Build(ctx); err != nil {
		err = errors.Wrap(err)
		return blobGeneric, repool, err
	}

	blobGeneric = &tag_blobs.WasmV1{
		WasmVMPoolV1: sku_wasm.MakeWasmVMPoolV1(modulePool),
	}
```

**Step 4: Update MakeStores and its call site**

In `go/internal/mike/typed_blob_store/main.go`, update:

```go
func MakeStores(
	envRepo env_repo.Env,
	envLua env_lua.Env,
	wasmRt *wasm.Runtime,
	boxFormat *box_format.BoxTransacted,
) Stores {
	return Stores{
		// ...
		Tag:  MakeTagStore(envRepo, envLua, wasmRt),
		// ...
	}
}
```

In `go/internal/victor/local_working_copy/main.go:189`, create a WASM runtime
and pass it:

```go
// Add a wasmRuntime field to the local working copy struct
// Initialize it during setup
wasmRt, err := wasm.MakeRuntime(context.Background())
// ... error handling ...

local.typedBlobStore = typed_blob_store.MakeStores(
	local.envRepo,
	local.envLua,
	wasmRt,
	boxFormatArchive,
)
```

**Step 5: Verify compilation**

Run: `cd go && go build ./...`
Expected: clean build.

**Step 6: Commit**

```bash
git add go/internal/mike/typed_blob_store/ go/internal/victor/local_working_copy/
git commit -m "feat(mike/typed_blob_store): integrate WASM tag blob store"
```

---

### Task 10: Build and run full test suite

**Step 1: Run unit tests**

Run: `cd go && go test -v -tags test,debug ./lib/bravo/wasm/ ./internal/kilo/sku_wasm/ ./internal/lima/tag_blobs/`
Expected: all PASS.

**Step 2: Run full unit test suite**

Run: `just test-go`
Expected: all PASS — no regressions from the new wazero dependency or type
registration.

**Step 3: Build**

Run: `just build`
Expected: clean build.

**Step 4: Run integration tests**

Run: `just test-bats`
Expected: all PASS.

**Step 5: Commit any fixups**

If any tests needed adjustment, commit the fixes.

---

### Task 11: Create example Rust guest filter

**Files:**
- Create: `examples/wasm-filters/genre-filter/Cargo.toml`
- Create: `examples/wasm-filters/genre-filter/src/lib.rs`
- Create: `examples/wasm-filters/genre-filter/wit/` (symlink to `wit/`)

**Step 1: Initialize Rust project**

```bash
mkdir -p examples/wasm-filters/genre-filter
cd examples/wasm-filters/genre-filter
cargo init --lib
```

**Step 2: Add wit-bindgen dependency and configure**

`Cargo.toml`:

```toml
[package]
name = "genre-filter"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
wit-bindgen = "0.36"
```

**Step 3: Write the filter**

`src/lib.rs`:

```rust
wit_bindgen::generate!({
    world: "query-filter",
    path: "../../../wit",
});

struct GenreFilter;

impl Guest for GenreFilter {
    fn contains_sku(object: Sku) -> bool {
        object.genre == "zettel"
    }
}

export!(GenreFilter);
```

**Step 4: Build**

Run: `cd examples/wasm-filters/genre-filter && cargo build --target wasm32-wasip1 --release`
Expected: produces `target/wasm32-wasip1/release/genre_filter.wasm`

**Step 5: Commit**

```bash
git add examples/wasm-filters/
git commit -m "feat: add example Rust WASM genre filter using wit-bindgen"
```
