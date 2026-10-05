package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// ListTransformV2 is the English-keyed counterpart of ListTransformV1 (#407):
// the Go-side backing for the opt-in `dodder.list_v2()` binding of the
// inventory-list transform plugin (FDR-0024 / RFC-0008 §3). Objects are
// projected via ToLuaTableV2 (Genre/ObjectId/Type/Tags/Fields) plus the
// transform-only Blob, Description, References, and read-only Tai keys.
//
// Projection is lazy: nothing is projected until the script first calls
// dodder.list_v2(), so a script on the V1 `dodder.list()` binding pays
// nothing for this one being registered beside it.
type ListTransformV2 struct {
	vm        *lua.VM
	tablePool LuaTablePoolV2

	objects   []*sku.Transacted
	requested bool

	handle        *lua.LTable
	entries       []listTransformEntryV2
	handleToIndex map[*lua.LTable]int

	// skipsHooks holds the objects whose script set SkipHooks = true,
	// recorded at read-back.
	skipsHooks map[*sku.Transacted]struct{}

	repools []func()
}

type listTransformEntryV2 struct {
	object  *sku.Transacted
	table   *LuaTableV2
	removed bool
}

// MakeListTransformV2 builds the list handle table exposing
// each()/remove()/add() over objects. Register the result via
// RegisterGlobals before the script runs; call Repool when done with the VM.
func MakeListTransformV2(
	vm *lua.VM,
	objects []*sku.Transacted,
) (binding *ListTransformV2) {
	binding = &ListTransformV2{
		vm:            vm,
		tablePool:     MakeLuaTablePoolV2(vm),
		objects:       objects,
		handle:        vm.NewTable(),
		handleToIndex: make(map[*lua.LTable]int, len(objects)),
		skipsHooks:    make(map[*sku.Transacted]struct{}),
		repools:       make([]func(), 0, len(objects)),
	}

	vm.SetField(binding.handle, "each", vm.NewFunction(binding.luaEach))
	vm.SetField(binding.handle, "remove", vm.NewFunction(binding.luaRemove))
	vm.SetField(binding.handle, "add", vm.NewFunction(binding.luaAdd))

	return binding
}

func (binding *ListTransformV2) appendObject(
	object *sku.Transacted,
) (table *LuaTableV2) {
	table, repool := binding.tablePool.GetWithRepool() //repool:owned
	binding.repools = append(binding.repools, repool)

	ToLuaTableV2(object, binding.vm.LState, table)

	// Transform-only projections, deliberately NOT part of ToLuaTableV2:
	// hook and tag-filter scripts must not see a blob mutation surface
	// (RFC-0006 Phase 2 gate, issue #319). FromLuaTableTransformV2 reads
	// Blob, Description, and References back; the Tai keys are read-only.
	binding.vm.SetField(
		table.Transacted,
		"Blob",
		lua.LString(object.GetBlobDigest().String()),
	)

	projectTaiReadOnly(binding.vm, table.Transacted, object)

	binding.vm.SetField(
		table.Transacted,
		"Description",
		lua.LString(object.GetMetadata().GetDescription().String()),
	)

	references := binding.vm.NewTable()

	for reference := range object.GetMetadata().AllReferencedObjects() {
		references.Append(lua.LString(reference.String()))
	}

	binding.vm.SetField(table.Transacted, "References", references)

	// Transform-only, writable: SkipHooks = true commits this object without
	// its lua hook stages (see SkipsHooks).
	binding.vm.SetField(table.Transacted, "SkipHooks", lua.LFalse)

	binding.handleToIndex[table.Transacted] = len(binding.entries)
	binding.entries = append(binding.entries, listTransformEntryV2{
		object: object,
		table:  table,
	})

	return table
}

// RegisterGlobals installs list_v2() on the `dodder` global, creating the
// global when it does not exist yet. To offer both bindings, call
// ListTransformV1.RegisterGlobals first: it replaces the global outright.
func (binding *ListTransformV2) RegisterGlobals() {
	dodderTable, ok := binding.vm.GetGlobal("dodder").(*lua.LTable)

	if !ok {
		dodderTable = binding.vm.NewTable()
		binding.vm.SetGlobal("dodder", dodderTable)
	}

	binding.vm.SetField(
		dodderTable,
		"list_v2",
		binding.vm.NewFunction(binding.luaList),
	)
}

// IsHandle reports whether value is the list handle produced by
// dodder.list_v2(), for validating the script's return value (RFC-0008 §3.4).
func (binding *ListTransformV2) IsHandle(value lua.LValue) bool {
	table, ok := value.(*lua.LTable)
	return ok && table == binding.handle
}

// WasRequested reports whether the script called dodder.list_v2() at all.
func (binding *ListTransformV2) WasRequested() bool {
	return binding.requested
}

// Objects reads the script's mutations back off every non-removed entry via
// FromLuaTableTransformV2 and returns the output object set in input order
// (added objects last, in add order).
func (binding *ListTransformV2) Objects() (
	objects []*sku.Transacted,
	err error,
) {
	return binding.writeBackEntries(false)
}

// RemovedObjects returns the input objects the script list:remove()d, in
// input order, with the same write-back applied (so tags set just before
// removal survive). Objects added and then removed are skipped: they never
// had an object id.
func (binding *ListTransformV2) RemovedObjects() (
	objects []*sku.Transacted,
	err error,
) {
	var removed []*sku.Transacted

	if removed, err = binding.writeBackEntries(true); err != nil {
		return objects, err
	}

	for _, object := range removed {
		if object.GetObjectId().String() == "" {
			continue
		}

		objects = append(objects, object)
	}

	return objects, err
}

func (binding *ListTransformV2) writeBackEntries(
	removed bool,
) (objects []*sku.Transacted, err error) {
	for index := range binding.entries {
		entry := &binding.entries[index]

		if entry.removed != removed {
			continue
		}

		if _, err = FromLuaTableTransformV2(
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

		if binding.vm.GetField(entry.table.Transacted, "SkipHooks") == lua.LTrue {
			binding.skipsHooks[entry.object] = struct{}{}
		} else {
			delete(binding.skipsHooks, entry.object)
		}

		objects = append(objects, entry.object)
	}

	return objects, err
}

// SkipsHooks reports whether the script set `SkipHooks = true` on object (one
// of the objects Objects() returned): the consumer commits it without its lua
// hook stages -- on_new, on_pre_commit, on_commit_fields -- while reference
// discovery, fields projection, and validation still run. Meaningful only
// after Objects().
func (binding *ListTransformV2) SkipsHooks(object *sku.Transacted) bool {
	_, skips := binding.skipsHooks[object]
	return skips
}

func (binding *ListTransformV2) Repool() {
	for _, repool := range binding.repools {
		repool()
	}
}

// luaList implements dodder.list_v2(): the first call projects the input
// objects; every call returns the same handle.
func (binding *ListTransformV2) luaList(luaState *lua.LState) int {
	if !binding.requested {
		binding.requested = true

		for _, object := range binding.objects {
			binding.appendObject(object)
		}
	}

	luaState.Push(binding.handle)
	return 1
}

// luaEach implements list:each(): returns an iterator over the non-removed
// per-object tables, suitable for `for object in list:each() do ... end`.
// Objects added mid-iteration are visited too.
func (binding *ListTransformV2) luaEach(luaState *lua.LState) int {
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
// list.
func (binding *ListTransformV2) luaRemove(luaState *lua.LState) int {
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
func (binding *ListTransformV2) luaAdd(luaState *lua.LState) int {
	object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned ownership transfers to the output list
	object.GetObjectIdMutable().SetGenre(genres.Zettel)

	table := binding.appendObject(object)

	luaState.Push(table.Transacted)
	return 1
}
