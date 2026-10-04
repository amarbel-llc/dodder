package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// FDR-0024 / RFC-0008: English-keyed write-back for the inventory-list
// transform plugin mechanism (#407). The read-side projection is
// ToLuaTableV2 plus the transform-only keys ListTransformV2 adds; the list
// handle lives in lua_list_transform_v2.go.

// FromLuaTableTransformV2 is the English-keyed counterpart of
// FromLuaTableTransformV1: genre, id, tags, and fields write-back as in
// FromLuaTableV2, plus Type, Blob, Description, and References. Those extra
// write-backs are safe only in the list-transform context -- a single-pass
// batch operation running before any commit begins -- and stay withheld from
// the commit-hook write-back FromLuaTableV2 (issue #319, RFC-0006 Phase 2).
func FromLuaTableTransformV2(
	object *sku.Transacted,
	luaState *lua.LState,
	luaTable *LuaTableV2,
) (fieldsChanged bool, err error) {
	transacted := luaTable.Transacted

	genre := genres.MakeOrUnknown(
		luaState.GetField(transacted, "Genre").String(),
	)

	object.GetObjectIdMutable().SetGenre(genre)
	id := luaState.GetField(transacted, "ObjectId").String()

	if id != "" {
		if err = object.GetObjectIdMutable().Set(id); err != nil {
			err = errors.Wrap(err)
			return fieldsChanged, err
		}
	}

	typeString := luaState.GetField(transacted, "Type").String()

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

	if err = writeBlobDigestBackV2(object, luaState, transacted); err != nil {
		return fieldsChanged, err
	}

	if err = writeDescriptionBackV2(object, luaState, transacted); err != nil {
		return fieldsChanged, err
	}

	if err = writeReferencesBack(object, luaState, transacted); err != nil {
		return fieldsChanged, err
	}

	if err = writeTransformTagsBackV2(object, luaState, transacted); err != nil {
		return fieldsChanged, err
	}

	fieldsChanged = writeFieldsBack(object, luaTable.Fields)

	return fieldsChanged, err
}

// writeBlobDigestBackV2 applies the transform-only Blob field. An absent key
// leaves the digest alone; an empty string clears it.
func writeBlobDigestBackV2(
	object *sku.Transacted,
	luaState *lua.LState,
	transacted *lua.LTable,
) (err error) {
	blobValue := luaState.GetField(transacted, "Blob")

	if blobValue == lua.LNil {
		return err
	}

	blobString := blobValue.String()

	if blobString == object.GetBlobDigest().String() {
		return err
	}

	blobDigestMutable := object.GetMetadataMutable().GetBlobDigestMutable()

	if blobString == "" {
		blobDigestMutable.Reset()
		return err
	}

	if err = blobDigestMutable.Set(blobString); err != nil {
		err = errors.Wrapf(err, "invalid Blob digest %q", blobString)
		return err
	}

	return err
}

func writeDescriptionBackV2(
	object *sku.Transacted,
	luaState *lua.LState,
	transacted *lua.LTable,
) (err error) {
	value := luaState.GetField(transacted, "Description")

	if value == lua.LNil {
		return err
	}

	description := value.String()

	if description == object.GetMetadata().GetDescription().String() {
		return err
	}

	descriptionMutable := object.GetMetadataMutable().GetDescriptionMutable()

	// Description.Set appends to an existing value (flag semantics), so
	// reset first: a script assignment replaces the description.
	descriptionMutable.Reset()

	if err = descriptionMutable.Set(description); err != nil {
		err = errors.Wrapf(err, "invalid Description %q", description)
		return err
	}

	return err
}

func writeTransformTagsBackV2(
	object *sku.Transacted,
	luaState *lua.LState,
	transacted *lua.LTable,
) (err error) {
	tags := luaState.GetField(transacted, "Tags")
	tagsTable, ok := tags.(*lua.LTable)

	if !ok {
		err = errors.ErrorWithStackf("expected table but got %T", tags)
		return err
	}

	object.GetMetadataMutable().ResetTags()

	tagsTable.ForEach(
		func(key, value lua.LValue) {
			if err != nil {
				return
			}

			// `= false` is the other natural Lua set-removal idiom besides
			// `= nil` (which ForEach never even visits); treat it as absent
			// rather than re-adding the tag
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

	return err
}
