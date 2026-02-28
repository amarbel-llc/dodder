# WASM Queryable Design

## Summary

Add a new `sku.Queryable` implementation backed by WebAssembly modules, enabling
query filters written in any language that compiles to WASM. Uses wazero (pure
Go, no CGo), a WIT-defined contract with wit-bindgen for guest-side bindings,
and a minimal canonical ABI shim on the Go host side.

## Motivation

- **Language-agnostic**: query filters in Rust, C, AssemblyScript, etc.
- **Sandboxing**: WASM provides memory isolation and capability-based security
- **Performance**: wazero's compiler mode offers near-native execution
- **Future platform target**: foundation for compiling dodder to WASM
- **Module system**: WIT provides a formal, versioned interface contract

## WIT Contract

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

Guest authors use `wit-bindgen` to generate typed bindings in their language and
implement `contains-sku`. The `.wit` file is the source of truth for the
host-guest contract.

## Package Layout

| Package | Tier | Role |
|---------|------|------|
| `lib/bravo/wasm/` | bravo | Domain-agnostic wazero wrapper: Runtime, Module, ModulePool, ModulePoolBuilder |
| `internal/kilo/sku_wasm/` | kilo | SKU-to-WASM marshaling, canonical ABI string helpers, WasmVMV1 pool type |
| `internal/lima/tag_blobs/wasm_v1.go` | lima | WasmV1 tag blob implementing Queryable and Blob |

### lib/bravo/wasm/ — Core Types

```go
type Runtime struct {
    runtime wazero.Runtime
}

type Module struct {
    mod         api.Module
    memory      api.Memory
    containsSku api.Function  // cached export reference
    cabiRealloc api.Function  // cached export reference
    reset       api.Function  // cached export reference
}

type ModulePool struct {
    interfaces.PoolPtr[Module, *Module]
    compiled wazero.CompiledModule
    runtime  *Runtime
}

type ModulePoolBuilder struct {
    // Mirrors lua.VMPoolBuilder
    // Accepts io.Reader (raw .wasm bytes) or pre-compiled module
}
```

### internal/kilo/sku_wasm/

```go
type WasmVMV1 struct {
    *wasm.Module
}

type WasmVMPoolV1 = interfaces.PoolPtr[WasmVMV1, *WasmVMV1]

func MakeWasmVMPoolV1(modulePool *wasm.ModulePool) WasmVMPoolV1
func ToWasmSku(tg sku.TransactedGetter, mod *wasm.Module) (ptr uint32, err error)
```

### internal/lima/tag_blobs/wasm_v1.go

```go
type WasmV1 struct {
    sku_wasm.WasmVMPoolV1
}

func (a *WasmV1) GetQueryable() sku.Queryable { return a }
func (a *WasmV1) Reset() {}
func (a *WasmV1) ResetWith(b WasmV1) {}
func (tb *WasmV1) ContainsSku(tg sku.TransactedGetter) bool { ... }
```

## Type Registration

In `internal/echo/ids/types_builtin.go`:

```go
TypeWasmTagV1 = "!wasm-tag-v1"
```

Registered with `registerBuiltinTypeString(TypeWasmTagV1, genres.Tag, false)`.

## Typed Blob Store Integration

In `internal/mike/typed_blob_store/tag.go`:

- Add `wasm_v1` field: `domain_interfaces.TypedStore[tag_blobs.WasmV1, *tag_blobs.WasmV1]`
- Add `case ids.TypeWasmTagV1:` in `GetBlob()`:
  1. Read blob via `MakeBlobReader` (raw `.wasm` bytes)
  2. Build `ModulePool` via `ModulePoolBuilder.WithReader(readCloser).Build()`
  3. Return `&tag_blobs.WasmV1{WasmVMPoolV1: ...}`

## Canonical ABI Subset

Only the subset needed for the SKU record:

- **String lowering**: `(ptr: i32, len: i32)` in linear memory, UTF-8
- **List lowering**: `(ptr: i32, count: i32)`, each element is a lowered string
- **Record lowering**: fields laid out sequentially
- **Bool lifting**: `i32` return, 0 = false, non-zero = true
- **Memory allocation**: guest exports `cabi_realloc(old_ptr, old_size, align, new_size) -> ptr`

No variants, options, or nested records needed.

## ContainsSku Execution Flow

```
ContainsSku(tg TransactedGetter) bool
  ├─ module, repool := pool.GetWithRepool()
  ├─ defer repool()
  ├─ Marshal SKU → WASM linear memory via canonical ABI:
  │    ├─ Allocate via guest-exported cabi_realloc
  │    ├─ Write flat record (5 strings + 2 lists = 18 i32s)
  │    └─ String data written as UTF-8 bytes
  ├─ Call guest-exported contains-sku(ptr) -> i32
  └─ Return result != 0
```

## Pool Lifecycle

```
wazero.Runtime (shared, long-lived, one per process)
  └─ wazero.CompiledModule (compiled once from .wasm bytes)
       └─ ModulePool (interfaces.PoolPtr[Module, *Module])
            ├─ Get: instantiate module from compiled code
            └─ Repool: call guest-exported reset() to clear arena
```

- Each pooled module retains its linear memory across borrows
- Arena-style reset: guest exports `reset()` that resets bump allocator to 0
- Mirrors Lua's `vm.SetTop(0)` pattern on repool

## Not In Scope (V1)

- Cross-module imports (WASM `require` equivalent)
- Store hooks (`on_new`) in WASM
- Direct WASM script execution (`exec_lua` equivalent)
- WASI filesystem/network access from within query filters
- Compiling dodder itself to WASM

## Deliverables

1. `.wit` file defining the `query-filter` world
2. `lib/bravo/wasm/` — Runtime, Module, ModulePool, ModulePoolBuilder
3. `internal/kilo/sku_wasm/` — WasmVMV1, WasmVMPoolV1, ToWasmSku
4. `internal/lima/tag_blobs/wasm_v1.go` — WasmV1 implementing Queryable
5. `internal/echo/ids/types_builtin.go` — TypeWasmTagV1 registration
6. `internal/mike/typed_blob_store/tag.go` — GetBlob case for WASM
7. Example guest filter in Rust (with wit-bindgen) for testing

## Testing

- Unit tests in `lib/bravo/wasm/` with a test `.wasm` module
- Unit tests in `internal/kilo/sku_wasm/` for SKU marshaling round-trips
- Unit tests in `internal/lima/tag_blobs/` for WasmV1.ContainsSku
- Integration test: tag zettel with type `!wasm-tag-v1`, store `.wasm` blob,
  query against it
