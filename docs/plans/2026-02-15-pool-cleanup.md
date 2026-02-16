# Pool Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Standardize all pool usage on `GetWithRepool()`, collapse interfaces, delete redundant code.

**Architecture:** Single `Pool[T]` interface with one method (`GetWithRepool`). Concrete types keep unexported `get()`/`put()` for internal data structure use (heap). All callers migrate to `element, repool := pool.GetWithRepool(); defer repool()`.

**Tech Stack:** Go generics, `sync.Pool`

---

### Task 1: Simplify pool interfaces

**Files:**
- Modify: `go/src/_/interfaces/pools.go`

**Step 1: Read the current file**

Read `go/src/_/interfaces/pools.go` to confirm current state.

**Step 2: Replace interface definitions**

Replace the entire file contents with:

```go
package interfaces

type FuncRepool func()

type Pool[T any] interface {
	GetWithRepool() (T, FuncRepool)
}

type PoolPtr[T any, TPtr Ptr[T]] interface {
	Pool[TPtr]
}
```

This removes: `PoolablePtr`, `PoolValue`, `PoolWithErrors`, `PoolWithErrorsPtr`.

**Step 3: Build to see what breaks**

Run: `cd go && go build ./...`

Expected: Many compilation errors across the codebase. This is expected — the
remaining tasks fix them all. Just confirm the interfaces file itself is valid
Go.

Run: `cd go && go build ./src/_/interfaces/`

Expected: PASS (the interfaces package itself should compile)

**Step 4: Commit**

```
Simplify pool interfaces to Pool[T] and PoolPtr[T, TPtr]
```

---

### Task 2: Unexport Get/Put on concrete pool types

**Files:**
- Modify: `go/src/alfa/pool/main.go`
- Modify: `go/src/alfa/pool/value.go`
- Modify: `go/src/alfa/pool/bespoke.go`
- Modify: `go/src/alfa/pool/fake_pool.go`

**Step 1: Modify `main.go`**

Rename `Get` to `get`, `Put` to `put`. Keep `GetWithRepool` public. Update the
`GetWithRepool` body to call `pool.get()` and `pool.put()`. Remove the
`interfaces.Pool` compile-time check (the old two-type-param interface no longer
exists). Add a new compile-time check for `interfaces.PoolPtr`.

```go
var _ interfaces.PoolPtr[string, *string] = pool[string, *string]{}

func (pool pool[SWIMMER, SWIMMER_PTR]) get() SWIMMER_PTR {
	return pool.inner.Get().(SWIMMER_PTR)
}

func (pool pool[SWIMMER, SWIMMER_PTR]) GetWithRepool() (SWIMMER_PTR, interfaces.FuncRepool) {
	element := pool.get()
	return element, func() {
		pool.put(element)
	}
}

func (pool pool[SWIMMER, SWIMMER_PTR]) put(swimmer SWIMMER_PTR) {
	if swimmer == nil {
		return
	}
	if pool.reset != nil {
		pool.reset(swimmer)
	}
	pool.inner.Put(swimmer)
}
```

Note: `put` no longer returns error (it was always nil).

**Step 2: Modify `value.go`**

Same treatment: `Get` -> `get`, `Put` -> `put`. Add compile-time check for
`interfaces.Pool`.

```go
var _ interfaces.Pool[string] = value[string]{}

func (pool value[SWIMMER]) get() SWIMMER {
	return pool.inner.Get().(SWIMMER)
}

func (pool value[SWIMMER]) GetWithRepool() (SWIMMER, interfaces.FuncRepool) {
	element := pool.get()
	return element, func() {
		pool.put(element)
	}
}

func (pool value[SWIMMER]) put(swimmer SWIMMER) {
	if pool.reset != nil {
		pool.reset(swimmer)
	}
	pool.inner.Put(swimmer)
}
```

**Step 3: Modify `bespoke.go`**

Rename `Get` -> `get`, `Put` -> `put`, update `GetWithRepool`. Also rename
struct fields `FuncGet` -> `funcGet`, `FuncPut` -> `funcPut` since callers
should only use `GetWithRepool`.

Wait — `Bespoke` is used in `juliett/sku/type_checked_out.go` where `FuncGet`
and `FuncPut` are set by the caller. Keep the fields exported but rename methods:

```go
func (ip Bespoke[T]) get() T {
	return ip.FuncGet()
}

func (pool Bespoke[SWIMMER]) GetWithRepool() (SWIMMER, interfaces.FuncRepool) {
	element := pool.get()
	return element, func() {
		pool.put(element)
	}
}

func (ip Bespoke[T]) put(i T) {
	ip.FuncPut(i)
}
```

**Step 4: Modify `fake_pool.go`**

Rename `Get` -> `get`, `Put` -> `put`. Remove `PutMany`. Update compile-time
check.

```go
var _ interfaces.PoolPtr[string, *string] = fakePool[string, *string]{}

func (pool fakePool[T, TPtr]) get() TPtr {
	var t T
	return &t
}

func (pool fakePool[SWIMMER, SWIMMER_PTR]) GetWithRepool() (SWIMMER_PTR, interfaces.FuncRepool) {
	element := pool.get()
	return element, func() {}
}

func (pool fakePool[T, TPtr]) put(i TPtr) {}
```

**Step 5: Build the pool package**

Run: `cd go && go build ./src/alfa/pool/`

Expected: PASS

**Step 6: Commit**

```
Unexport Get/Put on concrete pool types
```

---

### Task 3: Add Slice pool and delete `_/pool_value/`

**Files:**
- Create: `go/src/alfa/pool/slice.go`
- Modify: `go/src/alfa/pool/common.go` (remove `_/pool_value` import)
- Modify: `go/src/alfa/errors/group.go` (switch to `alfa/pool`)
- Delete: `go/src/_/pool_value/main.go`
- Delete: `go/src/_/pool_value/slice.go`

**Step 1: Create `go/src/alfa/pool/slice.go`**

```go
package pool

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

type Slice[SWIMMER any, SWIMMER_SLICE ~[]SWIMMER] struct {
	inner *sync.Pool
}

func MakeSlice[SWIMMER any, SWIMMER_SLICE ~[]SWIMMER]() Slice[SWIMMER, SWIMMER_SLICE] {
	return Slice[SWIMMER, SWIMMER_SLICE]{
		inner: &sync.Pool{
			New: func() any {
				swimmer := make(SWIMMER_SLICE, 0)
				return swimmer
			},
		},
	}
}

func (pool Slice[_, SWIMMER_SLICE]) get() SWIMMER_SLICE {
	return pool.inner.Get().(SWIMMER_SLICE)
}

func (pool Slice[_, SWIMMER_SLICE]) GetWithRepool() (SWIMMER_SLICE, interfaces.FuncRepool) {
	element := pool.get()
	return element, func() {
		pool.put(element)
	}
}

func (pool Slice[_, SWIMMER_SLICE]) put(swimmer SWIMMER_SLICE) {
	swimmer = swimmer[:0]
	pool.inner.Put(swimmer)
}
```

**Step 2: Update `go/src/alfa/pool/common.go`**

Replace `pool_value.Make(...)` for `sha256Hash` with `MakeValue(...)`. Remove
the `_/pool_value` import. Also update all helper functions to use
`GetWithRepool` internally instead of manual `Get`/`Put`:

```go
package pool

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"hash"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

var (
	bufioReader   = Make[bufio.Reader](nil, nil)
	bufioWriter   = Make[bufio.Writer](nil, nil)
	byteReaders   = Make[bytes.Reader](nil, nil)
	stringReaders = Make[strings.Reader](nil, nil)
	sha256Hash    = MakeValue(
		func() hash.Hash {
			return sha256.New()
		},
		func(hash hash.Hash) {
			hash.Reset()
		},
	)
)

func GetStringReader(
	value string,
) (stringReader *strings.Reader, repool interfaces.FuncRepool) {
	stringReader, repool = stringReaders.GetWithRepool()
	stringReader.Reset(value)
	return stringReader, repool
}

func GetByteReader(
	value []byte,
) (byteReader *bytes.Reader, repool interfaces.FuncRepool) {
	byteReader, repool = byteReaders.GetWithRepool()
	byteReader.Reset(value)
	return byteReader, repool
}

func GetSha256Hash() (hash hash.Hash, repool interfaces.FuncRepool) {
	hash, repool = sha256Hash.GetWithRepool()
	return hash, repool
}

func GetBufferedWriter(
	writer io.Writer,
) (bufferedWriter *bufio.Writer, repool interfaces.FuncRepool) {
	bufferedWriter, repool = bufioWriter.GetWithRepool()
	bufferedWriter.Reset(writer)
	return bufferedWriter, repool
}

func GetBufferedReader(
	reader io.Reader,
) (bufferedReader *bufio.Reader, repool interfaces.FuncRepool) {
	bufferedReader, repool = bufioReader.GetWithRepool()
	bufferedReader.Reset(reader)
	return bufferedReader, repool
}
```

**Step 3: Update `go/src/alfa/errors/group.go`**

Change import from `_/pool_value` to `alfa/pool` and update the pool
construction:

```go
package errors

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/pool"
)

type Group []error

func (group Group) Error() string {
	return fmt.Sprintf("error group: %d errors", group.Len())
}

func (group Group) Unwrap() []error {
	return group
}

func (group Group) Len() int {
	return len(group)
}

var groupPool = pool.MakeSlice[error, Group]()
```

**Step 4: Delete `_/pool_value/` package**

Delete `go/src/_/pool_value/main.go` and `go/src/_/pool_value/slice.go`.

**Step 5: Build**

Run: `cd go && go build ./src/alfa/pool/ ./src/alfa/errors/`

Expected: PASS

**Step 6: Commit**

```
Move slice pool to alfa/pool, delete _/pool_value package
```

---

### Task 4: Delete poolWithError and migrate Lua VM pool

**Files:**
- Delete: `go/src/alfa/pool/with_error.go`
- Modify: `go/src/bravo/lua/vm.go`
- Modify: `go/src/bravo/lua/vm_pool.go`
- Modify: `go/src/bravo/lua/vm_pool_builder.go`
- Modify: `go/src/kilo/sku_lua/lua_transacted_v1_pool.go`
- Modify: `go/src/kilo/sku_lua/lua_transacted_v2_pool.go`

**Step 1: Modify `bravo/lua/vm_pool.go`**

Change `VMPool` to embed `interfaces.PoolPtr[VM, *VM]` instead of
`interfaces.PoolWithErrorsPtr[VM, *VM]`. In `SetCompiled`, use `pool.Make`
instead of `pool.MakeWithError`. The `New` function panics on error (same
behavior — `MakeWithError` already panicked):

```go
type VMPool struct {
	interfaces.PoolPtr[VM, *VM]
	Require  LGFunction
	Searcher LGFunction
	compiled *lua.FunctionProto
}
```

In `SetCompiled`, replace `sp.PoolWithErrorsPtr = pool.MakeWithError(...)` with:

```go
sp.PoolPtr = pool.Make(
	func() (vm *VM) {
		vm = &VM{
			LState: lua.NewState(),
		}

		if err := sp.PrepareVM(vm, apply); err != nil {
			panic(errors.Wrap(err))
		}

		return vm
	},
	func(vm *VM) {
		vm.SetTop(0)
	},
)
```

In `PrepareVM`, change `vm.Pool.Get()` calls (lines 68, 82) to use
`GetWithRepool`. The tables obtained here live as long as the VM, so the repool
func is unused but harmless:

```go
table, _ := vm.Pool.GetWithRepool()
```

And line 82:

```go
searcherTable, _ := vm.Pool.GetWithRepool()
```

**Step 2: Modify `bravo/lua/vm.go`**

Change the embedded pool type:

```go
type VM struct {
	*lua.LState
	Top lua.LValue
	interfaces.PoolPtr[LTable, *LTable]
}
```

**Step 3: Modify `kilo/sku_lua/lua_transacted_v1_pool.go`**

Change the type alias:

```go
type (
	LuaVMPoolV1    interfaces.PoolPtr[LuaVMV1, *LuaVMV1]
	LuaTablePoolV1 = interfaces.PoolPtr[LuaTableV1, *LuaTableV1]
)
```

Update `PushTopFuncV1` — change error-returning Get to GetWithRepool:

```go
func PushTopFuncV1(
	lvm LuaVMPoolV1,
	args []string,
) (vm *LuaVMV1, argsOut []string, err error) {
	var repool interfaces.FuncRepool
	vm, repool = lvm.GetWithRepool()
	_ = repool // VM lifecycle managed by caller
	// ... rest unchanged
}
```

Update `MakeLuaVMPoolV1` to use `pool.Make` with panic:

```go
func MakeLuaVMPoolV1(vmPool *lua.VMPool, self *sku.Transacted) LuaVMPoolV1 {
	return pool.Make(
		func() (out *LuaVMV1) {
			vm, repool := vmPool.PoolPtr.GetWithRepool()
			_ = repool // VM lifecycle managed by LuaVMV1

			out = &LuaVMV1{
				VM:        vm,
				TablePool: MakeLuaTablePoolV1(vm),
				Selbst:    self,
			}

			return out
		},
		nil,
	)
}
```

Update `MakeLuaTablePoolV1` — change `vm.Pool.Get()` to `vm.Pool.GetWithRepool()`:

```go
func MakeLuaTablePoolV1(vm *lua.VM) LuaTablePoolV1 {
	return pool.Make(
		func() (table *LuaTableV1) {
			transacted, _ := vm.Pool.GetWithRepool()
			tags, _ := vm.Pool.GetWithRepool()
			tagsImplicit, _ := vm.Pool.GetWithRepool()

			table = &LuaTableV1{
				Transacted:   transacted,
				Tags:         tags,
				TagsImplicit: tagsImplicit,
			}

			vm.SetField(table.Transacted, "Etiketten", table.Tags)
			vm.SetField(table.Transacted, "EtikettenImplicit", table.TagsImplicit)

			return table
		},
		func(t *LuaTableV1) {
			lua.ClearTable(vm.LState, t.Tags)
			lua.ClearTable(vm.LState, t.TagsImplicit)
		},
	)
}
```

**Step 4: Apply same changes to `kilo/sku_lua/lua_transacted_v2_pool.go`**

Mirror the V1 changes for V2.

**Step 5: Check `vm_pool_builder.go` for references**

Read `go/src/bravo/lua/vm_pool_builder.go` and update any `PoolWithErrorsPtr`
references or `.Get()` calls that return errors.

**Step 6: Delete `go/src/alfa/pool/with_error.go`**

**Step 7: Build**

Run: `cd go && go build ./src/bravo/lua/ ./src/kilo/sku_lua/`

Expected: PASS

**Step 8: Commit**

```
Remove PoolWithErrors, migrate Lua VM pools to panic semantics
```

---

### Task 5: Migrate Alfred ItemPool to alfa/pool

**Files:**
- Modify: `go/src/delta/alfred/item_pool.go`

**Step 1: Read current callers of ItemPool**

Search for `ItemPool` usage to understand callers.

**Step 2: Rewrite `item_pool.go`**

```go
package alfred

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
)

type ItemPool struct {
	inner *pool.pool[Item, *Item]
}

// Wait — pool.pool is unexported. ItemPool needs to use the public API.
// Rewrite to embed the pool interface instead:
```

Actually, since `pool.pool` is unexported, `ItemPool` should just wrap the
public interface:

```go
package alfred

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
)

var itemPool = pool.Make(
	func() *Item {
		return &Item{
			Match: &catgut.String{},
			Mods:  make(map[string]Mod),
		}
	},
	func(i *Item) {
		i.Reset()
	},
)

func GetItemPool() interfaces.PoolPtr[Item, *Item] {
	return itemPool
}
```

Then update callers to use `GetItemPool().GetWithRepool()` instead of
`pool.Get()` / `pool.Put()`.

**Step 3: Find and update callers**

Search for `ItemPool` in the codebase and update call sites from:
```go
item := pool.Get()
defer pool.Put(item)
```
to:
```go
item, repool := alfred.GetItemPool().GetWithRepool()
defer repool()
```

**Step 4: Build**

Run: `cd go && go build ./src/delta/alfred/`

Expected: PASS

**Step 5: Commit**

```
Migrate Alfred ItemPool to alfa/pool
```

---

### Task 6: Consolidate store pools with proper resetters

**Files:**
- Modify: `go/src/papa/store_fs/pools.go`
- Modify: `go/src/sierra/store_browser/pools.go`
- Modify: `go/src/tango/store/pools.go`

**Step 1: Fix `papa/store_fs/pools.go`**

Add proper resetters and update interface types:

```go
package store_fs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

var (
	poolExternal = pool.Make[sku.Transacted](
		nil,
		sku.TransactedResetter.Reset,
	)

	poolCheckedOut = pool.Make[sku.CheckedOut](
		nil,
		sku.CheckedOutResetter.Reset,
	)
)

func GetExternalPool() interfaces.PoolPtr[sku.Transacted, *sku.Transacted] {
	return poolExternal
}

func GetCheckedOutPool() interfaces.PoolPtr[sku.CheckedOut, *sku.CheckedOut] {
	return poolCheckedOut
}
```

**Step 2: Apply same fix to `sierra/store_browser/pools.go`**

Same content, different package name.

**Step 3: Update `tango/store/pools.go`**

The `PutCheckedOutLike` function calls `Put()` directly. With `Put` now
unexported, this function needs to be rethought. Since the design says callers
handle their own repooling, this dispatcher function may no longer be needed.
Check its callers and determine if it can be deleted (the TODO comment on line 9
says "TODO remove entirely").

If callers already defer their own repool, delete `PutCheckedOutLike` and its
callers' `Put` calls.

**Step 4: Build**

Run: `cd go && go build ./src/papa/store_fs/ ./src/sierra/store_browser/ ./src/tango/store/`

Expected: PASS

**Step 5: Commit**

```
Consolidate store pools with proper resetters
```

---

### Task 7: Update juliett/sku pool accessors and types

**Files:**
- Modify: `go/src/juliett/sku/pools.go`
- Modify: `go/src/juliett/sku/transacted.go` (CloneTransacted)
- Modify: `go/src/juliett/sku/checked_out.go` (Clone)
- Modify: `go/src/juliett/sku/type_checked_out.go` (objectFactoryCheckedOut)

**Step 1: Update `pools.go` return types**

```go
func GetTransactedPool() interfaces.PoolPtr[Transacted, *Transacted] {
	return poolTransacted
}

func GetCheckedOutPool() interfaces.PoolPtr[CheckedOut, *CheckedOut] {
	return poolCheckedOut
}
```

**Step 2: Update `transacted.go` CloneTransacted**

`CloneTransacted` allocates and returns ownership. Use `GetWithRepool` but don't
defer — caller owns the lifecycle:

```go
func (transacted *Transacted) CloneTransacted() (cloned *Transacted, repool interfaces.FuncRepool) {
	cloned, repool = GetTransactedPool().GetWithRepool()
	TransactedResetter.ResetWith(cloned, transacted)
	return cloned, repool
}
```

This changes the signature — all callers of `CloneTransacted` must be updated to
handle the repool return value.

**Step 3: Update `checked_out.go` Clone**

Same pattern:

```go
func (checkedOut *CheckedOut) Clone() (dst *CheckedOut, repool interfaces.FuncRepool) {
	dst, repool = GetCheckedOutPool().GetWithRepool()
	CheckedOutResetter.ResetWith(dst, checkedOut)
	return dst, repool
}
```

**Step 4: Update `type_checked_out.go`**

Change `objectFactoryCheckedOut` to embed `interfaces.Pool[*CheckedOut]` instead
of `interfaces.PoolValue[*CheckedOut]`:

```go
type objectFactoryCheckedOut struct {
	interfaces.Pool[*CheckedOut]
	interfaces.Resetter[*CheckedOut]
}
```

Update `SetDefaultsIfNecessary` — the `Bespoke` now uses `GetWithRepool`
internally so this should work. But `FuncGet`/`FuncPut` on Bespoke need to
change. Actually, since `Bespoke` fields are still exported but the methods
`get()`/`put()` are unexported, callers still set `FuncGet`/`FuncPut` and
`Bespoke.GetWithRepool()` calls them internally. This still works.

Update `makeCheckedOut` and `cloneFromTransactedCheckedOut` to use
`GetWithRepool`:

```go
func makeCheckedOut() (*CheckedOut, interfaces.FuncRepool) {
	return GetCheckedOutPool().GetWithRepool()
}

func cloneFromTransactedCheckedOut(
	src *Transacted,
	newState checked_out_state.State,
) (*CheckedOut, interfaces.FuncRepool) {
	dst, repool := GetCheckedOutPool().GetWithRepool()
	TransactedResetter.ResetWith(dst.GetSku(), src)
	TransactedResetter.ResetWith(dst.GetSkuExternal(), src)
	dst.state = newState
	return dst, repool
}
```

**Step 5: Find and update all callers of CloneTransacted and Clone**

These signature changes cascade. Search for `.CloneTransacted()` and `.Clone()`
on CheckedOut types across the codebase and add the repool return handling.

**Step 6: Build**

Run: `cd go && go build ./src/juliett/sku/`

Expected: PASS (or cascade errors from callers — fix in Task 8)

**Step 7: Commit**

```
Update juliett/sku pool accessors to PoolPtr interface
```

---

### Task 8: Update heap to use unexported get/put

**Files:**
- Modify: `go/src/charlie/heap/private.go`
- Modify: `go/src/charlie/heap/main.go`

**Step 1: Understand the constraint**

The heap stores a `interfaces.Pool[ELEMENT, ELEMENT_PTR]` field. With the new
interface, this becomes `interfaces.PoolPtr[ELEMENT, ELEMENT_PTR]` which only
has `GetWithRepool()`. But the heap needs `get()` and `put()` directly.

Option: The heap stores a concrete `*pool.pool[E, EP]` instead of the interface.
But `pool.pool` is unexported.

Better option: The heap stores the interface and uses `GetWithRepool()`. For
`Pop()`, it calls `GetWithRepool()` and discards the repool func (the caller
owns the element). For `restore()`/`Reset()`, it stores a `FuncRepool` alongside
`lastPopped` and calls it.

**Step 2: Modify `charlie/heap/private.go`**

Change the pool field type and add a `lastPoppedRepool` field:

```go
type heapPrivate[ELEMENT Element, ELEMENT_PTR ElementPtr[ELEMENT]] struct {
	Lessor  interfaces.Lessor[ELEMENT_PTR]
	equaler interfaces.Equaler[ELEMENT_PTR]

	Resetter        interfaces.ResetterPtr[ELEMENT, ELEMENT_PTR]
	Elements        []ELEMENT_PTR
	lastPopped      ELEMENT_PTR
	lastPoppedRepool interfaces.FuncRepool
	pool            interfaces.PoolPtr[ELEMENT, ELEMENT_PTR]
}
```

Update `GetPool()` to return the new type:

```go
func (heap *heapPrivate[ELEMENT, ELEMENT_PTR]) GetPool() interfaces.PoolPtr[ELEMENT, ELEMENT_PTR] {
	if heap.pool == nil {
		heap.pool = pool.MakeFakePool[ELEMENT, ELEMENT_PTR]()
	}
	return heap.pool
}
```

Update `saveLastPopped`:

```go
func (heap *heapPrivate[ELEMENT, ELEMENT_PTR]) saveLastPopped(e ELEMENT_PTR) {
	if heap.lastPopped == nil {
		heap.lastPopped, heap.lastPoppedRepool = heap.GetPool().GetWithRepool()
	}
	heap.Resetter.ResetWith(heap.lastPopped, e)
}
```

**Step 3: Modify `charlie/heap/main.go`**

Update `SetPool`:

```go
func (heap *Heap[ELEMENT, ELEMENT_PTR]) SetPool(
	v interfaces.PoolPtr[ELEMENT, ELEMENT_PTR],
) {
	heap.private.pool = v
}
```

Update `Pop()` and `popAndSave()` — use `GetWithRepool`, discard repool:

```go
func (heap *Heap[ELEMENT, ELEMENT_PTR]) Pop() (element ELEMENT_PTR, ok bool) {
	heap.lock.Lock()
	defer heap.lock.Unlock()

	if heap.private.Len() == 0 {
		return element, ok
	}

	element, _ = heap.private.GetPool().GetWithRepool()
	heap.private.Resetter.ResetWith(
		element,
		pkg_heap.Pop(&heap.private).(ELEMENT_PTR),
	)
	ok = true
	heap.private.saveLastPopped(element)

	return element, ok
}
```

Same for `popAndSave()`.

Update `restore()` — call the stored repool func:

```go
func (heap *Heap[ELEMENT, ELEMENT_PTR]) restore() {
	heap.private.Elements = heap.private.Elements[:heap.savedIndex]
	heap.savedIndex = 0
	if heap.private.lastPoppedRepool != nil {
		heap.private.lastPoppedRepool()
		heap.private.lastPoppedRepool = nil
	}
	heap.private.lastPopped = nil

	quiter.ReverseSortable(&heap.private)
}
```

Update `Reset()`:

```go
func (heap *Heap[ELEMENT, ELEMENT_PTR]) Reset() {
	heap.private.Elements = make([]ELEMENT_PTR, 0)
	if heap.private.lastPoppedRepool != nil {
		heap.private.lastPoppedRepool()
		heap.private.lastPoppedRepool = nil
	}
	heap.private.pool = nil
	heap.private.lastPopped = nil
}
```

Update `ResetWith`:

```go
func (heap *Heap[ELEMENT, ELEMENT_PTR]) ResetWith(
	b *Heap[ELEMENT, ELEMENT_PTR],
) {
	heap.private.equaler = b.private.equaler
	heap.private.Lessor = b.private.Lessor
	heap.private.Resetter = b.private.Resetter
	heap.private.Elements = make([]ELEMENT_PTR, b.Len())
	copy(heap.private.Elements, b.private.Elements)
	heap.private.pool = b.private.GetPool()
}
```

**Step 4: Build**

Run: `cd go && go build ./src/charlie/heap/`

Expected: PASS

**Step 5: Commit**

```
Update heap to use GetWithRepool pattern
```

---

### Task 9: Migrate catgut String pool

**Files:**
- Modify: `go/src/charlie/catgut/pool.go`

**Step 1: Read current file**

Read `go/src/charlie/catgut/pool.go`.

**Step 2: Update return type**

```go
func GetPool() interfaces.PoolPtr[String, *String] {
	ponce.Do(func() {
		p = pool.Make[String, *String](nil, func(v *String) { v.Reset() })
	})
	return p
}
```

Update the `p` variable type accordingly.

**Step 3: Find callers of catgut.GetPool() and migrate**

Search for `catgut.GetPool()` and update Get/Put patterns.

**Step 4: Build and commit**

```
Update catgut String pool to PoolPtr interface
```

---

### Task 10: Mechanical caller migration — tango/store

**Files:**
- Modify: `go/src/tango/store/create.go`
- Modify: `go/src/tango/store/reader.go`
- Modify: `go/src/tango/store/mutating.go`
- Modify: `go/src/tango/store/flush.go`
- Modify: `go/src/tango/store/dormancy_and_tags.go`

For each file, apply the mechanical transformation:

Before:
```go
object := sku.GetTransactedPool().Get()
defer sku.GetTransactedPool().Put(object)
```

After:
```go
object, repool := sku.GetTransactedPool().GetWithRepool()
defer repool()
```

For non-deferred Put calls (error paths in create.go, reader.go, mutating.go),
restructure to use defer instead. If the function Get+defer is already at the
top, the non-deferred Puts become unnecessary.

**Build and commit:**

```
Migrate tango/store pool callers to GetWithRepool
```

---

### Task 11: Mechanical caller migration — papa/store_fs

**Files:**
- Modify: `go/src/papa/store_fs/checkout.go`
- Modify: `go/src/papa/store_fs/checkout2.go`
- Modify: `go/src/papa/store_fs/dir_info.go`
- Modify: `go/src/papa/store_fs/merge.go`
- Modify: `go/src/papa/store_fs/query.go`
- Modify: `go/src/papa/store_fs/read_checked_out.go`
- Modify: `go/src/papa/store_fs/read_external.go`
- Modify: `go/src/papa/store_fs/reading.go`

Same mechanical transformation. Pay attention to files where Get() returns
ownership (checkout2.go, read_checked_out.go, read_external.go) — these return
both the object and the repool func.

**Build and commit:**

```
Migrate papa/store_fs pool callers to GetWithRepool
```

---

### Task 12: Mechanical caller migration — remaining modules

**Files:**
- Modify: `go/src/november/queries/executor.go`
- Modify: `go/src/november/queries/build_state.go`
- Modify: `go/src/lima/stream_index/probe_index.go`
- Modify: `go/src/lima/stream_index/page_reader_stream.go`
- Modify: `go/src/lima/stream_index/page_additions_file.go`
- Modify: `go/src/lima/stream_index/probes.go`
- Modify: `go/src/lima/stream_index/binary_test.go`
- Modify: `go/src/lima/inventory_list_coders/main.go`
- Modify: `go/src/lima/env_lua/main.go`
- Modify: `go/src/mike/inventory_list_store/main.go`
- Modify: `go/src/victor/local_working_copy/format.go`
- Modify: `go/src/victor/local_working_copy/genesis.go`
- Modify: `go/src/sierra/store_browser/main.go`
- Modify: `go/src/sierra/store_browser/index.go`
- Modify: `go/src/uniform/remote_transfer/import.go`
- Modify: `go/src/uniform/remote_transfer/main.go`
- Modify: `go/src/whiskey/user_ops/create_from_shas.go`
- Modify: `go/src/whiskey/remote_http/server.go`
- Modify: `go/src/whiskey/remote_http/server_mcp.go`
- Modify: `go/src/xray/command_components_dodder/remote.go`
- Modify: `go/src/yankee/commands_dodder/clean.go`
- Modify: `go/src/yankee/commands_dodder/cat_alfred.go`
- Modify: `go/src/yankee/commands_dodder/edit_config.go`
- Modify: `go/src/yankee/commands_dodder/revert.go`
- Modify: `go/src/yankee/commands_dodder/last.go`
- Modify: `go/src/foxtrot/tag_paths/main.go`
- Modify: `go/src/foxtrot/tag_paths/tags_with_parents_and_types.go`
- Modify: `go/src/echo/ids/main.go`
- Modify: `go/src/echo/ids/object_id3.go`
- Modify: `go/src/romeo/store_config/accessors.go`
- Modify: `go/src/romeo/store_config/main_test.go`
- Modify: `go/src/romeo/store_config/compiled.go`

Same mechanical transformation. This is the largest task — can be split across
multiple parallel subagents by NATO module grouping.

**Build and commit per module group:**

```
Migrate lima pool callers to GetWithRepool
Migrate november pool callers to GetWithRepool
Migrate victor pool callers to GetWithRepool
...etc
```

---

### Task 13: Delete remaining PoolValue references

**Files:**
- Search all files for `PoolValue` or `interfaces.PoolValue` references
- Update any remaining type annotations

Any file still referencing `interfaces.PoolValue` needs to switch to
`interfaces.Pool[T]`. The `objectFactoryCheckedOut` in
`juliett/sku/type_checked_out.go` was already handled in Task 7.

**Build and commit:**

```
Remove all remaining PoolValue references
```

---

### Task 14: Full build and test

**Step 1: Full build**

Run: `cd go && go build ./...`

Expected: PASS with zero errors

**Step 2: Unit tests**

Run: `just test-go`

Expected: PASS

**Step 3: Integration tests**

Run: `just test-bats`

Expected: PASS (may need fixture regeneration if pool behavior changes affected
serialization — unlikely but check)

**Step 4: Format**

Run: `just codemod-go-fmt`

**Step 5: Final commit if formatting changed**

```
Format code after pool cleanup
```

---

### Task 15: Clean up CLAUDE.md files

**Files:**
- Modify: `go/src/alfa/pool/CLAUDE.md`
- Modify: `go/src/_/interfaces/CLAUDE.md`
- Delete: `go/src/_/pool_value/CLAUDE.md`

Update the CLAUDE.md files to reflect the new simplified pool API.

**Commit:**

```
Update CLAUDE.md files for pool cleanup
```
