//go:build test

package sku_lua

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
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
