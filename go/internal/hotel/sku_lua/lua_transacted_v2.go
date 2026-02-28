package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/lua"
)

type LuaTableV2 struct {
	Transacted *lua.LTable

	// TODO transition to single Tags table with Tag objects that reflect
	// tag_paths.PathWithType
	Tags         *lua.LTable
	TagsImplicit *lua.LTable
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
}

func FromLuaTableV2(
	object *sku.Transacted,
	luaState *lua.LState,
	luaTable *LuaTableV2,
) (err error) {
	t := luaTable.Transacted

	genre := genres.MakeOrUnknown(luaState.GetField(t, "Genre").String())

	object.ObjectId.SetGenre(genre)
	id := luaState.GetField(t, "ObjectId").String()

	if id != "" {
		if err = object.ObjectId.Set(id); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	tags := luaState.GetField(t, "Tags")
	tagsTable, ok := tags.(*lua.LTable)

	if !ok {
		err = errors.ErrorWithStackf("expected table but got %T", tags)
		return err
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

	// TODO Description
	// TODO Type
	// TODO Tai
	// TODO Blob
	// TODO Cache

	return err
}
