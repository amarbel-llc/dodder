package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// ListTransformV1 is the Go-side backing for the `dodder.list()` binding of
// the inventory-list transform plugin (FDR-0024 / RFC-0008 §3). It projects
// the input objects via ToLuaTableV1 (the V1 projection carries the Fields
// table the transform write-back needs; V2 has no fields projection) and
// tracks membership mutations (remove/add) so the command can read the
// output set back after the script returns.
type ListTransformV1 struct {
	vm        *lua.VM
	tablePool LuaTablePoolV1

	handle        *lua.LTable
	entries       []listTransformEntryV1
	handleToIndex map[*lua.LTable]int

	repools []func()
}

type listTransformEntryV1 struct {
	object  *sku.Transacted
	table   *LuaTableV1
	removed bool
}

// MakeListTransformV1 projects objects into per-object LuaTableV1 handles
// and builds the list handle table exposing each()/remove()/add(). Register
// the result via RegisterGlobals before the script runs; call Repool when
// done with the VM.
func MakeListTransformV1(
	vm *lua.VM,
	objects []*sku.Transacted,
) (binding *ListTransformV1) {
	binding = &ListTransformV1{
		vm:            vm,
		tablePool:     MakeLuaTablePoolV1(vm),
		handle:        vm.NewTable(),
		handleToIndex: make(map[*lua.LTable]int, len(objects)),
		repools:       make([]func(), 0, len(objects)),
	}

	for _, object := range objects {
		binding.appendObject(object)
	}

	vm.SetField(binding.handle, "each", vm.NewFunction(binding.luaEach))
	vm.SetField(binding.handle, "remove", vm.NewFunction(binding.luaRemove))
	vm.SetField(binding.handle, "add", vm.NewFunction(binding.luaAdd))

	return binding
}

func (binding *ListTransformV1) appendObject(
	object *sku.Transacted,
) (table *LuaTableV1) {
	table, repool := binding.tablePool.GetWithRepool() //repool:owned
	binding.repools = append(binding.repools, repool)

	ToLuaTableV1(object, binding.vm.LState, table)

	// Transform-only projection: the blob digest, so a script can point an
	// object at a blob it (re)wrote via blobs.write (the hash-migration
	// composition FDR-0024 motivates). Deliberately NOT part of
	// ToLuaTableV1 -- hook scripts must not see a blob mutation surface
	// (RFC-0006 Phase 2 gate, issue #319). FromLuaTableTransformV1 reads
	// it back.
	binding.vm.SetField(
		table.Transacted,
		"Blob",
		lua.LString(object.GetBlobDigest().String()),
	)

	binding.handleToIndex[table.Transacted] = len(binding.entries)
	binding.entries = append(binding.entries, listTransformEntryV1{
		object: object,
		table:  table,
	})

	return table
}

// RegisterGlobals installs the `dodder` global carrying list() (RFC-0008
// §3.1). The blob FFI is registered separately by the command since it needs
// repo access.
func (binding *ListTransformV1) RegisterGlobals() {
	dodderTable := binding.vm.NewTable()
	binding.vm.SetField(
		dodderTable,
		"list",
		binding.vm.NewFunction(binding.luaList),
	)
	binding.vm.SetGlobal("dodder", dodderTable)
}

// IsHandle reports whether value is the list handle produced by
// dodder.list(), for validating the script's return value (RFC-0008 §3.4).
func (binding *ListTransformV1) IsHandle(value lua.LValue) bool {
	table, ok := value.(*lua.LTable)
	return ok && table == binding.handle
}

// Objects reads the script's mutations back off every non-removed entry via
// FromLuaTableTransformV1 and returns the output object set in input order
// (added objects last, in add order).
func (binding *ListTransformV1) Objects() (
	objects []*sku.Transacted,
	err error,
) {
	for index := range binding.entries {
		entry := &binding.entries[index]

		if entry.removed {
			continue
		}

		if _, err = FromLuaTableTransformV1(
			entry.object,
			binding.vm.LState,
			entry.table,
		); err != nil {
			err = errors.Wrapf(
				err,
				"object %s write-back",
				entry.object.GetObjectId(),
			)
			return objects, err
		}

		objects = append(objects, entry.object)
	}

	return objects, err
}

func (binding *ListTransformV1) Repool() {
	for _, repool := range binding.repools {
		repool()
	}
}

// luaList implements dodder.list(): every call returns the same handle.
func (binding *ListTransformV1) luaList(luaState *lua.LState) int {
	luaState.Push(binding.handle)
	return 1
}

// luaEach implements list:each(): returns an iterator over the non-removed
// per-object tables, suitable for `for object in list:each() do ... end`.
// Objects added mid-iteration are visited too.
func (binding *ListTransformV1) luaEach(luaState *lua.LState) int {
	index := 0

	iterator := binding.vm.NewFunction(func(luaState *lua.LState) int {
		for index < len(binding.entries) {
			entry := &binding.entries[index]
			index++

			if entry.removed {
				continue
			}

			luaState.Push(entry.table.Transacted)
			return 1
		}

		luaState.Push(lua.LNil)
		return 1
	})

	luaState.Push(iterator)
	return 1
}

// luaRemove implements list:remove(object): drops the object from the output
// list (the transform equivalent of ObjectTransform's keep = false).
func (binding *ListTransformV1) luaRemove(luaState *lua.LState) int {
	table, ok := luaState.Get(2).(*lua.LTable)

	if !ok {
		luaState.RaiseError(
			"list:remove expects an object handle from this list",
		)
		return 0
	}

	index, ok := binding.handleToIndex[table]

	if !ok {
		luaState.RaiseError("list:remove: object handle not from this list")
		return 0
	}

	binding.entries[index].removed = true

	return 0
}

// luaAdd implements list:add(): creates a new zettel object absent from the
// input and returns its handle for mutation. The object id is left empty;
// allocation happens Go-side at plan build (RFC-0008 §3.3).
func (binding *ListTransformV1) luaAdd(luaState *lua.LState) int {
	object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned ownership transfers to the output list
	object.GetObjectIdMutable().SetGenre(genres.Zettel)

	table := binding.appendObject(object)

	luaState.Push(table.Transacted)
	return 1
}
