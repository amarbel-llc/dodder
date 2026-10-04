package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// FDR-0024 / RFC-0008: write-back for the inventory-list transform plugin
// mechanism. The read-side projection is ToLuaTableV1, unchanged; the list
// handle lives in lua_list_transform_v1.go.

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

	if err = writeGenreAndIdBack(object, luaState, transacted); err != nil {
		return fieldsChanged, err
	}

	if typeString := luaStringFieldOrEmpty(luaState, transacted, "Typ"); typeString != "" {
		var typeStruct ids.TypeStruct

		if err = typeStruct.Set(typeString); err != nil {
			err = errors.Wrap(err)
			return fieldsChanged, err
		}

		object.GetMetadataMutable().GetTypeMutable().ResetWithType(
			typeStruct.ToType(),
		)
	}

	// Blob write-back: the "Blob" field is projected only by the transform
	// list binding (absent in the hook context, where GetField yields nil).
	// An empty string clears the blob digest.
	if blobValue := luaState.GetField(transacted, "Blob"); blobValue != lua.LNil {
		blobString := blobValue.String()

		if blobString != object.GetBlobDigest().String() {
			blobDigestMutable := object.GetMetadataMutable().GetBlobDigestMutable()

			if blobString == "" {
				blobDigestMutable.Reset()
			} else if err = blobDigestMutable.Set(blobString); err != nil {
				err = errors.Wrapf(err, "invalid Blob digest %q", blobString)
				return fieldsChanged, err
			}
		}
	}

	if err = writeDescriptionBack(object, luaState, transacted); err != nil {
		return fieldsChanged, err
	}

	if err = writeReferencesBack(object, luaState, transacted); err != nil {
		return fieldsChanged, err
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
			if err != nil {
				return
			}

			// `= false` is the other natural Lua set-removal idiom
			// besides `= nil` (which ForEach never even visits); treat
			// it as absent rather than re-adding the tag
			if boolValue, isBool := value.(lua.LBool); isBool && !bool(boolValue) {
				return
			}

			var tag ids.TagStruct

			if tagErr := tag.Set(key.String()); tagErr != nil {
				err = errors.Wrapf(tagErr, "invalid tag %q", key.String())
				return
			}

			if addErr := object.GetMetadataMutable().AddTagPtr(tag); addErr != nil {
				err = errors.Wrapf(addErr, "adding tag %q", key.String())
				return
			}
		},
	)

	if err != nil {
		return fieldsChanged, err
	}

	fieldsChanged = writeFieldsBack(object, luaTable.Fields)

	return fieldsChanged, err
}

// writeDescriptionBack applies the transform-only Bezeichnung field (projected
// by the list binding; absent in hook contexts, where GetField yields nil).
func writeDescriptionBack(
	object *sku.Transacted,
	luaState *lua.LState,
	transacted *lua.LTable,
) (err error) {
	value := luaState.GetField(transacted, "Bezeichnung")

	if value == lua.LNil {
		return err
	}

	description := value.String()

	if description == object.GetMetadata().GetDescription().String() {
		return err
	}

	// Reset first: Description.Set appends to a non-empty, differing value
	// (its box-scanner accumulation semantics), but Bezeichnung replaces.
	descriptionMutable := object.GetMetadataMutable().GetDescriptionMutable()
	descriptionMutable.Reset()

	if err = descriptionMutable.Set(description); err != nil {
		err = errors.Wrapf(err, "invalid Bezeichnung %q", description)
		return err
	}

	return err
}

// writeReferencesBack adds every object id listed in the transform-only
// References array that the object does not already reference. Additive
// only: removing a reference is not expressible.
func writeReferencesBack(
	object *sku.Transacted,
	luaState *lua.LState,
	transacted *lua.LTable,
) (err error) {
	value := luaState.GetField(transacted, "References")
	referencesTable, ok := value.(*lua.LTable)

	if !ok {
		return err
	}

	existing := make(map[string]struct{})

	for reference := range object.GetMetadata().AllReferencedObjects() {
		existing[reference.String()] = struct{}{}
	}

	for index := 1; index <= referencesTable.Len(); index++ {
		idString := referencesTable.RawGetInt(index).String()

		if _, present := existing[idString]; present {
			continue
		}

		var reference ids.SeqId

		if err = reference.Set(idString); err != nil {
			err = errors.Wrapf(err, "invalid reference %q", idString)
			return err
		}

		if err = object.GetMetadataMutable().AddReference(reference); err != nil {
			err = errors.Wrapf(err, "adding reference %q", idString)
			return err
		}

		existing[idString] = struct{}{}
	}

	return err
}
