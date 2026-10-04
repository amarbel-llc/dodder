//go:build test

package sku_lua

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// V2 parity with V1 (#407): the metadata index fields are projected as
// object.Fields.<name> and a mutated value is written back, with the change
// reported.
func TestLuaTableV2ProjectsAndWritesFieldsBack(t1 *testing.T) {
	t := ui.MakeT(t1)

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	table, tableRepool := MakeLuaTablePoolV2(vm).GetWithRepool()
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

	ToLuaTableV2(object, vm.LState, table)

	fieldsTable, ok := vm.LState.GetField(
		table.Transacted,
		"Fields",
	).(*lua.LTable)
	t.AssertTrue(ok, "Fields table should be attached to Transacted")
	t.AssertEqualStrings(
		"todo",
		vm.LState.GetField(fieldsTable, "status").String(),
	)

	fieldsChanged, err := FromLuaTableV2(object, vm.LState, table)
	t.AssertNoError(err)
	t.AssertFalse(fieldsChanged, "an untouched Fields table must not report a change")

	vm.LState.SetField(table.Fields, "status", lua.LString("done"))

	fieldsChanged, err = FromLuaTableV2(object, vm.LState, table)
	t.AssertNoError(err)
	t.AssertTrue(fieldsChanged, "a mutated field should report fieldsChanged")

	got := make(map[string]string)
	for field := range object.GetMetadata().GetIndex().GetFields() {
		got[field.Key] = field.Value
	}

	t.AssertEqualStrings("done", got["status"])
	t.AssertEqualStrings("p1", got["priority"])
}

// The V2 pool clears the projected Fields table on repool, so a table
// borrowed for a fresh object never carries the prior object's fields.
func TestLuaTablePoolV2ClearsFieldsOnRepool(t1 *testing.T) {
	t := ui.MakeT(t1)

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	tablePool := MakeLuaTablePoolV2(vm)

	object, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	t.AssertNoError(object.GetObjectIdMutable().Set("one/uno"))
	object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(
		fields.Field{
			Type:  fields.TypeUserData,
			Key:   "status",
			Value: "done",
		},
	)

	table, tableRepool := tablePool.GetWithRepool()
	ToLuaTableV2(object, vm.LState, table)
	t.AssertTrue(
		countLuaTableEntries(vm.LState, table.Fields) > 0,
		"Fields should be populated after projecting an object with fields",
	)

	tableRepool()

	table2, table2Repool := tablePool.GetWithRepool()
	defer table2Repool()

	t.AssertEqual(0, countLuaTableEntries(vm.LState, table2.Fields))
}

// The hook-safe FromLuaTableV2 withholds Type write-back (#319), exactly as
// FromLuaTableV1 does; only FromLuaTableTransformV2 retypes.
func TestFromLuaTableV2WithholdsTypeWriteBack(t1 *testing.T) {
	t := ui.MakeT(t1)

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript("return {}").Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	table, tableRepool := MakeLuaTablePoolV2(vm).GetWithRepool()
	defer tableRepool()

	object, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	t.AssertNoError(object.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(object.GetMetadataMutable().GetTypeMutable().SetType("task"))

	original := object.GetType().String()

	ToLuaTableV2(object, vm.LState, table)
	vm.LState.SetField(table.Transacted, "Type", lua.LString("task2"))

	_, err = FromLuaTableV2(object, vm.LState, table)
	t.AssertNoError(err)
	t.AssertEqualStrings(original, object.GetType().String())

	_, err = FromLuaTableTransformV2(object, vm.LState, table)
	t.AssertNoError(err)

	expected, expectedRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer expectedRepool()
	t.AssertNoError(expected.GetMetadataMutable().GetTypeMutable().SetType("task2"))

	t.AssertEqualStrings(expected.GetType().String(), object.GetType().String())
}

// End-to-end V2 binding exercise with English keys: mutate type, tags, and
// description; read the transform-only Tai keys; remove one object; add one
// that references a survivor.
func TestListTransformV2EndToEnd(t1 *testing.T) {
	t := ui.MakeT(t1)

	one, oneRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer oneRepool()
	t.AssertNoError(one.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(one.GetMetadataMutable().GetTypeMutable().SetType("task"))
	t.AssertNoError(one.GetMetadataMutable().AddTagString("keep"))
	t.AssertNoError(one.GetMetadataMutable().AddTagString("drop_me"))
	t.AssertNoError(one.GetMetadataMutable().GetDescriptionMutable().Set("the note"))
	t.AssertNoError(
		one.GetMetadataMutable().GetTaiMutable().SetFromRFC3339(
			"2022-08-11T12:00:00Z",
		),
	)

	two, twoRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer twoRepool()
	t.AssertNoError(two.GetObjectIdMutable().Set("two/dos"))
	t.AssertNoError(two.GetMetadataMutable().GetTypeMutable().SetType("note"))

	script := `
assert(dodder.list_v2() == dodder.list_v2(), "list_v2 must return one handle")

local list = dodder.list_v2()
local survivor

for object in list:each() do
  assert(object.Kennung == nil, "V2 must not project the German keys")
  assert(object.Genre == "Zettel", "Genre: " .. tostring(object.Genre))

  if object.ObjectId == "one/uno" then
    survivor = object
    assert(object.Description == "the note", "Description: " .. tostring(object.Description))
    assert(object.TaiDate == "2022-08-11", "TaiDate: " .. tostring(object.TaiDate))
    assert(#object.References == 0, "References should start empty")
    object.Type = "task2"
    object.Description = "renamed"
    object.Tags["drop_me"] = false
  end

  if object.ObjectId == "two/dos" then
    list:remove(object)
  end
end

local fresh = list:add()
fresh.Type = "note"
fresh.Tags["brand_new"] = true
fresh.References[#fresh.References + 1] = survivor.ObjectId

return list
`

	var binding *ListTransformV2

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV2(vm, []*sku.Transacted{one, two})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	t.AssertTrue(binding.WasRequested(), "script called dodder.list_v2()")
	t.AssertTrue(
		binding.IsHandle(vm.Top),
		"script return value should be the list handle",
	)

	outputs, err := binding.Objects()
	t.AssertNoError(err)
	t.AssertEqual(2, len(outputs))

	survivor := outputs[0]
	t.AssertEqualStrings("one/uno", survivor.GetObjectId().String())
	t.AssertEqualStrings("renamed", survivor.GetMetadata().GetDescription().String())

	expected, expectedRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer expectedRepool()
	t.AssertNoError(expected.GetMetadataMutable().GetTypeMutable().SetType("task2"))
	t.AssertEqualStrings(expected.GetType().String(), survivor.GetType().String())

	survivorTags := make(map[string]bool)
	for tag := range survivor.GetMetadata().AllTags() {
		survivorTags[tag.String()] = true
	}

	t.AssertTrue(survivorTags["keep"], "tag keep should survive")
	t.AssertFalse(survivorTags["drop_me"], "tag drop_me should be removed")

	added := outputs[1]
	t.AssertEqualStrings("", added.GetObjectId().String())
	t.AssertEqual(genres.Zettel, genres.Make(added.GetGenre()))

	addedTags := make(map[string]bool)
	for tag := range added.GetMetadata().AllTags() {
		addedTags[tag.String()] = true
	}

	t.AssertTrue(addedTags["brand_new"], "added object should carry brand_new tag")

	var references []string

	for reference := range added.GetMetadata().AllReferencedObjects() {
		references = append(references, reference.String())
	}

	t.AssertEqual(1, len(references))
	t.AssertEqualStrings("one/uno", references[0])

	removed, err := binding.RemovedObjects()
	t.AssertNoError(err)
	t.AssertEqual(1, len(removed))
	t.AssertEqualStrings("two/dos", removed[0].GetObjectId().String())
}

// Projection is lazy: a script that never calls dodder.list_v2() leaves the
// binding unrequested and yields no output objects.
func TestListTransformV2UnrequestedProjectsNothing(t1 *testing.T) {
	t := ui.MakeT(t1)

	one, oneRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer oneRepool()
	t.AssertNoError(one.GetObjectIdMutable().Set("one/uno"))

	var binding *ListTransformV2

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		"return {}",
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV2(vm, []*sku.Transacted{one})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	t.AssertFalse(binding.WasRequested(), "script never called dodder.list_v2()")
	t.AssertFalse(binding.IsHandle(vm.Top), "a plain table is not the handle")

	outputs, err := binding.Objects()
	t.AssertNoError(err)
	t.AssertEqual(0, len(outputs))
}

// A malformed digest assigned to the transform-only Blob field surfaces a
// wrapped parse error at read-back.
func TestListTransformV2RejectsInvalidBlobDigest(t1 *testing.T) {
	t := ui.MakeT(t1)

	one, oneRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer oneRepool()
	t.AssertNoError(one.GetObjectIdMutable().Set("one/uno"))

	script := `
local list = dodder.list_v2()

for object in list:each() do
  object.Blob = "not-a-valid-digest"
end

return list
`

	var binding *ListTransformV2

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV2(vm, []*sku.Transacted{one})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	t.AssertTrue(binding.IsHandle(vm.Top), "script should return the handle")

	_, err = binding.Objects()
	t.AssertError(err)
}
