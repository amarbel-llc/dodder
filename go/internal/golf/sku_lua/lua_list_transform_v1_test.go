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
    -- assigning false must remove like assigning nil does (the two
    -- natural Lua set-removal idioms); ForEach visits false values,
    -- so the write-back filters them
    object.Etiketten["drop_me"] = false
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

// The list binding projects each object's tai read-only, in three shapes a
// version-aware transform needs: the canonical string, a fixed-width key
// that sorts lexicographically in tai order (Lua numbers would lose the
// attosecond part), and the local calendar date.
func TestListTransformV1ProjectsTai(t1 *testing.T) {
	t := ui.MakeT(t1)

	earlier, earlierRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer earlierRepool()
	t.AssertNoError(earlier.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(
		earlier.GetMetadataMutable().GetTaiMutable().SetFromRFC3339(
			"2022-08-11T12:00:00Z",
		),
	)

	later, laterRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer laterRepool()
	t.AssertNoError(later.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(
		later.GetMetadataMutable().GetTaiMutable().SetFromRFC3339(
			"2023-01-02T12:00:00Z",
		),
	)

	script := `
local list = dodder.list()
local seen = {}

for object in list:each() do
  seen[#seen + 1] = object
end

assert(seen[1].Tai == "` + earlier.GetTai().String() + `", "Tai: " .. tostring(seen[1].Tai))
assert(seen[1].TaiDate == "2022-08-11", "TaiDate: " .. tostring(seen[1].TaiDate))
assert(seen[2].TaiDate == "2023-01-02", "TaiDate: " .. tostring(seen[2].TaiDate))
assert(seen[1].TaiSortKey < seen[2].TaiSortKey, "TaiSortKey must sort in tai order")

return list
`

	var binding *ListTransformV1

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV1(vm, []*sku.Transacted{earlier, later})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	t.AssertTrue(binding.IsHandle(vm.Top), "script should return the handle")

	outputs, err := binding.Objects()
	t.AssertNoError(err)
	t.AssertEqual(2, len(outputs))
}

// Transform scripts can read and write an object's description
// (Bezeichnung) and add metadata object references (References, an array of
// object id strings; write-back is additive). take4 uses both to create a new
// !task that points at an existing note.
func TestListTransformV1DescriptionAndReferences(t1 *testing.T) {
	t := ui.MakeT(t1)

	note, noteRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer noteRepool()
	t.AssertNoError(note.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(note.GetMetadataMutable().GetDescriptionMutable().Set("the note"))

	script := `
local list = dodder.list()
local note

for object in list:each() do
  note = object
end

assert(note.Bezeichnung == "the note", "Bezeichnung: " .. tostring(note.Bezeichnung))
assert(#note.References == 0, "References should start empty")

local task = list:add()
task.Typ = "task"
task.Bezeichnung = note.Bezeichnung
task.References[#task.References + 1] = note.Kennung

return list
`

	var binding *ListTransformV1

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV1(vm, []*sku.Transacted{note})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	t.AssertTrue(binding.IsHandle(vm.Top), "script should return the handle")

	outputs, err := binding.Objects()
	t.AssertNoError(err)
	t.AssertEqual(2, len(outputs))

	task := outputs[1]

	t.AssertEqualStrings("the note", task.GetMetadata().GetDescription().String())

	var references []string

	for reference := range task.GetMetadata().AllReferencedObjects() {
		references = append(references, reference.String())
	}

	t.AssertEqual(1, len(references))
	t.AssertEqualStrings("one/uno", references[0])
}

// Assigning Bezeichnung on an object that already has a description replaces
// it; Description.Set alone would append ("the note renamed").
func TestListTransformV1DescriptionReplacesExisting(t1 *testing.T) {
	t := ui.MakeT(t1)

	note, noteRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer noteRepool()
	t.AssertNoError(note.GetObjectIdMutable().Set("one/uno"))
	t.AssertNoError(note.GetMetadataMutable().GetDescriptionMutable().Set("the note"))

	script := `
local list = dodder.list()

for object in list:each() do
  object.Bezeichnung = "renamed"
end

return list
`

	var binding *ListTransformV1

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV1(vm, []*sku.Transacted{note})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	_, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	outputs, err := binding.Objects()
	t.AssertNoError(err)
	t.AssertEqual(1, len(outputs))
	t.AssertEqualStrings("renamed", outputs[0].GetMetadata().GetDescription().String())
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

// RemovedObjects returns exactly the removed input objects with the script's
// pre-removal mutations applied (here a drop-class tag), and omits an object
// the script added and then removed.
func TestListTransformV1RemovedObjects(t1 *testing.T) {
	t := ui.MakeT(t1)

	one, oneRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer oneRepool()
	t.AssertNoError(one.GetObjectIdMutable().Set("one/uno"))

	two, twoRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer twoRepool()
	t.AssertNoError(two.GetObjectIdMutable().Set("two/dos"))

	script := `
local list = dodder.list()

for object in list:each() do
  if object.Kennung == "two/dos" then
    object.Etiketten["zz-dropped-test"] = true
    list:remove(object)
  end
end

local scratch = list:add()
list:remove(scratch)

return list
`

	var binding *ListTransformV1

	vmPool, err := (&lua.VMPoolBuilder{}).WithScript(
		script,
	).WithApply(func(vm *lua.VM) error {
		binding = MakeListTransformV1(vm, []*sku.Transacted{one, two})
		binding.RegisterGlobals()
		return nil
	}).Build()
	t.AssertNoError(err)

	_, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()
	defer binding.Repool()

	outputs, err := binding.Objects()
	t.AssertNoError(err)
	t.AssertEqual(1, len(outputs))
	t.AssertEqualStrings("one/uno", outputs[0].GetObjectId().String())

	removed, err := binding.RemovedObjects()
	t.AssertNoError(err)
	t.AssertEqual(1, len(removed))
	t.AssertEqualStrings("two/dos", removed[0].GetObjectId().String())

	removedTags := make(map[string]bool)
	for tag := range removed[0].GetMetadata().AllTags() {
		removedTags[tag.String()] = true
	}

	t.AssertTrue(
		removedTags["zz-dropped-test"],
		"removed object should carry the tag set before removal",
	)
}
