//go:build test

package sku_lua

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// An object whose metadata index carries projected fields exposes them to the
// hook table as kinder.Fields.<name> (read-only projection, FDR safe-half).
func TestToLuaTableV1ProjectsFields(t1 *testing.T) {
	t := ui.MakeT(t1)

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	tablePool := MakeLuaTablePoolV1(vm)

	table, tableRepool := tablePool.GetWithRepool()
	defer tableRepool()

	object, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := object.GetMetadataMutable()
	t.AssertNoError(object.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(metadata.GetTypeMutable().SetType("task"))

	fieldsMutable := metadata.GetIndexMutable().GetFieldsMutable()
	fieldsMutable.Append(fields.Field{
		Type:  fields.TypeUserData,
		Key:   "status",
		Value: "done",
	})
	fieldsMutable.Append(fields.Field{
		Type:  fields.TypeUserData,
		Key:   "priority",
		Value: "p1",
	})

	ToLuaTableV1(object, vm.LState, table)

	// reachable directly on the projected Fields table
	t.AssertEqualStrings(
		"done",
		vm.LState.GetField(table.Fields, "status").String(),
	)
	t.AssertEqualStrings(
		"p1",
		vm.LState.GetField(table.Fields, "priority").String(),
	)

	// reachable as kinder.Fields.status through the attached Transacted table
	fieldsTable, ok := vm.LState.GetField(
		table.Transacted,
		"Fields",
	).(*lua.LTable)
	t.AssertTrue(ok, "Fields table should be attached to Transacted")
	t.AssertEqualStrings(
		"done",
		vm.LState.GetField(fieldsTable, "status").String(),
	)
}

// A hook that mutates kinder.Fields.<name> has its new value written back onto
// the object's projected index field, and FromLuaTableV1 reports the change
// (RFC 0006 Phase 1 field write-back).
func TestFromLuaTableV1WritesFieldsBack(t1 *testing.T) {
	t := ui.MakeT(t1)

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	tablePool := MakeLuaTablePoolV1(vm)

	table, tableRepool := tablePool.GetWithRepool()
	defer tableRepool()

	object, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := object.GetMetadataMutable()
	t.AssertNoError(object.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(metadata.GetTypeMutable().SetType("task"))

	fieldsMutable := metadata.GetIndexMutable().GetFieldsMutable()
	fieldsMutable.Append(fields.Field{
		Type:  fields.TypeUserData,
		Key:   "status",
		Value: "todo",
	})
	fieldsMutable.Append(fields.Field{
		Type:  fields.TypeUserData,
		Key:   "priority",
		Value: "p1",
	})

	// project, then simulate a hook mutating one field and leaving the other
	ToLuaTableV1(object, vm.LState, table)
	vm.LState.SetField(table.Fields, "status", lua.LString("done"))

	fieldsChanged, err := FromLuaTableV1(object, vm.LState, table)
	t.AssertNoError(err)
	t.AssertTrue(fieldsChanged, "a mutated field should report fieldsChanged")

	got := make(map[string]string)
	for field := range object.GetMetadata().GetIndex().GetFields() {
		got[field.Key] = field.Value
	}

	t.AssertEqualStrings("done", got["status"])
	t.AssertEqualStrings("p1", got["priority"])
}

// The pool clears the projected Fields table on repool, so a table borrowed
// for a fresh object never carries the prior object's fields. Without this
// reset a commit hook would observe a previous object's kinder.Fields.<name>
// (RFC 0006 leak). Locks the reset path in lua_transacted_v1_pool.go.
func TestLuaTablePoolV1ClearsFieldsOnRepool(t1 *testing.T) {
	t := ui.MakeT(t1)

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	tablePool := MakeLuaTablePoolV1(vm)

	object, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := object.GetMetadataMutable()
	t.AssertNoError(object.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(metadata.GetTypeMutable().SetType("task"))

	fieldsMutable := metadata.GetIndexMutable().GetFieldsMutable()
	fieldsMutable.Append(fields.Field{
		Type:  fields.TypeUserData,
		Key:   "status",
		Value: "done",
	})
	fieldsMutable.Append(fields.Field{
		Type:  fields.TypeUserData,
		Key:   "priority",
		Value: "p1",
	})

	// first borrow: project an object that carries fields, populating Fields
	table, tableRepool := tablePool.GetWithRepool()
	ToLuaTableV1(object, vm.LState, table)
	t.AssertTrue(
		countLuaTableEntries(vm.LState, table.Fields) > 0,
		"Fields should be populated after projecting an object with fields",
	)

	// repool runs the pool's reset, which clears Fields
	tableRepool()

	// second borrow must not leak the prior object's projected fields
	table2, table2Repool := tablePool.GetWithRepool()
	defer table2Repool()

	t.AssertEqual(
		0,
		countLuaTableEntries(vm.LState, table2.Fields),
	)
}

func countLuaTableEntries(luaState *lua.LState, table *lua.LTable) (count int) {
	luaState.ForEach(table, func(_, _ lua.LValue) {
		count++
	})

	return count
}

// A hook that leaves kinder.Fields untouched reports fieldsChanged=false, so
// the commit pipeline skips the bounded write-back pass entirely.
func TestFromLuaTableV1NoFieldChangeReportsFalse(t1 *testing.T) {
	t := ui.MakeT(t1)

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	tablePool := MakeLuaTablePoolV1(vm)

	table, tableRepool := tablePool.GetWithRepool()
	defer tableRepool()

	object, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := object.GetMetadataMutable()
	t.AssertNoError(object.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(metadata.GetTypeMutable().SetType("task"))

	fieldsMutable := metadata.GetIndexMutable().GetFieldsMutable()
	fieldsMutable.Append(fields.Field{
		Type:  fields.TypeUserData,
		Key:   "status",
		Value: "todo",
	})

	ToLuaTableV1(object, vm.LState, table)

	fieldsChanged, err := FromLuaTableV1(object, vm.LState, table)
	t.AssertNoError(err)
	t.AssertFalse(fieldsChanged, "an untouched Fields table must not report a change")
}
