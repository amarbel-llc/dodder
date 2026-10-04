package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type LuaTableV2 struct {
	Transacted *lua.LTable

	// TODO transition to single Tags table with Tag objects that reflect
	// tag_paths.PathWithType
	Tags         *lua.LTable
	TagsImplicit *lua.LTable
	Fields       *lua.LTable
}

func ToLuaTableV2(
	tg sku.TransactedGetter,
	luaState *lua.LState,
	luaTable *LuaTableV2,
) {
	object := tg.GetSku()

	luaState.SetField(
		luaTable.Transacted,
		"Genre",
		lua.LString(object.GetGenre().String()),
	)
	luaState.SetField(
		luaTable.Transacted,
		"ObjectId",
		lua.LString(object.GetObjectId().String()),
	)
	luaState.SetField(
		luaTable.Transacted,
		"Type",
		lua.LString(object.GetType().String()),
	)

	tags := luaTable.Tags

	for tag := range object.GetMetadata().AllTags() {
		luaState.SetField(tags, tag.String(), lua.LBool(true))
	}

	tags = luaTable.TagsImplicit

	for tag := range object.GetMetadata().GetIndex().GetImplicitTags().All() {
		luaState.SetField(tags, tag.String(), lua.LBool(true))
	}

	// Project the metadata index fields (name -> string value) as
	// object.Fields.<name>, mirroring ToLuaTableV1 (RFC 0006 Phase 1);
	// FromLuaTableV2 reads mutated values back.
	fieldsTable := luaTable.Fields

	for field := range object.GetMetadata().GetIndex().GetFields() {
		luaState.SetField(fieldsTable, field.Key, lua.LString(field.Value))
	}
}

// FromLuaTableV2 is the English-keyed counterpart of FromLuaTableV1: it writes
// genre, id, tags, and projected field values back onto object, and returns
// fieldsChanged when any projected field value was altered (RFC 0006 Phase 1
// field write-back). Like FromLuaTableV1 it deliberately withholds Type and
// Blob write-back (RFC 0006 Phase 2, #319); the batch transform context uses
// FromLuaTableTransformV2 for those.
func FromLuaTableV2(
	object *sku.Transacted,
	luaState *lua.LState,
	luaTable *LuaTableV2,
) (fieldsChanged bool, err error) {
	t := luaTable.Transacted

	genre := genres.MakeOrUnknown(luaState.GetField(t, "Genre").String())

	object.GetObjectIdMutable().SetGenre(genre)
	id := luaState.GetField(t, "ObjectId").String()

	if id != "" {
		if err = object.GetObjectIdMutable().Set(id); err != nil {
			err = errors.Wrap(err)
			return fieldsChanged, err
		}
	}

	tags := luaState.GetField(t, "Tags")
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

	// TODO Description
	// TODO Type — retyping from a hook: RFC 0006 Phase 2 (currently forbidden), #319
	// TODO Tai
	// TODO Blob — hook direct blob mutation: RFC 0006 Phase 2, #319
	// TODO Cache

	return fieldsChanged, err
}
