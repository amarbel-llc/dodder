//go:build test

package sku_lua

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// FromLuaTableTransformV1 additionally writes Typ back onto the object (the
// capability RFC-0006's hook write-back deliberately withholds; safe in the
// batch transform context per FDR-0024).
func TestFromLuaTableTransformV1WritesTypeBack(t1 *testing.T) {
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

	ToLuaTableV1(object, vm.LState, table)
	vm.LState.SetField(table.Transacted, "Typ", lua.LString("task2"))

	_, err = FromLuaTableTransformV1(object, vm.LState, table)
	t.AssertNoError(err)

	expected, expectedRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer expectedRepool()
	t.AssertNoError(expected.GetMetadataMutable().GetTypeMutable().SetType("task2"))

	t.AssertEqualStrings(
		expected.GetType().String(),
		object.GetType().String(),
	)
}

// End-to-end binding exercise: a script iterates the list, mutates a type
// and tags, removes an object, and adds a new one; the read-back reflects
// all of it and the script's return value is recognized as the handle.
func TestListTransformV1EndToEnd(t1 *testing.T) {
	t := ui.MakeT(t1)

	one, oneRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer oneRepool()
	t.AssertNoError(one.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(one.GetMetadataMutable().GetTypeMutable().SetType("task"))
	t.AssertNoError(one.GetMetadataMutable().AddTagString("keep"))
	t.AssertNoError(one.GetMetadataMutable().AddTagString("drop_me"))

	two, twoRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer twoRepool()
	t.AssertNoError(two.GetObjectIdMutable().Set("two/dos"))
	t.AssertNoError(two.GetMetadataMutable().GetTypeMutable().SetType("note"))

	objects := []*sku.Transacted{one, two}

	script := `
local list = dodder.list()

for object in list:each() do
  if object.Kennung == "one/uno" then
    object.Typ = "task2"
    object.Etiketten["drop_me"] = nil
  end

  if object.Kennung == "two/dos" then
    list:remove(object)
  end
end

local fresh = list:add()
fresh.Typ = "note"
fresh.Etiketten["brand_new"] = true

return list
`

	var binding *ListTransformV1

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV1(vm, objects)
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	t.AssertTrue(
		binding.IsHandle(vm.Top),
		"script return value should be the list handle",
	)

	outputs, err := binding.Objects()
	t.AssertNoError(err)

	t.AssertEqual(2, len(outputs))

	// survivor: type mutated, tag removed
	t.AssertEqualStrings("one/uno", outputs[0].GetObjectId().String())

	expected, expectedRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer expectedRepool()
	t.AssertNoError(expected.GetMetadataMutable().GetTypeMutable().SetType("task2"))
	t.AssertEqualStrings(
		expected.GetType().String(),
		outputs[0].GetType().String(),
	)

	survivorTags := make(map[string]bool)
	for tag := range outputs[0].GetMetadata().AllTags() {
		survivorTags[tag.String()] = true
	}

	t.AssertTrue(survivorTags["keep"], "tag keep should survive")
	t.AssertFalse(survivorTags["drop_me"], "tag drop_me should be removed")

	// added object: empty id (allocation happens at plan build), zettel
	// genre, scripted type and tag
	added := outputs[1]
	t.AssertEqualStrings("", added.GetObjectId().String())
	t.AssertEqual(genres.Zettel, genres.Make(added.GetGenre()))

	addedTags := make(map[string]bool)
	for tag := range added.GetMetadata().AllTags() {
		addedTags[tag.String()] = true
	}

	t.AssertTrue(addedTags["brand_new"], "added object should carry brand_new tag")
}

// A script assigning a malformed digest to the transform-only Blob field
// surfaces a wrapped parse error at read-back rather than committing junk.
func TestListTransformV1RejectsInvalidBlobDigest(t1 *testing.T) {
	t := ui.MakeT(t1)

	one, oneRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer oneRepool()
	t.AssertNoError(one.GetObjectIdMutable().Set("one/uno"))

	script := `
local list = dodder.list()

for object in list:each() do
  object.Blob = "not-a-valid-digest"
end

return list
`

	var binding *ListTransformV1

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV1(vm, []*sku.Transacted{one})
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

// list:remove rejects a table that is not an object handle from this list.
func TestListTransformV1RemoveRejectsForeignTable(t1 *testing.T) {
	t := ui.MakeT(t1)

	one, oneRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer oneRepool()
	t.AssertNoError(one.GetObjectIdMutable().Set("one/uno"))

	script := `
local list = dodder.list()
local ok, err = pcall(function() list:remove({}) end)
assert(not ok, "remove of a foreign table should raise")
return list
`

	var binding *ListTransformV1

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV1(vm, []*sku.Transacted{one})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	t.AssertTrue(binding.IsHandle(vm.Top), "script should still return the handle")

	outputs, err := binding.Objects()
	t.AssertNoError(err)
	t.AssertEqual(1, len(outputs))
}
