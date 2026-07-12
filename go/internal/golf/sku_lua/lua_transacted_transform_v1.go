package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// PROTOTYPE (FDR-0024 / RFC-0008): list-in/list-out projection and
// write-back for the inventory-list transform plugin mechanism. This is
// exploratory code to validate the design, not a finished implementation --
// see docs/features/0024-inventory-list-transform-plugins.md and
// docs/rfcs/0008-inventory-list-transform-plugin-api.md.

// ToLuaArrayV1 projects objects into a Lua array table (1-indexed) of
// per-object tables, reusing ToLuaTableV1's per-object read-side projection
// unchanged. Returns the array plus the individual *LuaTableV1 handles (same
// order) -- the caller must retain these to later read mutations back via
// FromLuaTableTransformV1, since the array itself only exposes each
// element's merged Transacted table, not the Fields handle write-back needs
// directly.
func ToLuaArrayV1(
	vm *lua.VM,
	tablePool LuaTablePoolV1,
	objects []*sku.Transacted,
) (array *lua.LTable, tables []*LuaTableV1, repoolAll func()) {
	array = vm.NewTable()
	tables = make([]*LuaTableV1, 0, len(objects))
	repools := make([]func(), 0, len(objects))

	for i, object := range objects {
		table, repool := tablePool.GetWithRepool() //repool:owned
		repools = append(repools, repool)

		ToLuaTableV1(object, vm.LState, table)

		array.RawSetInt(i+1, table.Transacted)
		tables = append(tables, table)
	}

	repoolAll = func() {
		for _, repool := range repools {
			repool()
		}
	}

	return array, tables, repoolAll
}

// FromLuaTableTransformV1 mirrors FromLuaTableV1 (genre, id, tags, fields
// write-back) but additionally writes back Typ/Type. This is safe in the
// list-transform context -- a single-pass batch operation running before
// any commit begins -- unlike RFC-0006's live-commit hook context, whose
// FromLuaTableV1 deliberately withholds type write-back (issue #319,
// gated on RFC-0006 Phase 2) because that binding runs under a
// re-entrancy guard protecting against nested commits. No such nesting
// risk exists here, so this is intentionally a separate function rather
// than a modification to FromLuaTableV1.
func FromLuaTableTransformV1(
	object *sku.Transacted,
	luaState *lua.LState,
	luaTable *LuaTableV1,
) (fieldsChanged bool, err error) {
	transacted := luaTable.Transacted

	genre := genres.MakeOrUnknown(
		luaState.GetField(transacted, "Gattung").String(),
	)

	object.GetObjectIdMutable().SetGenre(genre)
	id := luaState.GetField(transacted, "Kennung").String()

	if id != "" {
		if err = object.GetObjectIdMutable().Set(id); err != nil {
			err = errors.Wrap(err)
			return fieldsChanged, err
		}
	}

	typeString := luaState.GetField(transacted, "Typ").String()

	if typeString != "" {
		var typeStruct ids.TypeStruct

		if err = typeStruct.Set(typeString); err != nil {
			err = errors.Wrap(err)
			return fieldsChanged, err
		}

		object.GetMetadataMutable().GetTypeMutable().ResetWithType(
			typeStruct.ToType(),
		)
	}

	tags := luaState.GetField(transacted, "Etiketten")
	tagsTable, ok := tags.(*lua.LTable)

	if !ok {
		err = errors.ErrorWithStackf("expected table but got %T", tags)
		return fieldsChanged, err
	}

	object.GetMetadataMutable().ResetTags()

	tagsTable.ForEach(
		func(key, value lua.LValue) {
			var tag ids.TagStruct

			if err = tag.Set(key.String()); err != nil {
				err = errors.Wrap(err)
				panic(err)
			}

			errors.PanicIfError(object.GetMetadataMutable().AddTagPtr(tag))
		},
	)

	fieldsChanged = writeFieldsBack(object, luaTable.Fields)

	return fieldsChanged, err
}
